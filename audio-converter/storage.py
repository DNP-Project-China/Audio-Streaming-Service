import os
import boto3
from uuid import UUID
from botocore.client import Config
from botocore.exceptions import ClientError


class S3Client:
    def __init__(self):
        self.endpoint_url = os.environ["S3_ENDPOINT"]
        self.access_key = os.environ["S3_ACCESS_KEY"]
        self.secret_key = os.environ["S3_SECRET_KEY"]
        self.region = os.environ["S3_REGION"]
        self.bucket = os.environ["S3_BUCKET"]

        self.client = boto3.client(
            's3',
            endpoint_url=self.endpoint_url,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
            region_name=self.region,
            config=Config(signature_version='s3v4')
        )

    def download_file(self, s3_key: str, local_path: str) -> bool:
        try:
            print(f"Downloading {s3_key} from bucket {self.bucket}...")
            self.client.download_file(self.bucket, s3_key, local_path)
            print("Download completed.")
            return True
        except ClientError as e:
            print(f"Error occurred while downloading {s3_key}: {e}")
            return False

    def upload_hls_folder(self, track_id: UUID, local_folder: str) -> bool:
        try:
            for filename in os.listdir(local_folder):
                local_filepath = os.path.join(local_folder, filename)

                if not os.path.isfile(local_filepath):
                    continue

                s3_key = f"hls/{track_id}/{filename}"
                content_type = "application/vnd.apple.mpegurl" if filename.endswith(
                    ".m3u8") else "video/MP2T"

                print(f"Uploading {filename} -> {s3_key}...")
                self.client.upload_file(
                    local_filepath,
                    self.bucket,
                    s3_key,
                    ExtraArgs={'ContentType': content_type}
                )

            print(f"Track {track_id} uploaded successfully to S3!")
            return True
        except ClientError as e:
            print(f"Error occurred while uploading HLS files: {e}")
            return False
