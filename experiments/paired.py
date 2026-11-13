#!/usr/bin/env python3
"""TODO(Yuquan): Extend controlled cold-start comparisons with warm-up and timed fault traces."""
import argparse
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import subprocess
import sys
import time
import tarfile

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'tests/system'))
from cluster import Cluster,ing

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--scenario',type=Path,default=ROOT/'experiments/scenarios/main.json')
    parser.add_argument('--output-dir',type=Path,required=True)
    parser.add_argument('--seeds',help='comma-separated override for diagnostic runs')
    parser.add_argument('--groups',help='comma-separated group override')
    args=parser.parse_args();cfg=json.loads(args.scenario.read_text())
    seeds=[int(s) for s in args.seeds.split(',')] if args.seeds else cfg['seeds']
    groups=args.groups.split(',') if args.groups else cfg['groups']
    for group in groups:
        if group not in ('fixed-round_robin','fixed-deadline','adaptive-round_robin','adaptive-deadline'):parser.error('unsupported group')
    output=args.output_dir
    if output.exists() and any(output.iterdir()):parser.error('output must be new or empty; never mix runs')
    output.mkdir(parents=True,exist_ok=True)
    environment={'commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
                 'dirty':bool(subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True).strip()),
                 'platform':platform.platform(),'python':sys.version,
                 'go':subprocess.check_output(['go','version'],text=True).strip()}
    (output/'environment.json').write_text(json.dumps(environment,indent=2))
    # A dirty commit identifier alone cannot reproduce local, uncommitted code.
    files=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z'],cwd=ROOT).decode().split('\0')
    fingerprints={}
    with tarfile.open(output/'source-snapshot.tar.gz','w:gz') as archive:
        for name in sorted(set(files)):
            path=ROOT/name
            if not name or not path.is_file() or path.is_symlink():continue
            fingerprints[name]=hashlib.sha256(path.read_bytes()).hexdigest()
            archive.add(path,arcname=name,recursive=False)
    (output/'source-sha256.json').write_text(json.dumps(fingerprints,indent=2))
    (output/'binary-sha256.json').write_text(json.dumps({name:hashlib.sha256((ROOT/'bin'/name).read_bytes()).hexdigest() for name in ['controller','peer','client']},indent=2))
    (output/'config.json').write_text(json.dumps({**cfg,'seeds':seeds,'groups':groups},indent=2))
    with (output/'sessions.jsonl').open('w') as sessions,(output/'runs.jsonl').open('w') as runs:
        for seed in seeds:
            rng=random.Random(seed)
            trace=[{'viewer':i,'video':f'video-{rng.randrange(cfg["videos"])}','arrival_ms':rng.randrange(cfg['arrival_span_ms']+1)} for i in range(cfg['viewers'])]
            trace.sort(key=lambda item:(item['arrival_ms'],item['viewer']))
            (output/f'trace-{seed}.json').write_text(json.dumps(trace,indent=2))
            # Alternate order to reduce drift systematically favoring one policy.
            ordered=list(groups)
            if seed%2:ordered.reverse()
            for group in ordered:
                policy,strategy=group.split('-',1);run_id=f'{group}-{seed}'
                with Cluster(policy=policy,controllers=3,repair=True,extra=cfg['extra_replica_bytes'],rates=cfg['peer_upload_caps']) as cluster:
                    manifests=[cluster.fixture(count=cfg['segments'],duration=cfg['segment_duration_ms'],size=cfg['segment_bytes'],video=f'video-{i}') for i in range(cfg['videos'])]
                    initial=[ing.request(p['url']+'/api/v1/stats') for p in cluster.peers]
                    logical_bytes=sum(s['size_bytes'] for m in manifests for s in m['segments'])
                    expected=2*logical_bytes+cfg['videos']*min(cfg['extra_replica_bytes']//cfg['segment_bytes'],cfg['segments'])*cfg['segment_bytes']
                    assert sum(p['stored_bytes'] for p in initial)==expected,'mismatched initial replica budget'
                    (output/f'{run_id}-manifests.json').write_text(json.dumps(manifests,indent=2))
                    started=time.monotonic()
                    def viewer(item):
                        time.sleep(max(0,started+item['arrival_ms']/1000-time.monotonic()))
                        command=[str(ROOT/'bin/client'),'-controllers',','.join(cluster.controllers),'-video',item['video'],'-strategy',strategy,'-window',str(cfg['window']),'-concurrency',str(cfg['concurrency']),'-startup-buffer-ms',str(cfg['startup_buffer_ms']),'-timeout',str(cfg['timeout_seconds'])+'s']
                        try:
                            result=subprocess.run(command,cwd=ROOT,text=True,capture_output=True,timeout=cfg['timeout_seconds']+10)
                            events=result.stderr
                            try:summary=json.loads(result.stdout)
                            except json.JSONDecodeError:summary={'status':'failed','error':events}
                        except subprocess.TimeoutExpired as error:summary={'status':'failed','error':'CLI timeout'};events=str(error)
                        (output/f'{run_id}-viewer-{item["viewer"]}.events.jsonl').write_text(events)
                        return {**summary,'run_id':run_id,'group':group,'seed':seed,'trace':item,'formal_experiment':False,'controlled_experiment':True}
                    with concurrent.futures.ThreadPoolExecutor(max_workers=cfg['viewers']) as pool:rows=list(pool.map(viewer,trace))
                    final=[ing.request(p['url']+'/api/v1/stats') for p in cluster.peers]
                    maintenance=cluster.call('/maintenance')
                    for row in rows:sessions.write(json.dumps(row)+'\n')
                    sessions.flush()
                    completed=[row for row in rows if row['status']=='completed']
                    metrics={'run_id':run_id,'group':group,'seed':seed,'trace_sha256':hashlib.sha256(json.dumps(trace,sort_keys=True).encode()).hexdigest(),
                             'sessions':len(rows),'completed':len(completed),'completion_rate':len(completed)/len(rows),
                             'mean_completed_rebuffering_ratio':sum(row['rebuffering_ratio'] for row in completed)/len(completed) if completed else None,
                             'stored_bytes_peak':max(sum(p['stored_bytes'] for p in initial),sum(p['stored_bytes'] for p in final)),
                             'uploaded_bytes':sum(b['uploaded_bytes']-a['uploaded_bytes'] for a,b in zip(initial,final)),
                             'repair_tasks':maintenance['tasks'],'initial_peers':initial,'final_peers':final,
                             'controlled_experiment':True,'formal_experiment':False}
                    runs.write(json.dumps(metrics)+'\n');runs.flush();print(run_id,metrics['completion_rate'],flush=True)
    print(output/'runs.jsonl')

if __name__=='__main__':main()
