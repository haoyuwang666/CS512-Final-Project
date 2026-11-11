#!/usr/bin/env python3
"""TODO(Yuquan): Replace advisory placement with durable reservations; preserve operation IDs on all retries."""
import argparse
import hashlib
import json
import time
import urllib.error
import urllib.request
from pathlib import Path


def request(url, method="GET", value=None, headers=None):
    data = value if isinstance(value, bytes) else None if value is None else json.dumps(value).encode()
    actual_headers = dict(headers or {})
    if value is not None and not isinstance(value, bytes):
        actual_headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, method=method, headers=actual_headers)
    with urllib.request.urlopen(req, timeout=15) as response:
        return json.load(response)


def controller(urls, path, method="GET", value=None):
    last = None
    for _ in range(25):
        for url in urls:
            try:
                return request(url.rstrip("/") + "/api/v1" + path, method, value)
            except urllib.error.HTTPError as error:
                detail = error.read().decode()
                last = RuntimeError(f"{error.code} {detail}")
                if error.code not in (502, 503, 504):
                    raise last from error
            except (urllib.error.URLError, TimeoutError) as error:
                last = error
        time.sleep(0.2)
    raise RuntimeError(f"no writable controller: {last}")


def ingest(manifest_path, urls):
    manifest_path = Path(manifest_path)
    manifest = json.loads(manifest_path.read_text())
    vid = manifest["video_id"]
    digest = hashlib.sha256(json.dumps(manifest, sort_keys=True).encode()).hexdigest()[:24]
    controller(urls, "/videos", "POST", {"operation_id": f"create-{digest}", "manifest": manifest})
    plan = controller(urls, f"/videos/{vid}/placement-plan", "POST", {})["placements"]
    for segment in manifest["segments"]:
        sid = segment["segment_id"]
        data = (manifest_path.parent / f"{sid}.bin").read_bytes()
        if len(data) != segment["size_bytes"] or hashlib.sha256(data).hexdigest() != segment["sha256"]:
            raise ValueError(f"local data mismatch: {sid}")
        for peer in plan[sid]:
            op = hashlib.sha256(f"{digest}-{sid}-{peer['peer_id']}".encode()).hexdigest()[:32]
            headers = {"X-Operation-ID": f"put-{op}", "X-Size-Bytes": str(len(data)),
                       "X-SHA256": segment["sha256"], "X-Representation-Version": "1"}
            receipt = request(peer["url"] + f"/api/v1/segments/{sid}", "PUT", data, headers)
            controller(urls, f"/segments/{sid}/replicas", "POST", {"operation_id": f"confirm-{op}", "receipt": receipt})
    controller(urls, f"/videos/{vid}/publish", "POST", {"operation_id": f"publish-{digest}", "expected_version": 1})
    return {"video_id": vid, "status": "READY", "segments": len(manifest["segments"])}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--controllers", default="http://127.0.0.1:8001,http://127.0.0.1:8002,http://127.0.0.1:8003")
    args = parser.parse_args()
    print(json.dumps(ingest(args.manifest, args.controllers.split(","))))
