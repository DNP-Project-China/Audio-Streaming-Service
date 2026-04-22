import os
import psycopg2
from uuid import UUID


def get_db_connection():
    return psycopg2.connect(
        dbname=os.environ["POSTGRES_DB"],
        user=os.environ["POSTGRES_USER"],
        password=os.environ["POSTGRES_PASSWORD"],
        host=os.environ["POSTGRES_HOST"],
        port=os.environ["POSTGRES_PORT"]
    )


def get_track_status(track_id: UUID) -> str:
    status = None
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT status FROM tracks WHERE id = %s", (str(track_id),))
                result = cur.fetchone()
                if result:
                    status = result[0]
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")

    return status


def update_track_status(track_id: UUID, status: str):
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "UPDATE tracks SET status = %s WHERE id = %s",
                    (status, str(track_id)),
                )
            conn.commit()
            
        print(f"[DB] Track {track_id} changed to '{status}'")
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")


def update_track_ready(track_id: UUID, hls_playlist_key: str):
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "UPDATE tracks SET status = 'ready', hls_playlist_key = %s WHERE id = %s",
                    (hls_playlist_key, str(track_id)),
                )
            conn.commit()
            
        print(f"[DB] Track {track_id} changed to 'ready' with key {hls_playlist_key}")
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
