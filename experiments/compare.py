#!/usr/bin/env python3
"""TODO(Yuquan): Validate sensitivity to workload design before interpreting diagnostic confidence intervals."""
import argparse
import json
from pathlib import Path
import random
import statistics

def compare(path):
    rows=[json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    indexed={(row['group'],row['seed']):row for row in rows}
    if len(indexed)!=len(rows):raise ValueError('duplicate group/seed rows')
    output=[]
    for group in sorted({row['group'] for row in rows}-{'fixed-round_robin'}):
        seeds=sorted(seed for g,seed in indexed if g==group and ('fixed-round_robin',seed) in indexed)
        for seed in seeds:
            a,b=indexed[group,seed],indexed['fixed-round_robin',seed]
            if a['trace_sha256']!=b['trace_sha256'] or a['initial_peers'] and sum(p['stored_bytes'] for p in a['initial_peers'])!=sum(p['stored_bytes'] for p in b['initial_peers']):raise ValueError('unmatched trace or storage budget')
        for metric in ['completion_rate','mean_completed_rebuffering_ratio','uploaded_bytes']:
            differences=[indexed[group,seed][metric]-indexed['fixed-round_robin',seed][metric] for seed in seeds if indexed[group,seed][metric] is not None and indexed['fixed-round_robin',seed][metric] is not None]
            if not differences:continue
            interval=None
            if len(differences)>=2:
                rng=random.Random(512);samples=sorted(statistics.mean(rng.choices(differences,k=len(differences))) for _ in range(2000));interval=[samples[50],samples[1949]]
            output.append({'group':group,'baseline':'fixed-round_robin','metric':metric,'pairs':len(differences),'mean_difference':statistics.mean(differences),'bootstrap_95_percent_interval':interval})
    destination=path.with_name('paired-comparison.json');destination.write_text(json.dumps(output,indent=2)+'\n');return destination

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('runs',type=Path);print(compare(parser.parse_args().runs))
