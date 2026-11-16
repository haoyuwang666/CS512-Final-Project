#!/usr/bin/env python3
"""TODO(team): Audit and summarize the late-arrival study with per-cohort outcomes."""
import csv
import json
import sys
from collections import Counter
from pathlib import Path
from statistics import mean


def read(path):
    return [json.loads(x) for x in path.read_text().splitlines()]


def main():
    root = Path(sys.argv[1])
    runs = read(root/'runs.jsonl')
    replacements = json.loads((root/'replacements.json').read_text()) if (root/'replacements.json').exists() else {}
    details, cohorts = [], []
    errors = Counter()
    for r in runs:
        d = root/r['run_id']
        if r['run_id'] in replacements:
            d = Path(replacements[r['run_id']]['directory'])
            r = json.loads((d/'summary.json').read_text())
        sessions = read(d/'sessions.jsonl')
        injections = read(d/'injections.jsonl')
        fault = next(x['observer_s'] for x in injections if x['kind']=='first_failure')
        events = read(d/'client-events.jsonl')
        assert r['status']=='completed' and r['injection_preconditions_met']
        assert len(sessions)==6 and len({s['viewer'] for s in sessions})==6
        assert all(s['arrival_s'] <= s['client_launch_s'] <= s['arrival_s']+1 and s['ended_s'] >= s['client_launch_s'] for s in sessions)
        assert r['unknown_copy_byte_seconds']==0
        affected = set(r['correctness']['affected'])
        snapshots = read(d/'tasks-and-copy-verification.jsonl')
        assert snapshots[-1]['observer_s'] >= 97, 'truncated observation'
        assert all(a['observer_s']<=b['observer_s'] for a,b in zip(snapshots,snapshots[1:]))
        for cohort in range(3):
            group = [s for s in sessions if s['viewer']//2==cohort]
            start = min(s['client_launch_s'] for s in group)
            prior = [v for v in snapshots if v['observer_s'] <= start]
            state = prior[-1]['copies'] if prior else {}
            restored = bool(state) and all(state.get(s) is not None and len(state[s])>=2 for s in affected)
            # Count verified deliveries from holders absent from that segment's initial placement.
            replacement_reads = sum(1 for e in events if e.get('viewer',-1)//2==cohort
                and e['type']=='segment_verified' and e.get('segment_id') in affected
                and e.get('peer_id') not in {p['peer_id'] for p in r['correctness']['initial'][e['segment_id']]})
            cohorts.append(dict(run_id=r['run_id'],policy=r['policy'],seed=r['seed'],cohort=cohort,
                start_s=start,completed=sum(s['status']=='completed' for s in group),scheduled=2,
                mean_total_stall_s=mean(s.get('stall_time_ms',0)/1000 for s in group),
                restored_before_arrival=restored,replacement_segment_deliveries=replacement_reads))
        errors.update(s.get('error','') for s in sessions if s['status']!='completed')
        details.append(dict(run_id=r['run_id'],policy=r['policy'],seed=r['seed'],failed_peer=r['correctness']['failed_peer'],
            completed=r['completed'],scheduled=r['scheduled'],mean_post_fault_stall_s=r['mean_post_fault_stall_s_all_sessions'],
            recovery_lower_s=r['recovery_interval_lower_s'],recovery_s=r['recovery_s'],within_40s=r['recovery_target_met'],
            exposure_mib_s=r['one_copy_byte_seconds']/2**20,repair_mib=r['traffic_bytes']['repair']/2**20,
            fault_s=fault,latest_session_end_s=max(s['ended_s'] for s in sessions)))
    for name, rows in [('run-details.csv',details),('cohort-details.csv',cohorts)]:
        with (root/name).open('w') as f:
            w=csv.DictWriter(f,fieldnames=list(rows[0]));w.writeheader();w.writerows(rows)
    lines=['# Late-staggered E3 results','',
        'TODO(team): Keep these measured results linked to the frozen configuration and raw logs.','',
        f'Completed experimental runs: {len(runs)}/12. Real HTTP transfers with simulated consumption.',
        'Arrival pairs: 0/0.2, 20/20.2, 50/50.2 s. Peer failure at 5 s; grace 3 s; horizon 100 s.',
        'Three failed-capacity contexts per policy, one run per context. These are exploratory results, not repeated estimates within each capacity stratum.','',
        '| Policy | Completed | Mean post-fault stall/viewer (s) | Recovery by 40 s | Recovery range (s) | Exposure (MiB s) | Repair (MiB) |',
        '|---|---:|---:|---:|---:|---:|---:|']
    for policy in ['S','F25','F50','L']:
        rows=[r for r in details if r['policy']==policy]
        if not rows:continue
        rec=[r['recovery_s'] for r in rows if r['recovery_s'] is not None]
        span=f'{min(rec):.2f}–{max(rec):.2f}' if rec else 'censored'
        lines.append(f'| {policy} | {sum(r["completed"] for r in rows)}/{6*len(rows)} | {mean(r["mean_post_fault_stall_s"] for r in rows):.2f} | {sum(r["within_40s"] for r in rows)}/{len(rows)} | {span} | {mean(r["exposure_mib_s"] for r in rows):.2f} | {mean(r["repair_mib"] for r in rows):.3f} |')
    lines+=['','| Policy | Arrival cohort | Completed | Mean full-session stall (s) | Verified recovery before arrival | Replacement segment deliveries |','|---|---:|---:|---:|---:|---:|']
    for policy in ['S','F25','F50','L']:
        for cohort in range(3):
            rows=[r for r in cohorts if r['policy']==policy and r['cohort']==cohort]
            if rows:lines.append(f'| {policy} | {[0,20,50][cohort]} s | {sum(r["completed"] for r in rows)}/{2*len(rows)} | {mean(r["mean_total_stall_s"] for r in rows):.2f} | {sum(r["restored_before_arrival"] for r in rows)}/{len(rows)} | {sum(r["replacement_segment_deliveries"] for r in rows)} |')
    lines+=['','Recovery times are relative to failure; sampled verification brackets completion at approximately two-second intervals.',
        'Cohort stall values cover full sessions; the main table uses post-fault stalls. Failed sessions remain included.',
        'Replacement deliveries count segment_verified events from a Peer absent from that segment’s initial holders.',
        'This study changes arrival timing and observation horizon from the archived 0/4/12 study. Historical comparisons are descriptive, not a new paired causal test.',
        'There is no OFF arm in this E3 extension: it cannot establish the completion benefit of enabling repair versus disabling it.','',
        'Failure messages: '+json.dumps(errors,ensure_ascii=False), '', 'Documented replacement runs: '+json.dumps(replacements,ensure_ascii=False)]
    (root/'RESULTS.md').write_text('\n'.join(lines)+'\n')
    print('\n'.join(lines))


if __name__=='__main__':main()
