#!/usr/bin/env python3
"""TODO(Yuquan): Maintain deterministic byte fixtures for protocol tests; these are NOT decodable video."""
import argparse
import hashlib
import json
from pathlib import Path


def generate(directory: Path, count=4, duration_ms=150, size=32768):
    directory.mkdir(parents=True, exist_ok=True)
    segments = []
    for i in range(count):
        data = (hashlib.sha256(f"fixture-{i}".encode()).digest() * ((size + 31) // 32))[:size]
        sid = f"fixture--{i:06d}"
        (directory / f"{sid}.bin").write_bytes(data)
        segments.append({"segment_id": sid, "index": i, "duration_ms": duration_ms,
                         "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()})
    manifest = {"schema_version": 1, "video_id": "fixture", "representation": "synthetic-bytes-v1", "segments": segments}
    path = directory / "manifest.json"
    path.write_text(json.dumps(manifest, indent=2) + "\n")
    return path


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path(".data/fixture"))
    args = parser.parse_args()
    print(generate(args.output))
