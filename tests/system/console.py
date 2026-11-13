#!/usr/bin/env python3
"""TODO(Haoyu): Exercise local orchestration, real upload/trim/ingest and verified Peer delivery in isolated data directories."""
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
BASE = 'http://127.0.0.1:8090'


def request(path, value=None, raw=None, headers=None):
    body = raw if raw is not None else json.dumps(value).encode() if value is not None else None
    req = urllib.request.Request(BASE + path, data=body, headers=headers or ({'Content-Type': 'application/json'} if value is not None else {}))
    with urllib.request.urlopen(req, timeout=20) as res:
        data = res.read()
        return json.loads(data) if 'json' in res.headers.get('Content-Type', '') else (data, res.headers)


def wait(predicate, timeout=60):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        try:
            out = predicate()
            if out:
                return out
        except (urllib.error.URLError, ConnectionError):
            pass
        time.sleep(.3)
    raise AssertionError('Timed out')


def ready():
    s = request('/api/local/status')
    return s if s['nodes'] and all(n['status'] == 'running' for n in s['nodes']) and any(n.get('details', {}).get('state') == 'Leader' for n in s['nodes']) else None


def upload(content, extra=''):
    boundary = 'console-test-boundary'
    body = (f'--{boundary}\r\nContent-Disposition: form-data; name="video"; filename="test.mp4"\r\nContent-Type: video/mp4\r\n\r\n'.encode() + content + b'\r\n')
    for k, v in [('start', '1'), ('duration', '4')]:
        body += f'--{boundary}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode()
    body += f'--{boundary}--\r\n'.encode()
    return request('/api/local/upload', raw=body, headers={'Content-Type': f'multipart/form-data; boundary={boundary}'})


def finished():
    s = request('/api/local/status')
    return s['job'] if not s['busy'] and s['job'] else None


def main():
    # Never attach to or stop a user-owned console.
    with socket.socket() as sock:
        assert sock.connect_ex(('127.0.0.1', 8090)) != 0, 'Stop the existing local console before testing'
    with tempfile.TemporaryDirectory(prefix='cs512-console-') as tmp:
        work = Path(tmp)
        for name in ('bin', 'scripts', 'frontend'):
            (work / name).symlink_to(ROOT / name, target_is_directory=True)
        source = work / 'fixture.mp4'
        subprocess.run(['ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'testsrc2=size=320x180:rate=24', '-f', 'lavfi', '-i', 'sine=frequency=440', '-t', '6', '-c:v', 'libx264', '-c:a', 'aac', str(source)], check=True)
        with (work / 'console.log').open('w') as log:
            proc = subprocess.Popen([str(ROOT / 'bin/local-console')], cwd=work, stdout=log, stderr=log)
            try:
                wait(lambda: request('/api/local/status'))
                try:
                    request('/api/local/start', {'peers': 1, 'controllers': 1})
                    raise AssertionError('Single Peer must be rejected')
                except urllib.error.HTTPError as e:
                    assert e.code == 400
                try:
                    request('/api/local/stop', {}, headers={'Content-Type': 'application/json', 'Origin': 'https://foreign.example'})
                    raise AssertionError('Foreign origin must be rejected')
                except urllib.error.HTTPError as e:
                    assert e.code == 403
                request('/api/local/start', {'peers': 2, 'controllers': 1})
                assert len(wait(ready)['nodes']) == 4
                vid = upload(source.read_bytes())['video_id']
                job = wait(finished)
                assert job['stage'] == 'ready', job
                session = request('/api/v1/media/sessions', {'video_id': vid, 'strategy': 'round_robin', 'window': 3, 'concurrency': 2})
                assert 3900 <= session['manifest']['media']['duration_ms'] <= 4200
                sources = set()
                for index, seg in enumerate(session['manifest']['segments']):
                    data, headers = request(f'/api/v1/media/sessions/{session["session_id"]}/segments/{index}')
                    assert hashlib.sha256(data).hexdigest() == seg['sha256']
                    sources.add(headers['X-Peer-ID'])
                assert len(sources) == 2, sources
                assert request('/api/local/videos')[0]['id'] == vid
                # A failed Peer must cause retry against the surviving replica.
                request('/api/local/peers/peer-2/stop', {})
                time.sleep(.3)
                data, headers = request(f'/api/v1/media/sessions/{session["session_id"]}/segments/1')
                assert headers['X-Peer-ID'] == 'peer-1'
                request('/api/local/peers/peer-2/start', {})
                wait(ready)
                request('/api/local/nodes/peer-1/restart', {})
                wait(ready)
                request('/api/local/nodes/demo-api/stop', {})
                assert request('/api/local/maintenance')['available']
                request('/api/local/nodes/demo-api/start', {})
                wait(ready)
                request('/api/local/nodes/controller-1/stop', {'expected_role': 'Leader'})
                assert not request('/api/local/maintenance')['available']
                assert request('/')[0]
                request('/api/local/nodes/controller-1/start', {})
                wait(ready)
                assert request('/api/local/maintenance')['available']
                try:
                    request('/api/local/peers/unknown/stop', {})
                    raise AssertionError('Must not control unowned processes')
                except urllib.error.HTTPError as e:
                    assert e.code == 404
                request('/api/local/stop', {})
                assert not request('/api/local/status')['nodes']
                request('/api/local/start', {'peers': 2, 'controllers': 1})
                wait(ready)
                assert request('/api/local/videos')[0]['id'] == vid
                request('/api/local/stop', {})
                request('/api/local/start', {'peers': 3, 'controllers': 3})
                assert len(wait(ready)['nodes']) == 7
                leader=next(n for n in request('/api/local/status')['nodes'] if n.get('details',{}).get('state')=='Leader')
                follower=next(n for n in request('/api/local/status')['nodes'] if n['kind']=='controller' and n['id']!=leader['id'])
                try:
                    request(f'/api/local/nodes/{follower["id"]}/stop', {'expected_role':'Leader'})
                    raise AssertionError('Stale Leader selection must be rejected')
                except urllib.error.HTTPError as e:
                    assert e.code==409
                request(f'/api/local/nodes/{leader["id"]}/stop', {'expected_role':'Leader'})
                wait(lambda:request('/api/local/maintenance')['available'])
                assert any(event['node_id']==leader['id'] and event['action']=='stop' for event in request('/api/local/maintenance')['events'])
                request(f'/api/local/nodes/{leader["id"]}/start', {})
                wait(ready)
                assert request('/api/local/videos') == []
                upload(b'not a video')
                job = wait(finished)
                assert job['stage'] == 'failed' and job['error'], job
                request('/api/local/stop', {})
                print('PASS: lifecycle, origin checks, trim, 2-Peer delivery + SHA256, failover, persistent restart, 3-Controller topology, failure feedback')
            finally:
                proc.terminate()
                try:
                    proc.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait()


if __name__ == '__main__':
    main()
