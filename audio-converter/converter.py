"""
Audio conversion functions for transcoding audio files to HLS format.
"""

import subprocess
import os


def convert_audio_to_hls(input_file: str, output_dir: str) -> bool:
    """
    Uses ffmpeg to convert an input audio file (e.g., MP3) into HLS format.
    The output consists of segmented .ts files and a master .m3u8 playlist.
    """
    os.makedirs(output_dir, exist_ok=True)

    segment_pattern = os.path.join(output_dir, 'segment_%04d.ts')
    playlist_file = os.path.join(output_dir, 'master.m3u8')

    # FFmpeg command breakdown:
    # -i: Input file
    # -vn: No video
    # -c:a: Audio codec (AAC)
    # -b:a: Audio bitrate (320 kbps)
    # -f hls: Output format is HLS
    # -hls_time: Duration of each segment (6 seconds)
    # -hls_playlist_type vod: Indicates this is a VOD playlist (not live)
    # -hls_segment_filename: Pattern for segment file names
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

    # Check if ffmpeg command executed successfully
    if result.returncode != 0:
        print(f"Error converting audio: {result.stderr}")
        return False
    return True
