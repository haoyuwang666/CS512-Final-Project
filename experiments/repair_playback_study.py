#!/usr/bin/env python3
"""E1-E4 compact, paired, real-transfer experiments; shared cells execute once."""
import argparse
import concurrent.futures
import json
import random
import threading
import time
from pathlib import Path
from study_common import *


def matrix(cfg):
    cells = {}
    def add(experiment, seed, load='high', policy='S', grace=None, outage='persistent', delay=0):
        cell = dict(case='peer', seed=seed, load=load, policy=policy,
                    grace=cfg['main_grace'] if grace is None else grace, outage=outage, delay=delay)
        key = run_identity(cell)
        if key not in cells:
            cells[key] = {**cell, 'experiments': []}
        cells[key]['experiments'].append(experiment)
    for seed in cfg['seeds']:
        for load in ('low', 'high'):
            for policy in ('N', 'OFF', 'S'):
                add('E1', seed, load, policy)
        for grace in cfg['graces']:
            for outage in ('brief', 'persistent'):
                add('E2', seed, grace=grace, outage=outage)
        for load in ('high', 'burst'):
            for policy in ('S', 'F25', 'F50', 'L'):
                add('E3', seed, load, policy)
    for seed in cfg['e4_seeds']:
        for policy in ('S', cfg['fixed_e4_policy'], 'L'):
            for delay in cfg['second_delays']:
                add('E4', seed, policy=policy, delay=delay)
    ordered = list(cells.values())
    random.Random(cfg['treatment_order_seed']).shuffle(ordered)
    return ordered


def trace_for(cell, cfg=None):
    cfg = cfg or {"low_viewers": 2, "high_viewers": 6}
    rng = random.Random(cell['seed'])
    arrivals = [rng.uniform(0, .5) for _ in range(cfg['low_viewers'] if cell['load'] == 'low' else cfg['high_viewers'])]
    if cell['load'] == 'burst':
        arrivals = [rng.uniform(0, .5), rng.uniform(0, .5), 4, 4.2, 12, 12.2]
    return [{'viewer': i, 'arrival_s': a} for i, a in enumerate(arrivals)]


def collect(c, cfg, manifest, journal, stop):
    next_stats = next_verify = next_task = 0
    while not stop.is_set():
        now = time.monotonic() - journal.epoch
        if now >= next_stats:
            for p in c.peers:
                res = probe(p['url'] + '/api/v1/stats')
                journal.write('peer-traffic-and-policy', {'peer_id': p['peer_id'], **res})
            next_stats = now + cfg['stats_interval_s']
        if now >= next_task:
            state = metadata(c, '/maintenance', journal)
            journal.write('tasks', {'state': state})
            next_task = now + cfg['task_interval_s']
        if now >= next_verify:
            valid = {}
            for p in c.peers:
                res = probe(p['url'] + '/api/v1/inventory')
                journal.write('inventory-probes', {'peer_id': p['peer_id'], **res})
                valid[p['peer_id']] = {a['segment_id']: a for a in res['body']['receipts']} if res['ok'] else {}
            copies = {}
            for seg in manifest['segments']:
                loc = metadata(c, f'/segments/{seg["segment_id"]}/locations')
                if loc is None:
                    copies[seg['segment_id']] = None
                else:
                    copies[seg['segment_id']] = [p['peer_id'] for p in loc['locations']
                        if c.processes[p['peer_id']].poll() is None and (a := valid.get(p['peer_id'], {}).get(seg['segment_id']))
                        and a['sha256'] == seg['sha256'] and a['size_bytes'] == seg['size_bytes']]
            journal.write('tasks-and-copy-verification', {'copies': copies})
            next_verify = now + cfg['verify_interval_s']
        stop.wait(.05)


def load_jsonl(path):
    return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []


def summarize(directory, cell, cfg, correctness, sessions):
    events = load_jsonl(directory / 'client-events.jsonl')
    inject = load_jsonl(directory / 'injections.jsonl')
    fault = next((r['observer_s'] for r in inject if r['kind'] in ('first_failure', 'no_fault_marker')), None)
    affected = correctness.get('affected', [])
    restored = None; previous = None; one = zero = unknown = 0
    snapshots = load_jsonl(directory / 'tasks-and-copy-verification.jsonl')
    post = [r for r in snapshots if fault is not None and fault <= r['observer_s'] <= cfg['horizon_s']]
    # Right-continuous step integration, with unobserved/failed samples explicit.
    last_t, last_counts = fault, {s: (['original-1', 'original-2'] if cell['policy']=='N' else ['original-survivor']) for s in affected}
    size = cfg['segment_bytes']
    for row in post + [{'observer_s': cfg['horizon_s'], 'copies': None}]:
        dt = max(0, row['observer_s'] - last_t)
        for sid in affected:
            v = last_counts.get(sid) if last_counts is not None else None
            if v is None: unknown += size * dt
            elif len(v) == 1: one += size * dt
            elif len(v) == 0: zero += size * dt
        last_t, last_counts = row['observer_s'], row['copies']
        if cell['policy']!='N' and row['copies'] is not None and affected and restored is None and all(row['copies'].get(s) is not None and len(row['copies'][s]) >= 2 for s in affected):
            restored = row['observer_s'] if cell['policy']!='N' else None; previous = max(fault, next((r['observer_s'] for r in reversed(post) if r['observer_s'] < restored), fault))
    stalls = []
    exposure = False; alternate = None
    for session in sessions:
        viewer_events = [e for e in events if e['viewer'] == session['viewer']]
        stall_start = None; total = 0
        verified = set()
        for e in viewer_events:
            # Wall-receipt alignment uncertainty is archived; duration uses the
            # client monotonic elapsed clock aligned at Popen launch.
            at = e['client_launch_s'] + e.get('since_start_ms', 0) / 1000
            if e['type'] == 'stall_started': stall_start = at
            if e['type'] in ('stall_ended', 'session_failed', 'session_stopped', 'session_completed') and stall_start is not None:
                total += max(0, at - max(stall_start, fault or 0)); stall_start = None
            if e['type'] == 'segment_requested' and fault is not None and at >= fault and e.get('segment_id') in affected and e['segment_id'] not in verified:
                exposure = True
            if e['type'] == 'segment_verified':
                verified.add(e['segment_id'])
                if fault is not None and at >= fault and e['segment_id'] in affected and e.get('peer_id') != correctness.get('failed_peer'):
                    alternate = at - fault if alternate is None else min(alternate, at - fault)
        if stall_start is not None:
            total += max(0, session['ended_s'] - max(stall_start, fault or 0))
        stalls.append(total)
    counters = {}
    overlap = False
    for row in load_jsonl(directory / 'peer-traffic-and-policy.jsonl'):
        if not row.get('ok'): continue
        b = row['body']; key = (row['peer_id'], b['instance_id'])
        counters[key] = b.get('class_bytes', {})
        for sample in b.get('policy_history', []):
            if sample['active_foreground'] > 0 and sample['active_repair'] > 0: overlap = True
    traffic = {cl: sum(v.get(cl, 0) for v in counters.values()) for cl in ('repair', 'foreground', 'probe')}
    completed = [s for s in sessions if s['status'] == 'completed']
    tasks = {}
    for row in load_jsonl(directory / 'tasks.jsonl'):
        if row['state']: tasks.update(row['state'].get('tasks') or {})
    return {**cell, 'run_id': directory.name, 'status': 'completed',
        'scheduled': len(trace_for(cell,cfg)), 'completed': len(completed),
        'completion_rate': len(completed) / len(trace_for(cell,cfg)),
        'mean_startup_s_completed': sum(s['startup_delay_ms'] for s in completed)/1000/len(completed) if completed else None,
        'mean_post_fault_stall_s_all_sessions': sum(stalls)/len(trace_for(cell,cfg)),
        'mean_rebuffering_ratio_completed': sum(s['rebuffering_ratio'] for s in completed)/len(completed) if completed else None,
        'recovery_s': restored - fault if restored is not None else None,
        'recovery_interval_lower_s': previous - fault if previous is not None else None,
        'recovery_target_met': restored is not None and restored - fault <= cfg['recovery_target_s'],
        'recovery_censored': restored is None and cell['policy'] != 'N',
        'censor_reason': 'not observed by fixed horizon' if restored is None and cell['policy'] != 'N' else None,
        'one_copy_byte_seconds': one, 'zero_copy_byte_seconds': zero, 'unknown_copy_byte_seconds': unknown,
        'traffic_bytes': traffic, 'repair_tasks': len(tasks),
        'failed_repair_tasks': sum(t['state']=='failed' for t in tasks.values()),
        'post_fault_exposure': exposure, 'repair_foreground_overlap': overlap,
        'injection_preconditions_met': all(r.get('precondition_satisfied',False) and r.get('effect_confirmed',True) for r in inject),
        'metadata_probe_attempts': len(load_jsonl(directory / 'metadata-probes.jsonl')),
        'alternate_delivery_s': alternate, 'correctness': correctness,
        'simulated_consumption': True, 'formal_experiment': False}


def run_cell(out, cfg, cell):
    directory = out / run_identity(cell); directory.mkdir()
    identity = {k: cell[k] for k in ('seed', 'policy', 'experiments')}
    journal = Journal(directory, {'run_id': directory.name, **identity})
    save(directory / 'config.json', {**cfg, **cell})
    c = fresh_cluster(cfg, cell['policy'], cell['grace'], cell['policy'] != 'OFF')
    stop = threading.Event(); collector = None; threads = []; sessions = []; observer_errors = []
    correctness = {}; status = 'setup_failed'
    try:
        c.__enter__()
        manifest = c.fixture(cfg['segments'], cfg['duration_ms'], cfg['segment_bytes'], video=f'seed-{cell["seed"]}')
        save(directory / 'manifest.json', manifest)
        initial = {s['segment_id']: c.call(f'/segments/{s["segment_id"]}/locations')['locations'] for s in manifest['segments']}
        failed = f'peer-{cell["seed"] % 3 + 1}'
        affected = [s for s, loc in initial.items() if any(p['peer_id'] == failed for p in loc)]
        # Sentinel selected near the end, before treatment outcomes are observed.
        sentinel = next(s for s in reversed(manifest['segments']) if s['segment_id'] in affected)
        holders = initial[sentinel['segment_id']]
        second = next(p['peer_id'] for p in holders if p['peer_id'] != failed)
        target = next(p for p in c.peers if p['peer_id'] not in [h['peer_id'] for h in holders])
        initial_stats = [ing.request(p['url'] + '/api/v1/stats') for p in c.peers]
        assert all(len(v) == 2 for v in initial.values())
        assert sum(s['stored_bytes'] for s in initial_stats) == 2 * cfg['segments'] * cfg['segment_bytes']
        for p in c.peers:
            inv = ing.request(p['url'] + '/api/v1/inventory')
            assert len(inv['receipts']) == sum(any(h['peer_id'] == p['peer_id'] for h in loc) for loc in initial.values())
        correctness.update(initial_two_copies=True, initial=initial, initial_stats=initial_stats,
                           failed_peer=failed, affected=affected, sentinel=sentinel, second_peer=second, target=target)
        trace = trace_for(cell,cfg)
        save(directory / 'workload-and-fault-trace.json', {'viewers': trace, 'failed_peer': failed,
            'first_fault_s': cfg['fault_s'], 'restore_s': cfg['fault_s']+cfg['brief_s'] if cell['outage']=='brief' else None,
            'second_peer': second if cell['delay'] else None, 'second_delay_s': cell['delay'], 'sentinel': sentinel})
        journal.epoch = time.monotonic()
        def observe():
            try: collect(c, cfg, manifest, journal, stop)
            except Exception as exc:
                observer_errors.append(str(exc))
                journal.write('observer-errors', {'error':str(exc)})
        collector = threading.Thread(target=observe); collector.start()
        for item in trace:
            t = threading.Thread(target=start_viewer, args=(c,cfg,manifest['video_id'],item,journal,sessions,stop));t.start();threads.append(t)
        schedule = [(cfg['fault_s'], 'no_fault_marker' if cell['policy']=='N' else 'first_failure', failed)]
        if cell['outage']=='brief': schedule.append((cfg['fault_s']+cfg['brief_s'], 'restore', failed))
        if cell['delay']: schedule.append((cfg['fault_s']+cell['delay'], 'second_failure', second))
        for at, kind, node in sorted(schedule):
            stop.wait(max(0, journal.epoch+at-time.monotonic()))
            requested = time.monotonic()-journal.epoch
            was_running = c.processes[node].poll() is None
            if kind == 'second_failure':
                reg = metadata(c, f'/segments/{sentinel["segment_id"]}/locations', journal)
                receipt = probe(target['url']+f'/api/v1/segments/{sentinel["segment_id"]}/receipt')
                journal.write('second-fault-precondition', {'registration':reg,'target_receipt':receipt})
            if kind in ('first_failure','second_failure'):
                stats=probe(next(p['url'] for p in c.peers if p['peer_id']==node)+'/api/v1/stats')
                journal.write('peer-traffic-and-policy', {'peer_id':node,**stats})
                c.stop(node)
            elif kind=='restore': c.start(node)
            journal.write('injections', {'kind':kind,'node':node,'requested_s':requested,
                'precondition_satisfied': ((not was_running) if kind=='restore' else was_running),
                'effect_confirmed': (c.processes[node].poll() is None if kind in ('restore','no_fault_marker') else c.processes[node].poll() is not None)})
            if kind == 'second_failure':
                res = probe(target['url']+f'/api/v1/segments/{sentinel["segment_id"]}', timeout=5, binary=True,headers={'X-Transfer-Class':'probe'})
                body=res.pop('body',b'')
                valid=res['ok'] and len(body)==sentinel['size_bytes'] and hashlib.sha256(body).hexdigest()==sentinel['sha256']
                reg=metadata(c,f'/segments/{sentinel["segment_id"]}/locations',journal)
                correctness['second_fault_delivery']={'hash_valid':valid,'probe':res,
                    'committed':reg is not None and any(p['peer_id']==target['peer_id'] for p in reg['locations'])}
        stop.wait(max(0,journal.epoch+cfg['horizon_s']-time.monotonic()))
        status='completed'
        if observer_errors:
            correctness['observer_errors']=observer_errors
            status='measurement_failed'
    except Exception as exc:
        correctness['error']=f'{type(exc).__name__}: {exc}'
    finally:
        stop.set()
        for t in threads:t.join(timeout=cfg['session_timeout_s']+5)
        if collector:collector.join(timeout=15)
        if status=='completed':
            for peer in c.peers:
                journal.write('peer-traffic-and-policy', {'peer_id':peer['peer_id'],'final':True,**probe(peer['url']+'/api/v1/stats')})
        archive_logs(c,directory)
        c.__exit__(None,None,None)
        journal.close()
    save(directory/'correctness.json',correctness)
    result=summarize(directory,cell,cfg,correctness,sessions) if status=='completed' else {**cell,'run_id':directory.name,'status':status,'scheduled':len(trace_for(cell,cfg)),'completed':0,'completion_rate':0,'correctness':correctness}
    save(directory/'summary.json',result)
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--config',type=Path,default=ROOT/'experiments/scenarios/compact-repair.json')
    p.add_argument('--output-dir',type=Path,required=True)
    p.add_argument('--resume',action='store_true')
    p.add_argument('--pilot',action='store_true')
    p.add_argument('--list',action='store_true')
    args=p.parse_args();cfg=json.loads(args.config.read_text());cells=matrix(cfg)
    if args.pilot:
        # Separate seed; outcomes never enter final comparisons.
        cells=[dict(case='peer',seed=9900,load='high',policy=policy,grace=3,outage='persistent',delay=0,experiments=['pilot']) for policy in ('N','OFF','S','F25','F50','L')]
    if args.list:
        print(json.dumps({'unique_runs':len(cells),'observation_minutes':len(cells)*cfg['horizon_s']/60,'cells':cells},indent=2));return
    out=args.output_dir
    if out.exists() and any(out.iterdir()) and not args.resume:p.error('output must be new or empty, or use --resume')
    if not (out/'environment-and-provenance.json').exists():provenance(out,cfg)
    else:
        assert json.loads((out/'study-config.json').read_text())==cfg,'cannot resume with a different configuration'
        old=json.loads((out/'environment-and-provenance.json').read_text())['binaries']
        assert all(hashlib.sha256((ROOT/'bin'/n).read_bytes()).hexdigest()==h for n,h in old.items()),'binary changed'
    save(out/'matrix.json',cells)
    began=time.monotonic()
    with (out/'runs.jsonl').open('a',buffering=1) as f:
        completed={json.loads(l)['run_id'] for l in (out/'runs.jsonl').read_text().splitlines()}
        for i,cell in enumerate(cells):
            if run_identity(cell) in completed:continue
            if time.monotonic()-began>cfg['time_budget_s']:break
            row=run_cell(out,cfg,cell);f.write(json.dumps(row)+'\n')
            print(f'{i+1}/{len(cells)} {row["run_id"]}: {row["status"]}, completion={row.get("completion_rate")}, recovery={row.get("recovery_s")}',flush=True)
            subprocess.run([sys.executable,str(ROOT/'experiments/analyze_repair_playback.py'),str(out)],check=True)
    print(out,flush=True)

if __name__=='__main__':main()
