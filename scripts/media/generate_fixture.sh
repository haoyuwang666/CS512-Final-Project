#!/usr/bin/env bash
# TODO(Yuquan): Extend the licensed media fixture library; this generates a real 12-second test video locally.
set -euo pipefail
output="${1:-.data/media}"
mkdir -p "$output"
ffmpeg -hide_banner -loglevel error -y -f lavfi -i 'testsrc2=size=320x180:rate=24' -t 12 \
  -c:v libx264 -pix_fmt yuv420p -g 48 -keyint_min 48 -sc_threshold 0 "$output/source.mp4"
bash scripts/media/segment_video.sh "$output/source.mp4" "$output/segments"
python3 scripts/media/generate_manifest.py "$output/segments" --video-id demo-video
