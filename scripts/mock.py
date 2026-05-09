import argparse
import os
import shutil
import sys
import time


def required_env(name: str) -> str:
    value = os.environ.get(name, "")
    if not value:
        sys.stderr.write(f"{name} is required\n")
        sys.exit(1)
    return value


def main() -> int:
    parser = argparse.ArgumentParser(description="Mock task that sleeps and writes output")
    parser.add_argument("--sleep-seconds", type=float, default=1.0)
    parser.add_argument("--message", default="mock task output\n")
    parser.add_argument("--copy-input", action="store_true")
    args = parser.parse_args()

    task_output = required_env("TASK_OUTPUT")
    task_input = os.environ.get("TASK_INPUT", "")

    time.sleep(args.sleep_seconds)

    if args.copy_input and task_input:
        shutil.copyfile(task_input, task_output)
        return 0

    with open(task_output, "w", encoding="utf-8") as f:
        f.write(args.message)

    return 0


if __name__ == "__main__":
    sys.exit(main())
