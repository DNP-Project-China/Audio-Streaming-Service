"""
Database interaction functions for managing track conversion status in PostgreSQL.
"""
import os
import psycopg2
from uuid import UUID


def get_db_connection():
    """
    Establishes a connection to the PostgreSQL database using credentials from environment variables.
    Returns a connection object that can be used with context managers (with statement).
    """
    return psycopg2.connect(
        dbname=os.environ["POSTGRES_DB"],
        user=os.environ["POSTGRES_USER"],
        password=os.environ["POSTGRES_PASSWORD"],
        host=os.environ["POSTGRES_HOST"],
        port=os.environ["POSTGRES_PORT"]
    )


def get_track_status(track_id: UUID) -> str:
    """
    Retrieves the current status of a track from the database using its UUID.
    Returns the status as a string (e.g., 'pending', 'processing', 'ready', 'failed')
    or None if the track is not found.
    """
    status = None
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT status FROM tracks WHERE id = %s", (str(track_id),))
                result = cur.fetchone()
                if result:
                    status = result[0]

    # Catch any exceptions that occur during database operations and log them.
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")

    return status


def update_track_status(track_id: UUID, status: str) -> bool:
    """
    Updates the status of a track in the database.
    If setting status to 'processing', it will only update if the current status is 'pending' or 'failed'.
    This ensures that only one worker can claim a track for processing, preventing race conditions
    in a distributed environment.
    Returns True if the status was updated successfully,or False if the track was already claimed
    by another worker or if an error occurred.
    """
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                # Optimistic locking: Only update to 'processing' if current status is 'pending' or 'failed'.
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

                # Check how many rows were updated
                # If rows_updated is 0 when trying to set to 'processing', 
                # it means another worker has already claimed this track.
                rows_updated = cur.rowcount
            conn.commit()

        # If rows_updated is 0, it means the update was not applied (e.g., another worker claimed the track)
        if rows_updated > 0:
            print(f"[DB] Track {track_id} changed to '{status}'")
            return True
        else:
            print(
                f"[DB WARN] Task {track_id} skipped. Already taken by another worker.")
            return False

    # Catch any exceptions that occur during database operations and log them.
    # Return False to indicate the update was not successful.
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
        return False


def update_track_ready(track_id: UUID, hls_playlist_key: str) -> bool:
    """
    Updates the track's status to 'ready' and sets the HLS playlist key in the database.
    This should only be called after successful processing and uploading of HLS files.
    Returns True if the update was successful, or False if an error occurred.
    """
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "UPDATE tracks SET status = 'ready', hls_playlist_key = %s WHERE id = %s AND status = 'processing'",
                    (hls_playlist_key, str(track_id)),
                )
                # Check how many rows were updated.
                # If 0, it means the track was not in 'processing' status, 
                # which could indicate a logic error in the workflow.
                rows_updated = cur.rowcount
            conn.commit()

        # If rows_updated is 0, it means the update was not applied.
        if rows_updated > 0:
            print(
                f"[DB] Track {track_id} changed to 'ready' with key {hls_playlist_key}")
            return True
        else:
            print(
                f"[DB WARN] Track {track_id} update to 'ready' ignored (status was not processing)")
            return False

    # Catch any exceptions that occur during database operations and log them.
    # Return False to indicate the update was not successful.
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
        return False
