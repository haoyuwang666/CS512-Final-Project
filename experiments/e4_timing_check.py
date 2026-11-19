#!/usr/bin/env python3
"""TODO(team): Maintain the exploratory E4 timing follow-up without modifying the frozen study."""
import argparse
import csv
import hashlib
import json
from pathlib import Path
import random

from repair_playback_study import load_jsonl, run_cell
from study_common import ROOT, provenance, save


def timing_result(out, row):
    directory = out / row['run_id']
    correctness = row.get('correctness', {})
    sentinel = correctness.get('sentinel', {}).get('segment_id')
    target = correctness.get('target', {}).get('peer_id')
    injections = load_jsonl(directory / 'injections.jsonl')
    first = next((r['observer_s'] for r in injections if r['kind'] == 'first_failure'), None)
    second = next((r['observer_s'] for r in injections if r['kind'] == 'second_failure'), None)
    preconditions = load_jsonl(directory / 'second-fault-precondition.jsonl')
    pre = preconditions[0] if preconditions else {}
    registration = pre.get('registration')
    pre_committed = registration is not None and any(
        p['peer_id'] == target for p in registration['locations'])
    delivery = correctness.get('second_fault_delivery', {})
    snapshots = load_jsonl(directory / 'tasks-and-copy-verification.jsonl')
    ready = next((s['observer_s'] for s in snapshots
                  if first is not None and second is not None and first <= s['observer_s'] < second
                  and target in ((s.get('copies') or {}).get(sentinel) or [])), None)
    return {
        'run_id': row['run_id'], 'policy': row['policy'], 'seed': row['seed'],
        'scheduled_delay_s': row['delay'], 'status': row['status'],
        'actual_delay_s': second - first if first is not None and second is not None else None,
        'injection_preconditions_met': row.get('injection_preconditions_met', False),
        'pre_stop_receipt_status': pre.get('target_receipt', {}).get('status'),
        'pre_stop_committed': pre_committed,
        'first_sampled_sentinel_ready_after_first_fault_s': ready - first if ready is not None else None,
        'post_stop_probe_status': delivery.get('probe', {}).get('status'),
        'post_stop_hash_valid': delivery.get('hash_valid', False),
        'post_stop_committed': delivery.get('committed', False),
        'delivery_success': bool(delivery.get('hash_valid') and delivery.get('committed')),
        'viewers_completed': row.get('completed'), 'viewers_scheduled': row.get('scheduled'),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output-dir', type=Path, required=True)
    args = parser.parse_args()
    out = args.output_dir
    if out.exists():
        parser.error('output directory must be new')
    frozen = ROOT / 'results/compact-study'
    cfg = json.loads((frozen / 'study-config.json').read_text())
    cfg.update(name='e4-timing-followup-f50-1101', horizon_s=65, second_delays=[30, 40, 50],
               study_kind='exploratory timing follow-up, one context, not a policy comparison')
    original = json.loads((frozen / 'environment-and-provenance.json').read_text())['binaries']
    matches = {name: hashlib.sha256((ROOT / 'bin' / name).read_bytes()).hexdigest() == digest
               for name, digest in original.items()}
    if not all(matches.values()):
        parser.error('binaries differ from the frozen study; do not claim timing-only follow-up')
    cells = [dict(case='peer', seed=1101, load='high', policy='F50', grace=cfg['main_grace'],
                  outage='persistent', delay=delay, experiments=['E4-timing-followup'])
             for delay in cfg['second_delays']]
    random.Random(cfg['treatment_order_seed']).shuffle(cells)
    provenance(out, cfg)
    save(out / 'original-binary-comparison.json', matches)
    save(out / 'matrix.json', cells)
    results = []
    with (out / 'runs.jsonl').open('w', buffering=1) as stream:
        for i, cell in enumerate(cells, 1):
            print(f'START {i}/{len(cells)} F50 seed=1101 second fault +{cell["delay"]}s', flush=True)
            row = run_cell(out, cfg, cell)
            stream.write(json.dumps(row) + '\n')
            result = timing_result(out, row)
            results.append(result)
            results.sort(key=lambda r: r['scheduled_delay_s'])
            save(out / 'timing-summary.json', results)
            with (out / 'timing-summary.csv').open('w', newline='') as csv_stream:
                writer = csv.DictWriter(csv_stream, fieldnames=list(result))
                writer.writeheader()
                writer.writerows(results)
            print(json.dumps(result), flush=True)
            if row['status'] != 'completed':
                raise RuntimeError(f'run failed; inspect {out / row["run_id"]}')
    print(f'Finished: {out}', flush=True)


if __name__ == '__main__':
    main()
