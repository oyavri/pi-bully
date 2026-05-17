import argparse
import os
import subprocess
import sys
import time


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

    parser.add_argument("--fail-after", type=float, default=-1.0)
    parser.add_argument("--fail-message", default="injected transcode failure")
    parser.add_argument("--fail-exit-code", type=int, default=2)

    return parser.parse_args()


def maybe_fail(args: argparse.Namespace) -> None:
    if args.fail_after < 0:
        return

    if args.fail_after > 0:
        time.sleep(args.fail_after)

    sys.stderr.write(args.fail_message + "\n")
    sys.exit(args.fail_exit_code)


def main() -> int:
    task_input = required_env("TASK_INPUT")
    task_output = required_env("TASK_OUTPUT")
    args = parse_args()
    maybe_fail(args)

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

    cmd.append(task_output)

    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr or proc.stdout)
        return proc.returncode

    return 0


if __name__ == "__main__":
    sys.exit(main())
