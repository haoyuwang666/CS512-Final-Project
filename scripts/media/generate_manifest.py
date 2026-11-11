#!/usr/bin/env python3
"""TODO(Yuquan): Extend manifest generation to audio/representations; actual duration is read using FFprobe."""
import argparse
import hashlib
import json
import re
import shutil
import subprocess
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--video-id", required=True)
    args = parser.parse_args()
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,96}", args.video_id):
        parser.error("video ID must contain only letters, digits, underscores or hyphens")
    files = sorted(args.directory.glob("segment-*.mp4"))
    if not files:
        parser.error("no segment-*.mp4 files found")
    segments = []
    for i, path in enumerate(files):
        duration = float(subprocess.check_output(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", str(path)], text=True))
        sid = f"{args.video_id}--{i:06d}"
        target = args.directory / f"{sid}.bin"
        shutil.copyfile(path, target)
        segments.append({"segment_id": sid, "index": i, "duration_ms": max(1, round(duration * 1000)),
                         "size_bytes": target.stat().st_size, "sha256": hashlib.sha256(target.read_bytes()).hexdigest()})
    manifest = {"schema_version": 1, "video_id": args.video_id, "representation": "independent-mp4-h264-v1", "segments": segments}
    output = args.directory / "manifest.json"
    output.write_text(json.dumps(manifest, indent=2) + "\n")
    print(output)


if __name__ == "__main__":
    main()
