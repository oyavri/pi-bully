import argparse
import os
import signal
import subprocess
import sys
import threading
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
    parser.add_argument("--kill-worker-after", type=float, default=-1.0)

    return parser.parse_args()


def maybe_kill_worker(args: argparse.Namespace) -> None:
    if args.kill_worker_after < 0:
        return

    worker_pid_raw = os.environ.get("WORKER_PID", "")
    if not worker_pid_raw:
        sys.stderr.write("WORKER_PID is required for worker termination\n")
        sys.exit(1)

    try:
        worker_pid = int(worker_pid_raw)
    except ValueError:
        sys.stderr.write(f"invalid WORKER_PID value: {worker_pid_raw}\n")
        sys.exit(1)

    time.sleep(args.kill_worker_after)

    os.kill(worker_pid, signal.SIGKILL)


def maybe_fail(args: argparse.Namespace, proc: subprocess.Popen[str]) -> None:
    if args.fail_after < 0:
        return

    time.sleep(args.fail_after)

    if proc.poll() is None:
        proc.kill()

    sys.stderr.write(args.fail_message + "\n")
    sys.stderr.flush()
    os._exit(args.fail_exit_code)


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

    cmd.append(task_output)

    proc: subprocess.Popen[str] = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    threads: list[threading.Thread] = []

    if args.fail_after >= 0:
        fail_thread = threading.Thread(
            target=maybe_fail, args=(args, proc), daemon=True
        )
        fail_thread.start()
        threads.append(fail_thread)

    if args.kill_worker_after >= 0:
        kill_thread = threading.Thread(
            target=maybe_kill_worker, args=(args,), daemon=True
        )
        kill_thread.start()
        threads.append(kill_thread)

    stdout, stderr = proc.communicate()
    if proc.returncode != 0:
        sys.stderr.write(stderr or stdout)
        return proc.returncode

    return 0


if __name__ == "__main__":
    sys.exit(main())
