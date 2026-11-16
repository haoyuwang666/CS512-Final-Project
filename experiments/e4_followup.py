# TODO(Yuquan): Maintain reproducible E4 follow-up execution and evidence audits.
"""Run E4 alone, preserving the frozen study and all actual probe outcomes."""
import argparse
import csv
import hashlib
import json
import random
import time
from pathlib import Path

from repair_playback_study import run_cell
from study_common import ROOT, metadata, probe, provenance, run_identity, save


def completed_repair_gate(c, cfg, manifest, journal, correctness):
    """Positive control: require two hash-valid, registered copies of EVERY affected segment."""
    affected = set(correctness['affected'])
    segments = [s for s in manifest['segments'] if s['segment_id'] in affected]
    deadline = journal.epoch + cfg['horizon_s'] - 10
    while time.monotonic() < deadline:
        inventories = {}
        for peer in c.peers:
            res = probe(peer['url'] + '/api/v1/inventory')
            inventories[peer['peer_id']] = {
                r['segment_id']: r for r in res['body']['receipts']
            } if res['ok'] else {}
        copies = {}
        for seg in segments:
            loc = metadata(c, f'/segments/{seg["segment_id"]}/locations', journal)
            copies[seg['segment_id']] = [] if loc is None else [
                p['peer_id'] for p in loc['locations']
                if c.processes[p['peer_id']].poll() is None
                and (r := inventories.get(p['peer_id'], {}).get(seg['segment_id']))
                and r['size_bytes'] == seg['size_bytes'] and r['sha256'] == seg['sha256']
            ]
        ready = bool(copies) and all(len(v) >= 2 for v in copies.values())
        journal.write('completed-repair-control', {'ready': ready, 'copies': copies})
        if ready:
            correctness['completed_repair_gate_passed'] = True
            return
        time.sleep(.5)
    raise TimeoutError('Two verified, registered copies of every affected segment not observed')


def export(out, rows):
    flat = []
    for row in rows:
        d = out / row['run_id']
        injections = [json.loads(l) for l in (d/'injections.jsonl').read_text().splitlines()] if (d/'injections.jsonl').exists() else []
        first = next((e['observer_s'] for e in injections if e['kind']=='first_failure'), None)
        second = next((e['observer_s'] for e in injections if e['kind']=='second_failure'), None)
        evidence = row.get('correctness', {}).get('second_fault_delivery', {})
        flat.append(dict(run_id=row['run_id'], seed=row['seed'], policy=row['policy'],
            requested_delay_s=row['delay'], actual_delay_s=second-first if second is not None and first is not None else None,
            status=row['status'], hash_valid=evidence.get('hash_valid', False),
            committed=evidence.get('committed', False),
            success=evidence.get('hash_valid', False) and evidence.get('committed', False),
            injection_preconditions_met=row.get('injection_preconditions_met', False),
            completed_repair_gate_passed=row.get('correctness', {}).get('completed_repair_gate_passed', False),
            fresh_full_video_valid=row.get('correctness', {}).get('fresh_full_video_valid', False),
            completed_viewers=row.get('completed'), scheduled_viewers=row.get('scheduled')))
    if flat:
        with (out/'outcomes.csv').open('w', newline='') as f:
            writer=csv.DictWriter(f, fieldnames=list(flat[0]));writer.writeheader();writer.writerows(flat)


def fresh_full_video_check(c, cfg, manifest, journal, correctness):
    """Fetch all segments afresh from the sole survivor; do not rely on viewer buffers."""
    target=correctness['target']
    outcomes=[]
    for seg in manifest['segments']:
        res=probe(target['url']+f'/api/v1/segments/{seg["segment_id"]}',
            timeout=8,binary=True,headers={'X-Transfer-Class':'probe'})
        body=res.pop('body',b'')
        valid=res['ok'] and len(body)==seg['size_bytes'] and hashlib.sha256(body).hexdigest()==seg['sha256']
        loc=metadata(c,f'/segments/{seg["segment_id"]}/locations',journal)
        committed=loc is not None and any(p['peer_id']==target['peer_id'] for p in loc['locations'])
        outcomes.append({'segment_id':seg['segment_id'],'hash_valid':valid,'committed':committed,'probe':res})
        journal.write('fresh-full-video-probes',outcomes[-1])
    correctness['fresh_full_video_valid']=all(r['hash_valid'] and r['committed'] for r in outcomes)
    correctness['fresh_full_video_segments']=len(outcomes)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output-dir', type=Path, required=True)
    p.add_argument('--delays', type=float, nargs='+', default=[33,34,35,36,40])
    p.add_argument('--seeds', type=int, nargs='+', default=[1101,1102])
    p.add_argument('--repeats', type=int, default=1)
    p.add_argument('--control', action='store_true')
    p.add_argument('--resume', action='store_true')
    args=p.parse_args()
    cfg=json.loads((ROOT/'experiments/scenarios/compact-repair.json').read_text())
    cfg.update(name='e4-followup-v1', horizon_s=75 if args.control else 55,
        second_delays=args.delays, e4_seeds=args.seeds,
        followup_control=args.control, repeats=args.repeats,
        purpose='Exploratory fixed-delay sweep' if not args.control else 'Verified completed-repair positive control')
    out=args.output_dir
    if out.exists() and any(out.iterdir()) and not args.resume:
        p.error('output must be new or empty, or use --resume')
    if (out/'study-config.json').exists():
        assert json.loads((out/'study-config.json').read_text())==cfg
        binaries=json.loads((out/'environment-and-provenance.json').read_text())['binaries']
        assert all(hashlib.sha256((ROOT/'bin'/name).read_bytes()).hexdigest()==digest
            for name,digest in binaries.items()), 'cannot resume with different binaries'
    else:
        provenance(out,cfg)
    cells=[]
    for repeat in range(1,args.repeats+1):
        for seed in args.seeds:
            for policy in ['S','F50','L']:
                for delay in ([1] if args.control else args.delays):
                    cells.append(dict(case=f'e4-{repeat}', seed=seed, load='high', policy=policy,
                        grace=cfg['main_grace'], outage='persistent', delay=delay, experiments=['E4-followup']))
    random.Random(20261115).shuffle(cells)
    save(out/'matrix.json',cells)
    rows=[json.loads(l) for l in (out/'runs.jsonl').read_text().splitlines()] if (out/'runs.jsonl').exists() else []
    done={r['run_id'] for r in rows}
    with (out/'runs.jsonl').open('a',buffering=1) as f:
        for i,cell in enumerate(cells,1):
            if run_identity(cell) in done: continue
            print(f'START {i}/{len(cells)} {run_identity(cell)}',flush=True)
            row=run_cell(out,cfg,cell,completed_repair_gate if args.control else None,
                fresh_full_video_check if args.control else None)
            f.write(json.dumps(row)+'\n');rows.append(row);export(out,rows)
            e=row.get('correctness',{}).get('second_fault_delivery',{})
            print(f'DONE {i}/{len(cells)} {row["run_id"]} status={row["status"]} hash={e.get("hash_valid")} committed={e.get("committed")}',flush=True)


if __name__=='__main__':
    main()
