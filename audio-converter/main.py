import os
import json
import tempfile
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
        track_id = job.get('track_id')
        s3_key = job.get('s3_key')

        print(f"\n--- New Task: Track ID = {track_id} ---")

        current_status = get_track_status(track_id)
        if current_status in ['processing', 'ready']:
            print(f"Track {track_id} already has status '{current_status}'. Skipping.")
            continue

        update_track_status(track_id, 'processing')

        with tempfile.TemporaryDirectory() as tmpdir:
            local_mp3 = os.path.join(tmpdir, "raw.mp3")
            hls_output_dir = os.path.join(tmpdir, "hls")

            if not s3_client.download_file(s3_key, local_mp3):
                update_track_status(track_id, 'failed')
                continue

            if not convert_audio_to_hls(local_mp3, hls_output_dir):
                update_track_status(track_id, 'failed')
                continue

            if not s3_client.upload_hls_folder(track_id, hls_output_dir):
                update_track_status(track_id, 'failed')
                continue

        update_track_status(track_id, 'ready')
        print(f"--- Task for track {track_id} completed successfully! ---\n")

if __name__ == "__main__":
    main()