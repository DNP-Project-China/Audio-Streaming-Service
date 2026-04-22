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
    brokers = os.environ["KAFKA_BROKERS"]
    retry_delay_seconds = int(os.environ.get("KAFKA_RETRY_DELAY_SECONDS", "5"))

    while True:
        try:
            print(f"Connecting to Kafka brokers: {brokers}", flush=True)
            return KafkaConsumer(
                topic,
                bootstrap_servers=brokers,
                group_id='transcoder-group',
                value_deserializer=lambda x: json.loads(x.decode('utf-8')),
                auto_offset_reset='earliest',
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

    for message in consumer:
        job = message.value
        job_id = job.get('job_id')
        track_id = UUID(job.get('track_id'))
        path = job.get('path')
        priority = job.get('priority', 0)

        print(f"\n--- New Task: Job ID = {job_id}, Track ID = {track_id}, Priority = {priority} ---")

        current_status = get_track_status(track_id)
        if current_status in ['processing', 'ready']:
            print(f"Track {track_id} already has status '{current_status}'. Skipping.")
            continue

        update_track_status(track_id, 'processing')

        with tempfile.TemporaryDirectory() as tmpdir:
            local_mp3 = os.path.join(tmpdir, "raw.mp3")
            hls_output_dir = os.path.join(tmpdir, "hls")

            if not s3_client.download_file(path, local_mp3):
                update_track_status(track_id, 'failed')
                print(f"[FAILED] {job_id}: Failed to download file")
                continue

            if not convert_audio_to_hls(local_mp3, hls_output_dir):
                update_track_status(track_id, 'failed')
                print(f"[FAILED] {job_id}: Failed to convert audio")
                continue

            if not s3_client.upload_hls_folder(track_id, hls_output_dir):
                update_track_status(track_id, 'failed')
                print(f"[FAILED] {job_id}: Failed to upload HLS files")
                continue

        hls_playlist_key = f"hls/{track_id}/master.m3u8" 
        update_track_ready(track_id, hls_playlist_key)
        print(f"[SUCCESS] {job_id}: Task completed successfully ---\n")

if __name__ == "__main__":
    main()