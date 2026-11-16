# TODO(Yuquan): Maintain reproducible E4 follow-up execution and evidence audits.
"""Audit and summarize every follow-up trial without pooling exploration and validation."""
import argparse
import csv
import hashlib
import json
from pathlib import Path


def analyze(root):
    rows=[]
    audit=[]
    binaries=None
    for phase in ['scan','validation','completed-control']:
        folder=root/phase
        if not (folder/'runs.jsonl').exists():
            continue
        env=json.loads((folder/'environment-and-provenance.json').read_text())
        if binaries is None:
            binaries=env['binaries']
        assert binaries==env['binaries'], 'binary mismatch between phases'
        trials=[json.loads(l) for l in (folder/'runs.jsonl').read_text().splitlines()]
        assert len({r['run_id'] for r in trials})==len(trials), 'duplicate trial'
        for r in trials:
            d=folder/r['run_id']
            assert json.loads((d/'summary.json').read_text())==r
            injections=[json.loads(l) for l in (d/'injections.jsonl').read_text().splitlines()]
            first=next(e['observer_s'] for e in injections if e['kind']=='first_failure')
            second=next(e['observer_s'] for e in injections if e['kind']=='second_failure')
            evidence=r['correctness'].get('second_fault_delivery',{})
            pre=json.loads((d/'second-fault-precondition.jsonl').read_text().splitlines()[0])
            sid=r['correctness']['sentinel']['segment_id']
            target=r['correctness']['target']['peer_id']
            receipt=pre['target_receipt']
            seg=r['correctness']['sentinel']
            ready_bytes=receipt['ok'] and receipt['body']['sha256']==seg['sha256'] and receipt['body']['size_bytes']==seg['size_bytes']
            ready_registration=any(p['peer_id']==target for p in (pre.get('registration') or {}).get('locations',[]))
            valid=r['status']=='completed' and r['injection_preconditions_met']
            row=dict(phase=phase,run_id=r['run_id'],seed=r['seed'],policy=r['policy'],
                delay_s=r['delay'] if phase!='completed-control' else 'completion-gated',
                actual_delay_s=second-first,valid_run=valid,
                target_ready_before_fault=ready_bytes and ready_registration,
                hash_valid=evidence.get('hash_valid',False),committed=evidence.get('committed',False),
                success=evidence.get('hash_valid',False) and evidence.get('committed',False),
                full_video_valid=r['correctness'].get('fresh_full_video_valid',False),
                full_video_segments=r['correctness'].get('fresh_full_video_segments',0))
            rows.append(row)
            if phase=='completed-control':
                probes=[json.loads(l) for l in (d/'fresh-full-video-probes.jsonl').read_text().splitlines()]
                assert len(probes)==len({x['segment_id'] for x in probes})==9
                assert row['full_video_valid']==all(x['hash_valid'] and x['committed'] for x in probes)
                assert r['correctness'].get('completed_repair_gate_passed')
            audit.append({'phase':phase,'run_id':r['run_id'],'valid_run':valid,
                'summary_matches_raw':True,'before_fault_receipt_status':receipt['status'],
                'post_fault_status':evidence.get('probe',{}).get('status')})
    if not rows:
        raise ValueError('no trials')
    with (root/'all-outcomes.csv').open('w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=list(rows[0]));w.writeheader();w.writerows(rows)
    aggregates=[]
    for phase in ['scan','validation','completed-control']:
        delays=sorted({r['delay_s'] for r in rows if r['phase']==phase},key=str)
        for delay in delays:
            for policy in ['S','F50','L']:
                rs=[r for r in rows if r['phase']==phase and r['delay_s']==delay and r['policy']==policy]
                aggregates.append(dict(phase=phase,delay_s=delay,policy=policy,runs=len(rs),
                    valid_runs=sum(r['valid_run'] for r in rs),successes=sum(r['valid_run'] and r['success'] for r in rs),
                    full_video_successes=sum(r['valid_run'] and r['full_video_valid'] for r in rs)))
    with (root/'aggregate.csv').open('w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=list(aggregates[0]));w.writeheader();w.writerows(aggregates)
    (root/'audit.json').write_text(json.dumps({'trials':len(rows),'all_valid':all(r['valid_run'] for r in rows),
        'identical_binaries_across_phases':True,'binary_sha256':binaries,'checks':audit},indent=2)+'\n')
    hashes=[]
    for p in sorted(root.rglob('*')):
        if p.is_file() and p.name!='SHA256SUMS':
            hashes.append(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.relative_to(root).as_posix())
    (root/'SHA256SUMS').write_text('\n'.join(hashes)+'\n')
    print(json.dumps(aggregates,indent=2))


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('directory',type=Path)
    analyze(p.parse_args().directory)
