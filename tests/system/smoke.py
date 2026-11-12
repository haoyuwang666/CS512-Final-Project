#!/usr/bin/env python3
"""TODO(Haoyu): Extend this real-process Raft smoke test with partitions, disk-loss faults and task recovery."""
import argparse
import importlib.util
import json
import os
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def load(name, relative):
    spec = importlib.util.spec_from_file_location(name, ROOT / relative)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


ing = load("ingest", "scripts/ingest/ingest.py")
fixture = load("fixture", "scripts/media/fixture.py")


def ports(count):
    sockets = [socket.socket() for _ in range(count)]
    try:
        for sock in sockets:
            sock.bind(("127.0.0.1", 0))
        return [sock.getsockname()[1] for sock in sockets]
    finally:
        for sock in sockets:
            sock.close()


def eventually(action, timeout=20):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            return action()
        except Exception as error:
            last = error
            time.sleep(0.2)
    raise RuntimeError(f"condition not reached: {last}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bin-dir", type=Path, default=ROOT / "bin")
    args = parser.parse_args()
    args.bin_dir = args.bin_dir.resolve()
    processes = {}
    logs = []
    with tempfile.TemporaryDirectory(prefix="cooperative-smoke-") as temp:
        temp = Path(temp)
        peer_ports, http_ports, raft_ports = ports(3), ports(3), ports(3)
        peers = [{"peer_id": f"peer-{i+1}", "url": f"http://127.0.0.1:{port}"} for i, port in enumerate(peer_ports)]
        members = [{"id": f"controller-{i+1}", "raft_address": f"127.0.0.1:{raft_ports[i]}", "http_url": f"http://127.0.0.1:{port}"} for i, port in enumerate(http_ports)]
        controllers = [m["http_url"] for m in members]

        def start(key, binary, extra):
            log = open(temp / f"{key}.log", "ab")
            logs.append(log)
            processes[key] = subprocess.Popen([str(args.bin_dir / binary)], cwd=ROOT, env={**os.environ, **extra}, stdout=log, stderr=log)

        def start_controller(i):
            m = members[i]
            start(m["id"], "controller", {"NODE_ID": m["id"], "HTTP_ADDR": f"127.0.0.1:{http_ports[i]}", "RAFT_BIND": m["raft_address"], "RAFT_MEMBERS": json.dumps(members), "PEERS": json.dumps(peers), "DATA_DIR": str(temp / m["id"]), "BOOTSTRAP": "true" if i == 0 else "false"})

        def stop(key):
            process = processes[key]
            process.terminate()
            try:
                process.wait(timeout=8)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()

        def read_manifest():
            for base in controllers:
                try:
                    return ing.request(base + "/api/v1/videos/fixture/manifest")
                except (urllib.error.URLError, TimeoutError):
                    pass
            raise RuntimeError("no readable controller")

        try:
            for i, peer in enumerate(peers):
                start(peer["peer_id"], "peer", {"PEER_ID": peer["peer_id"], "HTTP_ADDR": f"127.0.0.1:{peer_ports[i]}", "DATA_DIR": str(temp / peer["peer_id"]), "UPLOAD_BYTES_PER_SECOND": "10000000"})
                eventually(lambda p=peer: ing.request(p["url"] + "/api/v1/health"))
            for i in range(3):
                start_controller(i)
            for base in controllers:
                eventually(lambda b=base: ing.request(b + "/api/v1/health"))
            manifest_path = fixture.generate(temp / "fixture")
            ing.ingest(manifest_path, controllers)
            assert read_manifest()["video_id"] == "fixture"
            # Repeating the entire ingestion exercises operation-ID idempotency.
            ing.ingest(manifest_path, controllers)
            def play():
                result = subprocess.run([str(args.bin_dir / "client"), "-controllers", ",".join(controllers), "-video", "fixture", "-timeout", "20s"], cwd=ROOT, text=True, capture_output=True, timeout=25)
                if result.returncode:
                    raise RuntimeError(result.stdout + result.stderr)
                summary = json.loads(result.stdout)
                assert summary["status"] == "completed" and summary["segments_played"] == 4
                return summary
            first = play()
            demo_port = ports(1)[0]
            demo_url = f"http://127.0.0.1:{demo_port}"
            start("demo-api", "demo-api", {"HTTP_ADDR": f"127.0.0.1:{demo_port}", "CONTROLLERS": ",".join(controllers), "SESSION_DIR": str(temp / "sessions")})
            eventually(lambda: ing.request(demo_url + "/api/v1/health"))
            session_id = ing.request(demo_url + "/api/v1/sessions", "POST", {"video_id": "fixture", "strategy": "round_robin"})["session_id"]

            def completed_session():
                session = ing.request(demo_url + f"/api/v1/sessions/{session_id}")
                assert session["summary"]["status"] == "completed", session
                return session

            eventually(completed_session)
            with urllib.request.urlopen(demo_url + f"/api/v1/sessions/{session_id}/events", timeout=5) as response:
                assert "session_completed" in response.read().decode()
            try:
                ing.request(demo_url + "/api/v1/sessions", "POST", {"video_id": "fixture", "strategy": "unknown"})
                raise AssertionError("unknown strategy was accepted")
            except urllib.error.HTTPError as error:
                assert error.code == 400
            deadline_id = ing.request(demo_url + "/api/v1/sessions", "POST", {"video_id": "fixture", "strategy": "deadline", "startup_buffer_ms": 200})["session_id"]
            def completed_deadline():
                session = ing.request(demo_url + f"/api/v1/sessions/{deadline_id}")
                assert session["summary"]["status"] == "completed", session
                return session
            eventually(completed_deadline)
            experiment_output = temp / "experiment"
            subprocess.run([os.environ.get("PYTHON", "python3"), str(ROOT / "experiments/run.py"), "--scenario", str(ROOT / "experiments/scenarios/baseline.json"), "--controllers", ",".join(controllers), "--output-dir", str(experiment_output)], env={**os.environ, "CLIENT_BIN": str(args.bin_dir / "client")}, check=True, capture_output=True, text=True, timeout=30)
            rows = [json.loads(line) for line in (experiment_output / "sessions.jsonl").read_text().splitlines()]
            assert len(rows) == 5 and all(row["status"] == "completed" and not row["formal_experiment"] for row in rows)
            for script in ["aggregate.py", "plot.py"]:
                subprocess.run([os.environ.get("PYTHON", "python3"), str(ROOT / "experiments" / script), str(experiment_output / "sessions.jsonl")], check=True, capture_output=True, text=True)
            assert (experiment_output / "runs.csv").is_file() and (experiment_output / "diagnostic.svg").is_file()
            stop("peer-1")
            play()
            cluster = ing.request(controllers[0] + "/api/v1/cluster")
            leader_url = cluster["leader_url"]
            leader_index = controllers.index(leader_url)
            stop(members[leader_index]["id"])
            eventually(read_manifest)
            play()
            start_controller(leader_index)
            eventually(lambda: ing.request(leader_url + "/api/v1/health"))
            eventually(read_manifest)
            # With only one live voter no new write may return success.
            stop("controller-2")
            stop("controller-3")
            time.sleep(2)
            try:
                ing.request(controllers[0] + "/api/v1/videos", "POST", {"operation_id": "no-quorum", "manifest": {}})
                raise AssertionError("write unexpectedly succeeded without quorum")
            except urllib.error.HTTPError as error:
                assert error.code == 503, error.code
            print(json.dumps({"status": "PASS", "checks": ["3-node Raft", "durable dual-replica ingestion", "idempotent retry", "real HTTP playback", "session REST/SSE", "deadline playback with startup buffering", "unknown strategy rejection", "baseline runner and CSV/SVG pipeline", "peer fallback", "leader failover", "controller restart", "no-quorum write rejection"], "baseline": first}, indent=2))
        except Exception:
            for path in temp.glob("*.log"):
                print(f"\n{path.name}:\n{path.read_text()[-5000:]}")
            raise
        finally:
            for key in list(processes):
                if processes[key].poll() is None:
                    stop(key)
            for log in logs:
                log.close()


if __name__ == "__main__":
    main()
