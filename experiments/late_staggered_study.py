#!/usr/bin/env python3
"""TODO(team): Reproduce the late-arrival E3 extension without replacing compact-study evidence."""
import argparse
import hashlib
import json
import random
from pathlib import Path
from repair_playback_study import run_cell, trace_for
from study_common import ROOT, provenance, run_identity, save


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output-dir', type=Path, required=True)
    parser.add_argument('--resume', action='store_true')
    parser.add_argument('--seed', type=int, help='Run a specified context for a documented replacement.')
    parser.add_argument('--policy', choices=['S','F25','F50','L'])
    args = parser.parse_args()
    cfg = json.loads((ROOT/'experiments/scenarios/late-staggered-e3.json').read_text())
    assert max(cfg['staggered_arrivals_s']) + cfg['session_timeout_s'] < cfg['horizon_s']
    cells = [dict(case='peer', seed=seed, load='burst', policy=policy, grace=3,
                  outage='persistent', delay=0, experiments=['E3-late'])
             for seed in cfg['seeds'] for policy in ('S', 'F25', 'F50', 'L')]
    cells = [c for c in cells if (args.seed is None or c['seed']==args.seed) and (args.policy is None or c['policy']==args.policy)]
    random.Random(cfg['treatment_order_seed']).shuffle(cells)
    out = args.output_dir
    if not args.resume:
        if out.exists() and any(out.iterdir()):
            parser.error('output must be new or empty')
        provenance(out, cfg)
        save(out/'matrix.json', cells)
    else:
        assert json.loads((out/'study-config.json').read_text()) == cfg
        env = json.loads((out/'environment-and-provenance.json').read_text())
        assert all(hashlib.sha256((ROOT/'bin'/n).read_bytes()).hexdigest() == h for n,h in env['binaries'].items())
    done = {json.loads(line)['run_id'] for line in (out/'runs.jsonl').read_text().splitlines()} if (out/'runs.jsonl').exists() else set()
    with (out/'runs.jsonl').open('a', buffering=1) as f:
        for i, cell in enumerate(cells):
            if run_identity(cell) in done:
                continue
            print(f'START {i+1}/{len(cells)} {run_identity(cell)}', flush=True)
            row = run_cell(out, cfg, cell)
            f.write(json.dumps(row)+'\n')
            print(f'END {i+1}/{len(cells)} status={row["status"]} completed={row.get("completed")}/6 recovery={row.get("recovery_s")}', flush=True)
            if row['status'] != 'completed':
                raise RuntimeError(f'Invalid experiment: {row.get("correctness")}')


if __name__ == '__main__':
    main()
