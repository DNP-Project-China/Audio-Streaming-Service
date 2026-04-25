"""
Storage interface for distributed object storage (S3 API compatible).
By offloading file storage to an external S3-compatible service,
this worker remains entirely stateless and can be horizontally scaled
without data consistency issues.
"""

import os
import boto3
from uuid import UUID
from botocore.client import Config
from botocore.exceptions import ClientError


class S3Client:
    """
    Client wrapper for S3 operations. 
    Handles connection configuration and provides fault-tolerant methods for network I/O.
    """

    def __init__(self):
        self.endpoint_url = os.environ["S3_ENDPOINT"]
        self.access_key = os.environ["S3_ACCESS_KEY"]
        self.secret_key = os.environ["S3_SECRET_KEY"]
        self.region = os.environ["S3_REGION"]
        self.bucket = os.environ["S3_BUCKET"]

        # Initializing the boto3 client with specific configurations for broader compatibility.
        # 's3v4' signature and 'path' addressing style ensure compatibility with various
        # S3-compatible backends.
        self.client = boto3.client(
            's3',
            endpoint_url=self.endpoint_url,
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
            region_name=self.region,
            config=Config(
                signature_version='s3v4',
                s3={'addressing_style': 'path'},
                request_checksum_calculation='when_required',
                response_checksum_validation='when_required',
            )
        )

    def download_file(self, s3_key: str, local_path: str) -> bool:
        """
        Fetches the raw audio payload from distributed storage to the worker's ephemeral local disk.
        Returns True if the network transfer is successful, False otherwise.
        """
        try:
            print(f"Downloading {s3_key} from bucket {self.bucket}...")
            self.client.download_file(self.bucket, s3_key, local_path)
            print("Download completed.")
            return True
        # Catching ClientError ensures the worker does not crash due to transient network issues,
        # missing buckets, or unauthorized access, 
        # allowing the main loop to fail the job gracefully.
        except ClientError as e:
            print(f"Error occurred while downloading {s3_key}: {e}")
            return False

    def upload_hls_folder(self, track_id: UUID, local_folder: str) -> bool:
        """
        Persists the transcoded HLS artifacts (segments and playlist) back to distributed storage.
        Sets the correct MIME types so downstream services
        (Playback API / Browsers) can stream the content directly.
        """
        try:
            for filename in os.listdir(local_folder):
                local_filepath = os.path.join(local_folder, filename)

                if not os.path.isfile(local_filepath):
                    continue

                s3_key = f"hls/{track_id}/{filename}"

                # Explicitly defining Content-Type is critical for HTTP streaming clients
                # to properly interpret the chunks and playlist files.
                content_type = "application/vnd.apple.mpegurl" if filename.endswith(
                    ".m3u8") else "video/MP2T"

                print(f"Uploading {filename} -> {s3_key}...")
                with open(local_filepath, "rb") as file_obj:
                    self.client.put_object(
                        Bucket=self.bucket,
                        Key=s3_key,
                        Body=file_obj,
                        ContentType=content_type,
                    )

            print(f"Track {track_id} uploaded successfully to S3!")
            return True

        # Fault tolerance: any network failure during the batch upload is caught,
        # allowing the orchestrator to mark the entire transcode job as 'failed'.
        except ClientError as e:
            print(f"Error occurred while uploading HLS files: {e}")
            return False
