#!/usr/bin/env python3
"""TODO(Tian): Add paired run-level differences and uncertainty; keep failed/stopped sessions in completion denominators."""
import argparse
import csv
import json
from collections import defaultdict
from pathlib import Path


def aggregate(path):
    rows = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    if not rows:
        raise ValueError("no session rows")
    groups = defaultdict(list)
    for row in rows:
        groups[row["run_id"]].append(row)
    output = path.with_name("runs.csv")
    with output.open("w", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=["run_id", "sessions", "completed", "failed", "stopped", "completed_session_rate", "mean_completed_rebuffering_ratio"])
        writer.writeheader()
        for run_id, items in groups.items():
            completed = [item for item in items if item["status"] == "completed"]
            writer.writerow({"run_id": run_id, "sessions": len(items), "completed": len(completed), "failed": sum(item["status"] == "failed" for item in items), "stopped": sum(item["status"] == "stopped" for item in items), "completed_session_rate": len(completed) / len(items), "mean_completed_rebuffering_ratio": sum(item["rebuffering_ratio"] for item in completed) / len(completed) if completed else ""})
    return output


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sessions", type=Path)
    print(aggregate(parser.parse_args().sessions))
