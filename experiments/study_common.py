"""Real-process study support. No synthetic results; all clocks are observer monotonic."""
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import shutil
import subprocess
import sys
import tarfile
import threading
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'tests/system'))
from cluster import Cluster, ing, wait


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


def probe(url, method='GET', value=None, timeout=.7, binary=False, headers=None):
    """One request only. HTTP rejection, timeout, malformed/truncated body stay failures."""
    start = time.monotonic()
    result = {'url': url, 'method': method, 'started': start, 'status': None, 'ok': False}
    data = json.dumps(value).encode() if value is not None else None
    try:
        req = urllib.request.Request(url, data=data, method=method,
            headers={'Content-Type': 'application/json', **(headers or {})})
        with urllib.request.urlopen(req, timeout=timeout) as res:
            raw = res.read()
            result.update(status=res.status, body=raw if binary else json.loads(raw), ok=True)
    except urllib.error.HTTPError as exc:
        result.update(status=exc.code, error=exc.read().decode(errors='replace'))
    except Exception as exc:
        result['error'] = f'{type(exc).__name__}: {exc}'
    result['ended'] = time.monotonic()
    return result


class Journal:
    def __init__(self, directory, identity):
        self.directory, self.identity = directory, identity
        self.epoch = time.monotonic()
        self.lock = threading.Lock()
        self.files = {}

    def write(self, name, value):
        row = {**self.identity, 'observer_s': time.monotonic() - self.epoch, **value}
        with self.lock:
            if name not in self.files:
                self.files[name] = (self.directory / (name + '.jsonl')).open('a', buffering=1)
            self.files[name].write(json.dumps(row) + '\n')
        return row

    def close(self):
        for f in self.files.values():
            f.close()


def fresh_cluster(cfg, policy, grace=3, repair=True):
    return Cluster(controllers=3, rates=cfg['rates'], quota=32 << 20,
                   grace=grace, repair=repair, repair_policy=policy if policy in ('F25', 'F50', 'L') else 'S')


def provenance(out, cfg):
    out.mkdir(parents=True, exist_ok=True)
    save(out / 'study-config.json', cfg)
    env = {'platform': platform.platform(), 'python': sys.version,
           'go': subprocess.check_output(['go', 'version'], text=True).strip(),
           'commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
           'dirty': subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True),
           'binaries': {n: hashlib.sha256((ROOT / 'bin' / n).read_bytes()).hexdigest() for n in ('peer', 'controller', 'client')},
           'cache_procedure': 'fresh stores/processes; OS page cache not flushed',
           'mode': 'real HTTP payload; headless simulated media consumption',
           'wall_epoch': time.time()}
    names = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=ROOT).decode().split('\0')
    hashes = {}
    with tarfile.open(out / 'source-snapshot.tar.gz', 'w:gz') as archive:
        for name in sorted(set(names)):
            path = ROOT / name
            if not name or name.startswith(('results/', '.data/', 'bin/')) or not path.is_file() or path.is_symlink():
                continue
            hashes[name] = hashlib.sha256(path.read_bytes()).hexdigest()
            archive.add(path, arcname=name, recursive=False)
    env['source_sha256'] = hashes
    save(out / 'environment-and-provenance.json', env)


def start_viewer(c, cfg, video, item, journal, rows, stop):
    if stop.wait(max(0, item['arrival_s'] - (time.monotonic() - journal.epoch))):
        rows.append({**item, 'status': 'start_failed', 'error': 'study stopped'})
        return
    started = time.monotonic() - journal.epoch
    events = []
    cmd = [str(ROOT / 'bin/client'), '-controllers', ','.join(c.controllers), '-video', video,
           '-strategy', 'round_robin', '-window', '3', '-concurrency', '2',
           '-startup-buffer-ms', str(2 * cfg['duration_ms']), '-timeout', f'{cfg["session_timeout_s"]}s']
    try:
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        def read_events():
            for line in p.stderr:
                try:
                    event = json.loads(line)
                    # The process-start interval bounds alignment error. Preserve
                    # source elapsed time and observer receipt time separately.
                    event.update(viewer=item['viewer'], client_launch_s=started)
                    events.append(journal.write('client-events', event))
                except ValueError:
                    journal.write('client-events', {'viewer': item['viewer'], 'parse_error': line})
        thread = threading.Thread(target=read_events)
        thread.start()
        try:
            stdout = p.stdout.read()
            p.wait(timeout=cfg['session_timeout_s'] + 3)
            row = json.loads(stdout)
        except Exception as exc:
            p.kill(); p.wait()
            row = {'status': 'failed', 'error': str(exc)}
        finally:
            thread.join(); p.stdout.close(); p.stderr.close()
        row.update(exit_code=p.returncode)
    except Exception as exc:
        row = {'status': 'start_failed', 'error': str(exc)}
    row.update(item, client_launch_s=started, ended_s=time.monotonic() - journal.epoch)
    rows.append(row)
    journal.write('sessions', row)


def archive_logs(c, directory):
    target = directory / 'process-logs'
    target.mkdir(exist_ok=True)
    for log in c.path.glob('*.log'):
        shutil.copy2(log, target / log.name)


def metadata(c, path, journal=None, timeout=.7):
    for base in c.controllers:
        res = probe(base + '/api/v1' + path, timeout=timeout)
        if journal:
            journal.write('metadata-probes', res)
        if res['ok']:
            return res['body']
    return None


def run_identity(cell):
    return '-'.join(str(cell[k]) for k in ('case', 'load', 'policy', 'grace', 'outage', 'delay', 'seed'))
