"""
Audio Converter Worker Service.
Acts as a stateless, horizontally scalable consumer in a distributed architecture.
Implements At-Least-Once delivery guarantees, concurrency control via Optimistic Locking,
and fault-tolerant processing for heavy CPU-bound tasks (FFmpeg transcoding).
"""

import os
import json
import tempfile
import time
from uuid import UUID
from dotenv import load_dotenv
from kafka import KafkaConsumer
from kafka.errors import NoBrokersAvailable

from storage import S3Client
from converter import convert_audio_to_hls
from database import get_track_status, update_track_status, update_track_ready

load_dotenv()


def create_consumer_with_retry(topic: str) -> KafkaConsumer:
    """
    Establishes a connection to the Kafka cluster with a naive retry mechanism.
    Provides fault tolerance during system initialization (e.g., in Docker Compose environments
    where message brokers might start slower than the consumer services).
    """
    brokers = os.environ["KAFKA_BROKERS"]
    retry_delay_seconds = int(os.environ.get("KAFKA_RETRY_DELAY_SECONDS", "5"))

    while True:
        try:
            print(f"Connecting to Kafka brokers: {brokers}", flush=True)
            return KafkaConsumer(
                topic,
                bootstrap_servers=brokers,
                # Consumer group allows horizontal scaling by partitioning messages among workers
                group_id='transcoder-group',
                value_deserializer=lambda x: json.loads(x.decode('utf-8')),
                auto_offset_reset='earliest',
                # Disabled auto-commit to implement manual offset management,
                # ensuring At-Least-Once delivery semantics in case of worker crashes.
                enable_auto_commit=False,
            )
        except NoBrokersAvailable:
            print(
                f"Kafka is not available yet at {brokers}. "
                f"Retrying in {retry_delay_seconds}s...",
                flush=True,
            )
            time.sleep(retry_delay_seconds)


def main():
    print("audio-converter: service starting", flush=True)
    s3_client = S3Client()

    consumer = create_consumer_with_retry('transcode-jobs')
    print("audio-converter: waiting for transcode-jobs messages", flush=True)

    # Polling loop: continuously consumes messages from the broker
    for message in consumer:
        job = message.value
        job_id = job.get('job_id')
        track_id = UUID(job.get('track_id'))
        path = job.get('path')
        priority = job.get('priority', 0)

        print(f"\n--- New Task: Job ID = {job_id}, Track ID = {track_id}, Priority = {priority} ---")

        current_status = get_track_status(track_id)
        
        # Idempotency check: Since we use At-Least-Once delivery, duplicates may occur.
        # If the track is fully processed, we acknowledge the message and skip.
        if current_status == 'ready':
            print(f"Track {track_id} already has status 'ready'. Skipping.")
            consumer.commit() 
            continue

        # Concurrency Control (Optimistic Locking):
        # Attempts to acquire an exclusive lock on the task via a conditional database UPDATE.
        # This prevents race conditions if multiple worker instances receive the same message.
        is_task_acquired = update_track_status(track_id, 'processing')
        
        if not is_task_acquired:
            print(f"Task {track_id} is locked by another instance. Skipping.")
            # Acknowledge the message since another worker is already handling it
            consumer.commit()
            continue

        try:
            # Ephemeral storage guarantees the worker remains completely stateless
            # and prevents disk space exhaustion over time.
            with tempfile.TemporaryDirectory() as tmpdir:
                local_mp3 = os.path.join(tmpdir, "raw.mp3")
                hls_output_dir = os.path.join(tmpdir, "hls")

                # Step 1: Fetch raw data from distributed object storage
                if not s3_client.download_file(path, local_mp3):
                    update_track_status(track_id, 'failed')
                    print(f"[FAILED] {job_id}: Failed to download file")
                    consumer.commit() # Commit offset on explicit application failure
                    continue

                # Step 2: CPU-intensive transcoding task
                if not convert_audio_to_hls(local_mp3, hls_output_dir):
                    update_track_status(track_id, 'failed')
                    print(f"[FAILED] {job_id}: Failed to convert audio")
                    consumer.commit()
                    continue

                # Step 3: Persist processed artifacts back to storage
                if not s3_client.upload_hls_folder(track_id, hls_output_dir):
                    update_track_status(track_id, 'failed')
                    print(f"[FAILED] {job_id}: Failed to upload HLS files")
                    consumer.commit()
                    continue

            # Commit phase: Update distributed state only after successful side-effects
            hls_playlist_key = f"hls/{track_id}/master.m3u8" 
            update_track_ready(track_id, hls_playlist_key)
            
            # Finalize task: Acknowledge successful processing to the Kafka broker
            consumer.commit()
            print(f"[SUCCESS] {job_id}: Task completed successfully ---\n")
            
        except Exception as e:
            # Critical Failure Handler:
            # If the process crashes (e.g., OOM during FFmpeg, unhandled exception),
            # consumer.commit() is intentionally bypassed. Kafka will reassign 
            # this message to another consumer in the group after a session timeout.
            print(f"[CRITICAL] Worker crashed on job {job_id}: {e}")
            
if __name__ == "__main__":
    main()