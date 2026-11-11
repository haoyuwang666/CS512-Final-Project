#!/usr/bin/env python3
"""TODO(Tian): Add seeded request/failure traces, fresh-state resets and warm-up; current runner is a baseline plumbing check, not a controlled study."""
import argparse
import concurrent.futures
import json
import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scenario", type=Path, required=True)
    parser.add_argument("--controllers", default="http://127.0.0.1:8001,http://127.0.0.1:8002,http://127.0.0.1:8003")
    parser.add_argument("--output-dir", type=Path, help="optional run output location; defaults to results/scenario-name")
    args = parser.parse_args()
    cfg = json.loads(args.scenario.read_text())
    if cfg.get("implemented") is False or cfg.get("strategy") != "round_robin" or cfg.get("placement") != "fixed":
        parser.error("requested research policy is not implemented; refusing to produce substitute results")
    output = args.output_dir or ROOT / "results" / cfg["name"]
    output.mkdir(parents=True, exist_ok=True)
    # Seeds label runs only. No randomized workload is implemented yet.
    def viewer(seed, number):
        run_id = f"{cfg['name']}-{seed}"
        result = subprocess.run([os.environ.get("CLIENT_BIN", str(ROOT / "bin/client")), "-controllers", args.controllers,
                                 "-video", cfg["video_id"], "-strategy", cfg["strategy"],
                                 "-window", str(cfg["window"]), "-concurrency", str(cfg["concurrency"]),
                                 "-timeout", f"{cfg['timeout_seconds']}s"], text=True, capture_output=True,
                                timeout=cfg["timeout_seconds"] + 10)
        (output / f"{run_id}-viewer-{number}.events.jsonl").write_text(result.stderr)
        try:
            summary = json.loads(result.stdout)
        except json.JSONDecodeError:
            summary = {"status": "failed", "error": result.stderr, "video_id": cfg["video_id"]}
        return {**summary, "run_id": run_id, "seed_label": seed, "viewer": number, "formal_experiment": False}
    with (output / "sessions.jsonl").open("w") as stream:
        for seed in cfg["seeds"]:
            with concurrent.futures.ThreadPoolExecutor(max_workers=cfg["viewers"]) as pool:
                for result in pool.map(lambda number: viewer(seed, number), range(cfg["viewers"])):
                    stream.write(json.dumps(result) + "\n")
    (output / "config.json").write_text(json.dumps(cfg, indent=2) + "\n")
    print(output / "sessions.jsonl")


if __name__ == "__main__":
    main()
