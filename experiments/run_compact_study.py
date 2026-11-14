#!/usr/bin/env python3
"""Run the frozen serial E1-E5 study; incremental artifacts survive interruption."""
import json
from pathlib import Path
import shutil
import subprocess
import sys
import time
from study_common import ROOT, save

out=ROOT/'results/compact-study'
if out.exists():raise SystemExit('Output already exists; use the individual --resume commands after reviewing its configuration.')
gates=json.loads((ROOT/'results/compact-validation/gates.json').read_text())
if len(gates)!=9 or any(r['exit_code'] for r in gates):raise SystemExit('Correctness gates must all pass first')
started=time.time();wake=None
if shutil.which('caffeinate'):
    wake=subprocess.Popen(['caffeinate','-i','-w',str(__import__('os').getpid())],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
    subprocess.run([sys.executable,str(ROOT/'experiments/repair_playback_study.py'),'--output-dir',str(out)],cwd=ROOT,check=True)
    subprocess.run([sys.executable,str(ROOT/'experiments/controller_recovery_study.py'),'--output-dir',str(out)],cwd=ROOT,check=True)
    subprocess.run([sys.executable,str(ROOT/'experiments/analyze_repair_playback.py'),str(out)],cwd=ROOT,check=True)
    shutil.copytree(ROOT/'results/compact-validation',out/'validation',dirs_exist_ok=True)
    counts={name:len((out/name).read_text().splitlines()) for name in ('runs.jsonl','controller-runs.jsonl')}
    save(out/'execution.json',{'started_unix':started,'ended_unix':time.time(),'elapsed_s':time.time()-started,'run_counts':counts,
        'complete_matrix':counts['runs.jsonl']==72 and counts['controller-runs.jsonl']==9})
    print('ALL STUDY RUNS FINISHED',json.dumps(counts),f'elapsed={(time.time()-started)/60:.1f} min',flush=True)
finally:
    if wake is not None:wake.terminate();wake.wait()
