import os
import uuid
import asyncpg
import asyncio
import logging
import json
import time
import redis.asyncio as redis
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from aiokafka import AIOKafkaProducer
from contextlib import asynccontextmanager
from dotenv import load_dotenv

load_dotenv()

logging.basicConfig(level=logging.INFO,
                    format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

producer = None
db_pool = None
redis_client = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Manages global app state (startup/shutdown).
    Initializes connections to Kafka, PostgreSQL, and Redis,
    and spawns the background polling task for session expirations.
    """
    global producer, db_pool, redis_client

    kafka_brokers = os.getenv("KAFKA_BROKERS", "localhost:9094")
    producer = AIOKafkaProducer(bootstrap_servers=kafka_brokers)

    # Retry in case Kafka is still booting up
    max_retries = 5
    for attempt in range(max_retries):
        try:
            await producer.start()
            logger.info("Successfully connected to Kafka")
            break
        except Exception as e:
            logger.warning(
                f"Failed to connect to Kafka (attempt {attempt + 1}/{max_retries}): {e}")
            if attempt == max_retries - 1:
                logger.error(
                    "Max retries reached. Could not connect to Kafka. Raising exception...")
                await producer.stop()
                raise e
            logger.info("Retrying Kafka connection in 3 seconds...")
            await asyncio.sleep(3)

    # Connect to PostgreSQL DB for track metadata access
    db_user = os.getenv("POSTGRES_USER", "postgres")
    db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
    db_host = os.getenv("POSTGRES_HOST", "localhost")
    db_port = os.getenv("POSTGRES_PORT", "5432")
    db_name = os.getenv("POSTGRES_DB", "core")

    db_url = f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}"

    # Connection pool to reduce RTT
    db_pool = await asyncpg.create_pool(db_url)

    # Connect to Redis for real-time playback tracking
    redis_host = os.getenv("REDIS_HOST", "localhost")
    redis_port = os.getenv("REDIS_PORT", "6379")
    redis_url = f"redis://{redis_host}:{redis_port}"

    redis_client = redis.from_url(redis_url, decode_responses=True)

    # Background worker that guarantees session expiration processing
    poll_task = asyncio.create_task(poll_expired_sessions())

    yield

    # --- Shutdown Phase ---
    poll_task.cancel()
    try:
        await poll_task
    except asyncio.CancelledError:
        pass

    if producer:
        await producer.stop()
    if db_pool:
        await db_pool.close()
    if redis_client:
        await redis_client.close()


async def poll_expired_sessions():
    """
    Background task: periodically polls a Redis Sorted Set (ZSET) for expired sessions.
    ZSET guarantees that expired sessions are processed even if the Playback API restarts.
    """
    try:
        while True:
            now = int(time.time())
            # Fetch members with a score (expiration timestamp) less than or equal to current time
            expired_sessions = await redis_client.zrange('active_sessions', 0, now, byscore=True)

            for session_str in expired_sessions:
                try:
                    session_data = json.loads(session_str)

                    event_data = {
                        "track_id": session_data["track_id"],
                        "user_session": session_data["user_session"],
                        "status": "stopped_by_timeout"
                    }
                    kafka_msg = json.dumps(event_data).encode("utf-8")
                    await producer.send_and_wait("playback-events", kafka_msg)

                    # Remove successfully processed session from the ZSET
                    await redis_client.zrem('active_sessions', session_str)

                except json.JSONDecodeError as e:
                    logger.error(
                        f"Failed to decode session JSON '{session_str}': {e}")

                    # Remove corrupted data to prevent infinite retry loops
                    await redis_client.zrem('active_sessions', session_str)

                except Exception as e:
                    logger.error(
                        f"Error processing session '{session_str}': {e}")

            # Sleep briefly before polling again to avoid spamming Redis
            await asyncio.sleep(5)

    except asyncio.CancelledError:
        # Expected during app shutdown
        logger.info("Polling task cancelled.")

    except Exception as e:
        logger.critical(f"Critical Redis Polling error: {e}")


app = FastAPI(lifespan=lifespan)


@app.get("/play/{track_id}")
async def play_track(track_id: uuid.UUID):
    """
    Fetches the HLS playlist URL for streaming a specific track.
    """
    try:
        async with db_pool.acquire() as connection:
            query = "SELECT status, hls_playlist_key FROM tracks WHERE id = $1"
            row = await connection.fetchrow(query, track_id)

        if not row:
            raise HTTPException(status_code=404, detail="Track not found")

        # Prevent playback until processing is fully complete
        if row['status'] == "ready":
            base_url = os.getenv("S3_PUBLIC_BASE_URL",
                                 "https://s3.example.com/bucket")
            playlist_url = f"{base_url}/{row['hls_playlist_key']}"

            # Emit 'started' event for analytics
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
            raise HTTPException(
                status_code=400, detail=f"Track is not ready yet. Status: {row['status']}")

    except asyncpg.PostgresError as e:
        # Secure logging: hide DB structure details from the client
        logger.error(f"Database error when fetching track {track_id}: {e}")
        raise HTTPException(status_code=500, detail="Internal database error")


class PlaybackEvent(BaseModel):
    """
    Links a specific user's session to the track they are streaming.
    """
    track_id: uuid.UUID
    user_session: str


@app.post("/ping")
async def ping_event(payload: PlaybackEvent):
    """
    Heartbeat endpoint. 
    Clients must call this periodically to keep the audio session alive.
    """
    event_data = {
        "track_id": str(payload.track_id),
        "user_session": payload.user_session,
        "status": "playing"
    }
    message = json.dumps(event_data).encode("utf-8")
    await producer.send_and_wait("playback-events", message)

    # Calculate expiration time (current time + 20 seconds)
    expiration_time = int(time.time()) + 20

    # Store data as a JSON string to avoid ValueError during parsing
    # sort_keys=True ensures the string is deterministic for identical payloads
    session_dict = {
        "user_session": payload.user_session,
        "track_id": str(payload.track_id)
    }
    session_json = json.dumps(session_dict, sort_keys=True)

    # Upsert the session in the active_sessions ZSET.
    # If the session exists, its score (expiration time) is extended.
    await redis_client.zadd('active_sessions', {session_json: expiration_time})

    return {
        "status": "ok"
    }


@app.post("/stop")
async def stop_event(payload: PlaybackEvent):
    """
    Handles explicit playback termination (user presses pause/stop).
    """
    event_data = {
        "track_id": str(payload.track_id),
        "user_session": payload.user_session,
        "status": "stopped"
    }
    message = json.dumps(event_data).encode("utf-8")
    await producer.send_and_wait("playback-events", message)

    # Remove the session from the ZSET to prevent false 'stopped_by_timeout' events
    session_dict = {
        "user_session": payload.user_session,
        "track_id": str(payload.track_id)
    }
    session_json = json.dumps(session_dict, sort_keys=True)
    await redis_client.zrem('active_sessions', session_json)

    return {
        "status": "ok"
    }
