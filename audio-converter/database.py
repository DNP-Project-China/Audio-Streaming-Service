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


from uuid import UUID

def update_track_status(track_id: UUID, status: str) -> bool:
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                if status == 'processing':
                    cur.execute(
                        "UPDATE tracks SET status = %s WHERE id = %s AND status IN ('pending', 'failed')",
                        (status, str(track_id)),
                    )
                else:
                    cur.execute(
                        "UPDATE tracks SET status = %s WHERE id = %s",
                        (status, str(track_id)),
                    )
                
                rows_updated = cur.rowcount
            conn.commit()
            
        if rows_updated > 0:
            print(f"[DB] Track {track_id} changed to '{status}'")
            return True
        else:
            print(f"[DB WARN] Task {track_id} skipped. Already taken by another worker.")
            return False
            
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
        return False


def update_track_ready(track_id: UUID, hls_playlist_key: str) -> bool:
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "UPDATE tracks SET status = 'ready', hls_playlist_key = %s WHERE id = %s AND status = 'processing'",
                    (hls_playlist_key, str(track_id)),
                )
                rows_updated = cur.rowcount
            conn.commit()
            
        if rows_updated > 0:
            print(f"[DB] Track {track_id} changed to 'ready' with key {hls_playlist_key}")
            return True
        else:
            print(f"[DB WARN] Track {track_id} update to 'ready' ignored (status was not processing)")
            return False
            
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
        return False
