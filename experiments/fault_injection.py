#!/usr/bin/env python3
"""TODO(Tian): Add timed trace-driven partitions; current actions require an explicit service and never remove volumes."""
import argparse
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["stop", "start"])
    parser.add_argument("service", choices=[f"peer-{i}" for i in range(1, 4)] + [f"controller-{i}" for i in range(1, 4)])
    args = parser.parse_args()
    subprocess.run(["docker", "compose", "-f", str(ROOT / "deployments/compose.yaml"), args.action, args.service], check=True)
