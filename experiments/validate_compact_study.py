#!/usr/bin/env python3
"""TODO(team): Maintain this experiment tool. Archive serial correctness gates before measured clusters start."""
import json
from pathlib import Path
import subprocess
import sys
import time
ROOT=Path(__file__).resolve().parents[1]
out=ROOT/'results/compact-validation';out.mkdir(exist_ok=True)
commands=[['go','vet','./...'],['go','test','-race','./...'],
    [sys.executable,'experiments/study_test.py'],
    [sys.executable,'tests/system/smoke.py'],[sys.executable,'tests/system/repair.py'],
    [sys.executable,'tests/system/cleanup.py'],[sys.executable,'tests/system/delete_video.py'],
    [sys.executable,'scripts/media/fragment_mp4_test.py'],[sys.executable,'tests/system/media.py']]
rows=[]
for i,cmd in enumerate(commands):
    name=f'{i+1:02d}-'+Path(cmd[1] if cmd[0]==sys.executable else cmd[0]+'-'+cmd[1]).stem
    started=time.monotonic()
    with (out/(name+'.log')).open('w') as f:
        res=subprocess.run(cmd,cwd=ROOT,stdout=f,stderr=subprocess.STDOUT,timeout=180)
    row={'command':cmd,'exit_code':res.returncode,'elapsed_s':time.monotonic()-started,'log':name+'.log'};rows.append(row)
    (out/'gates.json').write_text(json.dumps(rows,indent=2)+'\n')
    print(name,'PASS' if res.returncode==0 else 'FAIL',flush=True)
    if res.returncode:sys.exit(res.returncode)
