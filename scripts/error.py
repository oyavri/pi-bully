import argparse
import sys
import time


def main() -> int:
    parser = argparse.ArgumentParser(description="Deterministic failing task")
    parser.add_argument("--sleep-seconds", type=float, default=0.0)
    parser.add_argument("--message", default="mock error task failure")
    parser.add_argument("--exit-code", type=int, default=1)
    args = parser.parse_args()

    if args.sleep_seconds > 0:
        time.sleep(args.sleep_seconds)

    sys.stderr.write(args.message + "\n")
    return args.exit_code


if __name__ == "__main__":
    sys.exit(main())
