#!/usr/bin/env bash
# TODO(Yuquan): Tune media encoding for research workloads; first version produces independently decodable MP4 segments.
set -euo pipefail
input="${1:?input video required}"
output="${2:?output directory required}"
mkdir -p "$output"
ffmpeg -hide_banner -loglevel error -y -i "$input" -map 0:v:0 -an -c:v libx264 \
  -force_key_frames 'expr:gte(t,n_forced*2)' -sc_threshold 0 \
  -f segment -segment_time 2 -reset_timestamps 1 "$output/segment-%06d.mp4"
