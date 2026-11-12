#!/usr/bin/env bash
# TODO(Yuquan): Preserve one encoder timeline when extending continuous fMP4 packaging.
set -euo pipefail
input="${1:?input video required}"
output="${2:?output directory required}"
mkdir -p "$output"
if compgen -G "$output/segment-*.mp4" > /dev/null; then
  echo 'Output already contains segments; use a new directory.' >&2
  exit 1
fi
encoded="$(mktemp "$output/.encoded-XXXXXXXX")"
trap 'rm -f "$encoded"' EXIT
# Encode once: restarting an AAC encoder per segment inserts repeated priming.
# Every GOP begins at an IDR and every moof retains its original decode timestamp.
ffmpeg -hide_banner -loglevel error -y -i "$input" -map 0:v:0 -map '0:a:0?' \
  -c:v libx264 -pix_fmt yuv420p -bf 0 -force_key_frames 'expr:gte(t,n_forced*2)' -sc_threshold 0 \
  -c:a aac -profile:a aac_low -b:a 128k \
  -movflags +frag_keyframe+empty_moov+default_base_moof -f mp4 "$encoded"
python3 "$(dirname "$0")/fragment_mp4.py" "$encoded" "$output"
