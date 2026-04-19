import os
import json
import tempfile
from uuid import UUID
from dotenv import load_dotenv
from kafka import KafkaConsumer

from storage import S3Client
from converter import convert_audio_to_hls
from database import get_track_status, update_track_status

load_dotenv()

def main():
    s3_client = S3Client()

    consumer = KafkaConsumer(
        'transcode-jobs',
        bootstrap_servers=os.environ["KAFKA_BROKERS"],
        group_id='transcoder-group',
        value_deserializer=lambda x: json.loads(x.decode('utf-8'))
    )

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

        update_track_status(track_id, 'ready')
        print(f"[SUCCESS] {job_id}: Task completed successfully ---\n")

if __name__ == "__main__":
    main()