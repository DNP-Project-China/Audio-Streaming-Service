import os
import uuid
import asyncpg
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from aiokafka import AIOKafkaProducer
import json
from contextlib import asynccontextmanager
from dotenv import load_dotenv

# Загружаем переменные из .env файла
load_dotenv()

producer = None
db_pool = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    global producer, db_pool
    
    kafka_brokers = os.getenv("KAFKA_BROKERS", "localhost:9094")
    producer = AIOKafkaProducer(bootstrap_servers=kafka_brokers)
    await producer.start()
    
    db_user = os.getenv("POSTGRES_USER", "postgres")
    db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
    db_host = os.getenv("POSTGRES_HOST", "localhost")
    db_port = os.getenv("POSTGRES_PORT", "5432")
    db_name = os.getenv("POSTGRES_DB", "core")
    
    db_url = f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}"
    
    db_pool = await asyncpg.create_pool(db_url)
    
    yield
    
    if producer:
        await producer.stop()
    if db_pool:
        await db_pool.close()

app = FastAPI(lifespan=lifespan)

@app.get("/play/{track_id}")
async def play_track(track_id: uuid.UUID):
    try:
        # Из пула берем одно готовое соединение
        async with db_pool.acquire() as connection:
            query = "SELECT status, hls_playlist_key FROM tracks WHERE id = $1"
            row = await connection.fetchrow(query, track_id)

        if not row:
            raise HTTPException(status_code=404, detail="Track not found")

        if row['status'] == "ready":
            base_url = os.getenv("S3_PUBLIC_BASE_URL", "https://s3.example.com/bucket/")
            playlist_url = f"{base_url}{row['hls_playlist_key']}"

            event_data = {
                "track_id": str(track_id),
                "status": "started"
            }

            message = json.dumps(event_data).encode("utf-8")

            await producer.send_and_wait("playback-events", message)

            return {
                "track_id": track_id, 
                "playlist_url": playlist_url
            }
        else:
            raise HTTPException(status_code=400, detail=f"Track is not ready yet. Status: {row['status']}")

    except asyncpg.PostgresError as e:
        raise HTTPException(status_code=500, detail=f"Database error: {str(e)}")
        

class PlaybackEvent(BaseModel):
    track_id: uuid.UUID
    user_session: str


@app.post("/ping")
async def ping_event(playload : PlaybackEvent):
    event_data = {
        "track_id": str(playload.track_id),
        "user_session": playload.user_session,
        "status": "playing"
    }

    message = json.dumps(event_data).encode("utf-8")

    await producer.send_and_wait("playback-events", message)
    
    return {
        "status": "ok"
    }

@app.post("/stop")
async def stop_event(playload : PlaybackEvent):
    event_data = {
        "track_id": str(playload.track_id),
        "user_session": playload.user_session,
        "status": "stopped"
    }

    message = json.dumps(event_data).encode("utf-8")

    await producer.send_and_wait("playback-events", message)
    
    return {
        "status": "ok"
    }

