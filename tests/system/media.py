#!/usr/bin/env python3
"""TODO(Tian): Add browser decoding automation when a supported browser runner is available."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("system_smoke", ROOT / "tests/system/smoke.py")
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)


def main():
    processes = []
    logs = []
    with tempfile.TemporaryDirectory(prefix="cooperative-media-") as temporary:
        temp = Path(temporary)
        try:
            existing = os.environ.get("MEDIA_MANIFEST")
            if existing:
                segments = Path(existing).resolve().parent
            else:
                # A real audiovisual source checks the single-encoder packaging path.
                source = temp / "source.mp4"
                subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
                                "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=24",
                                "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000",
                                "-t", "8", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", str(source)], check=True)
                segments = temp / "segments"
                subprocess.run(["bash", str(ROOT / "scripts/media/segment_video.sh"), str(source), str(segments)], check=True)
                subprocess.run(["python3", str(ROOT / "scripts/media/generate_manifest.py"), str(segments), "--video-id", "media-test"], check=True)
            manifest = json.loads((segments / "manifest.json").read_text())
            assert manifest["representation"] == "continuous-fmp4-h264-aac-v1"
            peer_ports = smoke.ports(3)
            http_ports, raft_ports = smoke.ports(3), smoke.ports(3)
            api_port = smoke.ports(1)[0]
            peers = [{"peer_id": f"peer-{i+1}", "url": f"http://127.0.0.1:{port}"} for i, port in enumerate(peer_ports)]

            def start(binary, name, extra):
                log = open(temp / f"{name}.log", "wb")
                logs.append(log)
                process = subprocess.Popen([str(ROOT / "bin" / binary)], cwd=ROOT,
                                           env={**os.environ, **extra}, stdout=log, stderr=log)
                processes.append(process)
                return process

            peer_processes = [start("peer", peer["peer_id"], {"PEER_ID": peer["peer_id"], "HTTP_ADDR": f"127.0.0.1:{peer_ports[i]}",
                                                               "DATA_DIR": str(temp / peer["peer_id"]), "UPLOAD_BYTES_PER_SECOND": "10000000"})
                              for i, peer in enumerate(peers)]
            for peer in peers:
                smoke.eventually(lambda p=peer: smoke.ing.request(p["url"] + "/api/v1/health"))
            members = [{"id": f"controller-{i+1}", "raft_address": f"127.0.0.1:{raft_ports[i]}",
                        "http_url": f"http://127.0.0.1:{port}"} for i, port in enumerate(http_ports)]
            controllers = [member["http_url"] for member in members]
            def start_controller(i):
                return start("controller", members[i]["id"], {
                    "NODE_ID": members[i]["id"], "HTTP_ADDR": f"127.0.0.1:{http_ports[i]}",
                    "RAFT_BIND": members[i]["raft_address"], "RAFT_MEMBERS": json.dumps(members),
                    "BOOTSTRAP": "true" if i == 0 else "false",
                    "DATA_DIR": str(temp / members[i]["id"]), "PEERS": json.dumps(peers)})
            controller_processes = [start_controller(i) for i in range(3)]
            for controller in controllers:
                smoke.eventually(lambda c=controller: smoke.ing.request(c + "/api/v1/health"))
            smoke.ing.ingest(segments / "manifest.json", controllers)
            def leader_index():
                for controller in controllers:
                    try:
                        leader = smoke.ing.request(controller + "/api/v1/cluster")["leader_url"]
                        if leader in controllers:
                            return controllers.index(leader)
                    except (OSError, ValueError):
                        pass
                raise RuntimeError("no leader")
            api = f"http://127.0.0.1:{api_port}"
            demo_process = start("demo-api", "demo-api", {"HTTP_ADDR": f"127.0.0.1:{api_port}", "CONTROLLERS": ",".join(controllers), "SESSION_DIR": str(temp / "sessions")})
            smoke.eventually(lambda: smoke.ing.request(api + "/api/v1/health"))
            session = smoke.ing.request(api + "/api/v1/media/sessions", "POST", {"video_id": manifest["video_id"], "strategy": "round_robin"})
            session_id = session["session_id"]
            # Force the first replica to fail so delivery must switch sources.
            peer_processes[0].terminate()
            peer_processes[0].wait(timeout=5)
            fragmenter = smoke.load("fragmenter", "scripts/media/fragment_mp4.py")
            combined = bytearray()
            initialization = None
            failed_leader = None
            for i, segment in enumerate(manifest["segments"]):
                if i == 1:
                    # Demand an uncached segment immediately, without waiting for
                    # a new leader: the same media session must survive election.
                    failed_leader = leader_index()
                    controller_processes[failed_leader].kill()
                    controller_processes[failed_leader].wait(timeout=5)
                with urllib.request.urlopen(api + f"/api/v1/media/sessions/{session_id}/segments/{i}", timeout=20) as response:
                    payload = response.read()
                    assert response.headers["Content-Type"] == "video/mp4"
                    assert response.headers["X-Peer-ID"] != "peer-1"
                assert len(payload) == segment["size_bytes"]
                assert hashlib.sha256(payload).hexdigest() == segment["sha256"]
                path = temp / f"delivered-{i}.mp4"
                path.write_bytes(payload)
                probe = json.loads(subprocess.check_output(["ffprobe", "-v", "error", "-show_streams", "-of", "json", str(path)], text=True))
                assert {stream["codec_type"] for stream in probe["streams"]} >= {"video", "audio"}
                structure = list(fragmenter.boxes(payload))
                init = b"".join(payload[start:end] for start, end, kind in structure if kind in (b"ftyp", b"moov"))
                if initialization is None:
                    initialization = init
                    combined.extend(init)
                assert init == initialization, "codec initialization changed between fragments"
                combined.extend(b"".join(payload[start:end] for start, end, kind in structure if kind in (b"moof", b"mdat")))
                # Decode the exact bytes returned through the gateway, including audio.
                subprocess.run(["ffmpeg", "-v", "error", "-xerror", "-i", str(path), "-f", "null", "-"], check=True)
            joined = temp / "continuous.mp4"
            joined.write_bytes(combined)
            packets = json.loads(subprocess.check_output(["ffprobe", "-v", "error", "-show_packets", "-of", "json", str(joined)], text=True))["packets"]
            # Check stored sample durations, including initial A/V alignment.
            # ffprobe may infer nominal packet duration or omit AAC priming's
            # duration, so it is not the authority for fragment continuity.
            intervals = fragmenter.validate_decode_timeline(combined)
            assert len(intervals) == 2
            previous_dts = {}
            for packet in packets:
                track = packet["stream_index"]
                dts = int(packet["dts"])
                if track in previous_dts:
                    assert dts > previous_dts[track], f"track {track} has nonmonotonic decode timestamps"
                previous_dts[track] = dts
            assert len(previous_dts) == 2
            subprocess.run(["ffmpeg", "-v", "error", "-xerror", "-i", str(joined), "-f", "null", "-"], check=True)
            if failed_leader is not None:
                controller_processes[failed_leader] = start_controller(failed_leader)
                smoke.eventually(lambda: smoke.ing.request(controllers[failed_leader] + "/api/v1/health"))
            smoke.ing.request(api + f"/api/v1/media/sessions/{session_id}/stop", "POST", {})
            if os.environ.get("BROWSER_TEST") == "1":
                for inject_failure in (False, True):
                    # Clear caches between baseline and failover runs so the
                    # browser must resolve uncached locations during election.
                    demo_process.terminate()
                    demo_process.wait(timeout=5)
                    demo_process = start("demo-api", f"demo-api-browser-{inject_failure}", {"HTTP_ADDR": f"127.0.0.1:{api_port}", "CONTROLLERS": ",".join(controllers), "SESSION_DIR": str(temp / "sessions")})
                    smoke.eventually(lambda: smoke.ing.request(api + "/api/v1/health"))
                    browser_env = {**os.environ, "MEDIA_VIDEO_ID": manifest["video_id"], "FAILOVER_LEADER_PID": ""}
                    if inject_failure:
                        browser_env["FAILOVER_LEADER_PID"] = str(controller_processes[leader_index()].pid)
                    subprocess.run(["node", str(ROOT / "frontend/tests/browser.mjs"), api], env=browser_env, check=True, timeout=120)
            print(json.dumps({"status": "PASS", "checks": ["H.264/AAC packaging", "real dual-replica ingestion", "peer failure fallback",
                                                              "active media session survives leader failure with uncached locations", "verified media gateway delivery", "all delivered segments decode video and audio", "identical codec initialization", "gap-free stored audio/video decode timelines", "monotonic demuxed decode timestamps", "continuous stream decode", "media stop"],
                              "segments": len(manifest["segments"])}, indent=2))
        finally:
            for process in reversed(processes):
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()
            for log in logs:
                log.close()


if __name__ == "__main__":
    main()
