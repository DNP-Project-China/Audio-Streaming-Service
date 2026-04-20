import os
import uuid
import asyncpg
import asyncio
import logging
import redis.asyncio as redis
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from aiokafka import AIOKafkaProducer
import json
from contextlib import asynccontextmanager
from dotenv import load_dotenv

load_dotenv()

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

producer = None
db_pool = None
redis_client = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    global producer, db_pool, redis_client
    
    kafka_brokers = os.getenv("KAFKA_BROKERS", "localhost:9094")
    producer = AIOKafkaProducer(bootstrap_servers=kafka_brokers)

    max_retries = 5
    for attempt in range(max_retries):
        try:
            await producer.start()
            logger.info("Successfully connected to Kafka")
            break
        except Exception as e:
            logger.warning(f"Failed to connect to Kafka (attempt {attempt + 1}/{max_retries}): {e}")
            if attempt == max_retries - 1:
                logger.error("Max retries reached. Could not connect to Kafka. Raising exception...")
                await producer.stop() 
                raise e
            logger.info("Retrying Kafka connection in 3 seconds...")
            await asyncio.sleep(3)
    
    db_user = os.getenv("POSTGRES_USER", "postgres")
    db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
    db_host = os.getenv("POSTGRES_HOST", "localhost")
    db_port = os.getenv("POSTGRES_PORT", "5432")
    db_name = os.getenv("POSTGRES_DB", "core")
    
    db_url = f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}"
    
    db_pool = await asyncpg.create_pool(db_url)

    redis_host = os.getenv("REDIS_HOST", "localhost")
    redis_port = os.getenv("REDIS_PORT", "6379")
    redis_url = f"redis://{redis_host}:{redis_port}"

    redis_client = redis.from_url(redis_url, decode_responses=True)

    listener_task = asyncio.create_task(listen_expired_sessions())
    
    yield

    listener_task.cancel()

    try:
        await listener_task
    except asyncio.CancelledError:
        pass
    
    if producer:
        await producer.stop()
    if db_pool:
        await db_pool.close()

    await redis_client.close()    


async def listen_expired_sessions():
    await redis_client.config_set('notify-keyspace-events', 'Ex')

    pubsub = redis_client.pubsub()
    await pubsub.psubscribe('__keyevent@0__:expired')

    try:
        async for message in pubsub.listen():
            try:
                if message['type'] == 'pmessage':
                    expired_key = message['data']

                    if expired_key.startswith("session:"):
                        parts = expired_key.split(":")

                        if len(parts) == 3:
                            _, user_session, track_id = parts

                            event_data = {
                                "track_id": track_id,
                                "user_session": user_session,
                                "status": "stopped_by_timeout"
                            }

                            kafka_msg = json.dumps(event_data).encode("utf-8")
                            await producer.send_and_wait("playback-events", kafka_msg)

            except Exception as e:
                logger.error(f"Error processing item: {e}")

    except asyncio.CancelledError:
        logger.info("Listener task cancelled.")

    except Exception as e:
        logger.critical(f"Critical Redis Listener error: {e}")


app = FastAPI(lifespan=lifespan)

@app.get("/play/{track_id}")
async def play_track(track_id: uuid.UUID):
    try:
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

    session_key = f"session:{playload.user_session}:{playload.track_id}"
    await redis_client.set(session_key, str(playload.track_id), ex=20)
    
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

    session_key = f"session:{playload.user_session}:{playload.track_id}"
    await redis_client.delete(session_key)
    
    return {
        "status": "ok"
    }

