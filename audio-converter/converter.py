import subprocess
import os


def convert_audio_to_hls(input_file: str, output_dir: str) -> bool:
    os.makedirs(output_dir, exist_ok=True)

    segment_pattern = os.path.join(output_dir, 'segment_%04d.ts')
    playlist_file = os.path.join(output_dir, 'master.m3u8')

    command = [
        "ffmpeg",
        "-i", input_file,
        "-vn",
        "-c:a", "aac",
        "-b:a", "320k",
        "-f", "hls",
        "-hls_time", "6",
        "-hls_playlist_type", "vod",
        "-hls_segment_filename", segment_pattern,
        playlist_file
    ]

    result = subprocess.run(command, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Error converting audio: {result.stderr}")
        return False
    return True
