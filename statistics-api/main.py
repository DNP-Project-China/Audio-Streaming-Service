import os
import json
import time
import asyncio
from datetime import datetime, timezone
from contextlib import asynccontextmanager
from typing import Literal, Optional
from uuid import UUID

import asyncpg
import redis.asyncio as redis
from aiokafka import AIOKafkaConsumer
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, ValidationError


load_dotenv()

db_pool = None
redis_client = None
consumer_task = None
flush_task = None
consumer_ready = False
UPDATE_TIME_RATE = 20
TOP_24_UPDATE_RATE = 24 * 60 * 60
TOP_TRACKS_LIMIT = 10
FLUSH_INTERVAL_SECONDS = 60
RETRY_DELAY_SECONDS = 10

# Validation (converter) of event from kafka 
class PlaybackEvent(BaseModel):
    track_id: UUID
    user_session: Optional[str] = None
    status: Literal["started", "playing", "stopped", "stopped_by_timeout"]
    ts: datetime

# Adding a timestamp for Redis sorted set
def now_utc_iso() -> str:
    return datetime.now(timezone.utc).isoformat()

# Main logic recording the events into Redis
async def process_event(event: PlaybackEvent) -> None:
    # Take the track ID and create a key for the sorted set
    track_id = str(event.track_id)
    key = f"online:track:{track_id}"
    top24key = f"top24:track:{track_id}"
    # In user pins the track (start playing) we add the session to the sorted set with the timestamp, and remove old sessions that are out of the UPDATE_TIME_RATE window and TOP_24_UPDATE_RATE window
    if event.status == "playing":
        if not event.user_session:
            return
        await redis_client.zadd(key, {event.user_session: event.ts.timestamp()})
        await redis_client.zremrangebyscore(key, "-inf", event.ts.timestamp() - UPDATE_TIME_RATE)
        
        await redis_client.zadd(top24key, {event.user_session: event.ts.timestamp()})
        await redis_client.zremrangebyscore(top24key, "-inf", event.ts.timestamp() - TOP_24_UPDATE_RATE)

    # If the user stopped the track, we remove the session from the sorted set
    elif event.status in ("stopped", "stopped_by_timeout"):
        if not event.user_session:
            return
        await redis_client.zrem(key, event.user_session)
    # If the track is started, we increment the play count in the sorted set and also create temporary recording for delta counting, which will be flushed to the database periodically
    elif event.status == "started":
        await redis_client.zincrby("plays:counter", 1, track_id)
        await redis_client.hincrby("plays:delta", track_id, 1)

    # ZSET stores presence information, HASH stores delta count of plays (in specified time)

# Consumer creation n
async def consumer_creation(kafka_brokers: str, kafka_topic: str, kafka_group: str) -> AIOKafkaConsumer:
    # Create an aiokafka consumer instance configured for our topic/group.
    # Returned consumer should be started by the caller with `await consumer.start()`.
    return AIOKafkaConsumer(
        kafka_topic,
        bootstrap_servers=kafka_brokers,
        group_id=kafka_group,
        enable_auto_commit=True,
        auto_offset_reset="latest",
    )

# Kafka consumer loop, which will consume events and process them, with error handling and reconnection logic
async def consume_playback_events(kafka_brokers: str, kafka_topic: str, kafka_group: str) -> None:
    global consumer_ready
    while True:
        local_consumer = None
        # We create a fresh consumer on each loop iteration to avoid re-using
        # a previously-started AIOKafkaConsumer (which can raise about double-start).
        try:
            local_consumer = await consumer_creation(kafka_brokers, kafka_topic, kafka_group)
            await local_consumer.start()
            consumer_ready = True
            await asyncio.sleep(RETRY_DELAY_SECONDS)
            print("statistics-api: kafka consumer started", flush=True)
            # Message processing loop: iterate over incoming messages and handle them.
            async for msg in local_consumer:
                try:
                    payload = json.loads(msg.value.decode("utf-8"))

                    # Adding a timestamp if it's not provided in the event, so we can use it in future
                    if "ts" not in payload:
                        payload["ts"] = now_utc_iso()
                    event = PlaybackEvent(**payload)
                    await process_event(event)
                    if event.status == "started":
                        print(f"statistics-api: play started for {event.track_id}", flush=True)

                # Error handling for message processing
                except (json.JSONDecodeError, ValidationError) as exc:
                    print(f"statistics-api: bad event skipped: {exc}", flush=True)
                except Exception as exc:
                    print(f"statistics-api: event processing error: {exc}", flush=True)
                    
        # Error handling for consumer connection and loop
        except asyncio.CancelledError:
            consumer_ready = False
            break
        except Exception as exc:
            print(f"statistics-api: consumer error: {exc}", flush=True)
            consumer_ready = False
            await asyncio.sleep(3)

        # Ensure the consumer is stopped on error/shutdown so resources are released
        finally:
            if local_consumer is not None:
                try:
                    await local_consumer.stop()
                    consumer_ready = False
                except Exception as exc:
                    print(f"statistics-api: consumer stop error: {exc}", flush=True)

# One-minute job for updating total_pays collums in database
async def flush_plays_once() -> None:
    data = await redis_client.hgetall("plays:delta")
    if not data:
        print("flush: plays:delta is empty, skipping", flush=True)
        return
    
    print(f"flush: processing {len(data)} tracks", flush=True)

    # updating database
    async with db_pool.acquire() as conn:
        async with conn.transaction():
            for track_id, cnt_str in data.items():
                cnt = int(cnt_str)
                print(f"flush: updating track {track_id} += {cnt}", flush=True)
                await conn.execute(
                    """
                    UPDATE tracks
                    SET total_plays = total_plays + $2
                    WHERE id = $1::uuid
                    """,
                    track_id,
                    cnt,
                )

    # Eliminate Redis delta data collection after flushing to the database, to avoid overcounting
    # Delete the keys we just processed. We call HGETALL again to collect current keys
    # and then HDEL them to avoid deleting entries added after this function started.
    # Read the hash atomically (await the coroutine) and extract keys
    hashes_dict = await redis_client.hgetall("plays:delta")
    hashes = list(hashes_dict.keys())
    if hashes:
        await redis_client.hdel("plays:delta", *hashes)
    print("flush: plays:delta cleared", flush=True)

    # Note: this simple approach may miss increments added concurrently during flush.
    # For stronger correctness consider RENAME-based swap or Lua script to atomically
    # move and clear the hash.

# Flush loop that runs flush_plays_once every FLUSH_INTERVAL_SECONDS seconds
async def flush_plays_loop() -> None:
    print("flush: loop started", flush=True)
    while True:
        try:
            await asyncio.sleep(FLUSH_INTERVAL_SECONDS)
            print("flush: tick", flush=True)
            await flush_plays_once()
        # Error handling for flush loop
        except asyncio.CancelledError:
            break
        except Exception as exc:
            print(f"flush error: {exc}", flush=True)

# Lifespan function for FastAPI
# Initialize the database pool, Redis client, and start the consumer and flush tasks on startup, and clean up on shutdown
@asynccontextmanager
async def lifespan(app: FastAPI):
    global db_pool, redis_client, consumer_task, flush_task

    # Redis connection 
    redis_host = os.getenv("REDIS_HOST", "localhost")
    redis_port = os.getenv("REDIS_PORT", "6379")
    redis_db = os.getenv("REDIS_DB", "0")
    redis_url = f"redis://{redis_host}:{redis_port}/{redis_db}"
    redis_client = redis.from_url(redis_url, decode_responses=True)

    # Database connection pool
    db_user = os.getenv("POSTGRES_USER", "postgres")
    db_pass = os.getenv("POSTGRES_PASSWORD", "postgres")
    db_host = os.getenv("POSTGRES_HOST", "localhost")
    db_port = os.getenv("POSTGRES_PORT", "5432")
    db_name = os.getenv("POSTGRES_DB", "core")
    db_url = f"postgresql://{db_user}:{db_pass}@{db_host}:{db_port}/{db_name}"
    db_pool = await asyncpg.create_pool(db_url)

    # Kafka consumer connection parameters
    kafka_brokers = os.getenv("KAFKA_BROKERS", "localhost:9094")
    kafka_topic = os.getenv("KAFKA_PLAYBACK_TOPIC", "playback-events")
    kafka_group = os.getenv("KAFKA_STATS_GROUP", "statistics-api-group")

    # Starting the consumer and flush tasks
    consumer_task = asyncio.create_task(consume_playback_events(kafka_brokers, kafka_topic, kafka_group))
    flush_task = asyncio.create_task(flush_plays_loop())
    print("statistics-api: consumer and flush tasks started", flush=True)

    # Separation of code for 2 parts:
    # Frist executes when the application starts
    # Second executes when the application is shutting down
    yield
    
    # Сlean up on shutdown: cancel tasks and close connections
    if consumer_task:
        consumer_task.cancel()
        try:
            await consumer_task
        except asyncio.CancelledError:
            pass

    if flush_task:
        flush_task.cancel()
        try:
            await flush_task
        except asyncio.CancelledError:
            pass

    if db_pool:
        await db_pool.close()
    if redis_client:
        await redis_client.close()


app = FastAPI(lifespan=lifespan)

# endpoint for dependences health check and therefore container health check, used by docker-compose
@app.get("/health")
async def health():
    try:
        # Redis ping
        if not await redis_client.ping():
            raise RuntimeError("redis not available")

        # DB check (use context manager so connection is released)
        async with db_pool.acquire() as conn:
            await conn.fetchval("SELECT 1")

        # Consumer should be ready (connected to Kafka)
        if not consumer_ready:
            raise RuntimeError("kafka consumer not ready")

        return {"status": "ok"}
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))

# main endpoint to get top tracks with their play counts and online listeners
@app.get("/stats")
async def stats():
    # Get top tracks by play count from Redis sorted set
    top = await redis_client.zrevrange("plays:counter", 0, TOP_TRACKS_LIMIT-1, withscores=True)
    # Fetch track info from PostgreSQL for the top tracks
    track_ids = [UUID(track_id) for track_id, _ in top]
    info_by_id = {}
    if track_ids:
        rows = await db_pool.fetch(
            """
            SELECT id, title, artist
            FROM tracks
            WHERE id = ANY($1::uuid[])
            """,
            track_ids,
        )
        # forming a dicktionary
        info_by_id = {str(row["id"]): {"title": row["title"], "artist": row["artist"]} for row in rows}
    # Forming the result with online listeners count for each track
    result = []
    for track_id, plays in top:
        online_key = f"online:track:{track_id}"
        top24key = f"top24:track:{track_id}"
        now_ts = int(time.time())
        # removing old sessions for online listeners and top 24h listeners to keep the sorted sets clean and accurate, then counting current online listeners and listeners in the last 24 hours
        await redis_client.zremrangebyscore(online_key, "-inf", now_ts - UPDATE_TIME_RATE)
        await redis_client.zremrangebyscore(top24key, "-inf", now_ts - TOP_24_UPDATE_RATE)
        online_now = await redis_client.zcard(online_key)
        last_24h = await redis_client.zcard(top24key)
        result.append(
            {
                "track_id": track_id,
                "title": info_by_id.get(track_id, {}).get("title"),
                "artist": info_by_id.get(track_id, {}).get("artist"),
                "total_plays": int(plays),
                "online_now": int(online_now),
                "last_24h": int(last_24h),
            }
        )
    # Returning the result
    return {"items": result, "total": len(result)}