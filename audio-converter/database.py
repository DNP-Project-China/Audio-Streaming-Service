import os
import psycopg2


def get_db_connection():
    return psycopg2.connect(
        dbname=os.environ["POSTGRES_DB"],
        user=os.environ["POSTGRES_USER"],
        password=os.environ["POSTGRES_PASSWORD"],
        host=os.environ["POSTGRES_HOST"],
        port=os.environ["POSTGRES_PORT"]
    )


def get_track_status(track_id: int) -> str:
    status = None
    try:
        conn = get_db_connection()
        cur = conn.cursor()

        cur.execute("SELECT status FROM tracks WHERE id = %s", (track_id,))
        result = cur.fetchone()
        if result:
            status = result[0]

        cur.close()
        conn.close()
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")

    return status


def update_track_status(track_id: int, status: str):
    try:
        conn = get_db_connection()
        cur = conn.cursor()

        cur.execute("UPDATE tracks SET status = %s WHERE id = %s",
                    (status, track_id))
        conn.commit()

        cur.close()
        conn.close()
        print(f"[DB] Track {track_id} changed to '{status}'")
    except Exception as e:
        print(f"[DB ERROR] {track_id}: {e}")
