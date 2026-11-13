#!/usr/bin/env python3
"""TODO(Yuquan): Add timed trace-driven faults; explicit Docker/Console actions never remove stored data."""
import argparse
import subprocess
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["stop", "start"])
    parser.add_argument("service", choices=[f"peer-{i}" for i in range(1, 4)] + [f"controller-{i}" for i in range(1, 4)])
    parser.add_argument("--backend",choices=["docker","console"],default="docker")
    parser.add_argument("--console",default="http://127.0.0.1:8090")
    args = parser.parse_args()
    if args.backend=="console":
        if not args.service.startswith("peer-"):
            parser.error("Console fault injection currently supports its own Peers only")
        req=urllib.request.Request(args.console.rstrip("/")+f"/api/local/peers/{args.service}/{args.action}",data=b"{}",headers={"Content-Type":"application/json"},method="POST")
        with urllib.request.urlopen(req,timeout=10) as response:
            print(response.read().decode())
    else:
        subprocess.run(["docker", "compose", "-f", str(ROOT / "deployments/compose.yaml"), args.action, args.service], check=True)
