import argparse
import os
import subprocess
import sys


def required_env(name: str) -> str:
    value = os.environ.get(name, "")
    if not value:
        sys.stderr.write(f"{name} is required\n")
        sys.exit(1)
    return value


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Transcode a video with ffmpeg")
    parser.add_argument("--width", type=int, default=640)
    parser.add_argument("--height", type=int, default=360)
    parser.add_argument("--video-codec", default="libx264")
    parser.add_argument("--audio-codec", default="aac")
    parser.add_argument("--preset", default="veryfast")
    parser.add_argument("--crf", type=int, default=28)
    parser.add_argument("--video-bitrate", default="")
    parser.add_argument("--audio-bitrate", default="")
    parser.add_argument("--extra-arg", action="append", default=[])
    return parser.parse_args()


def main() -> int:
    task_input = required_env("TASK_INPUT")
    task_output = required_env("TASK_OUTPUT")
    args = parse_args()

    cmd = [
        "ffmpeg",
        "-y",
        "-i",
        task_input,
        "-vf",
        f"scale={args.width}:{args.height}",
        "-c:v",
        args.video_codec,
        "-preset",
        args.preset,
        "-crf",
        str(args.crf),
        "-c:a",
        args.audio_codec,
    ]

    if args.video_bitrate:
        cmd.extend(["-b:v", args.video_bitrate])

    if args.audio_bitrate:
        cmd.extend(["-b:a", args.audio_bitrate])

    for value in args.extra_arg:
        cmd.extend(value.split())

    cmd.append(task_output)

    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr or proc.stdout)
        return proc.returncode

    return 0


if __name__ == "__main__":
    sys.exit(main())
