#!/usr/bin/env python3
"""E5: direct timed application probes, committed task continuity, quorum refusal."""
import argparse
import hashlib
import json
import subprocess
import threading
import time
from pathlib import Path
from study_common import *


def application_until(c,journal,path,deadline,checks=None):
    first=None
    while time.monotonic()<deadline:
        # Probe each endpoint directly, without hidden retries or redirection.
        for base in c.controllers:
            role=probe(base+'/api/v1/cluster',timeout=.25)
            journal.write('roles',role)
            res=probe(base+'/api/v1'+path,timeout=.35)
            row=journal.write('metadata-probes',res)
            if res['ok'] and first is None:first=row['observer_s']
        if first is not None:return first
        time.sleep(.1)
    return None


def current_leader(c):
    info=metadata(c,'/maintenance')
    if info is None:raise RuntimeError('no usable metadata leader')
    for m in c.members:
        res=probe(m['http_url']+'/api/v1/cluster')
        if res['ok'] and res['body']['state']=='Leader':return m
    raise RuntimeError('leader not observed')


def run_case(out,cfg,case,seed):
    directory=out/f'{case}-{seed}';directory.mkdir()
    journal=Journal(directory,{'run_id':directory.name,'case':case,'seed':seed,'policy':'L' if case=='E5b' else 'S'})
    save(directory/'config.json',{**cfg,'case':case,'seed':seed})
    c=fresh_cluster({**cfg,'rates':[v/3 for v in cfg['rates']]} if case=='E5b' else cfg,'L' if case=='E5b' else 'S',grace=1)
    checks={};rows=[];threads=[];stop=threading.Event();fault=None;app=None;status='setup_failed'
    try:
        c.__enter__()
        manifest=c.fixture(count=9 if case=='E5a' else 1,duration=2000,size=cfg['segment_bytes'],video=f'control-{seed}')
        cold=c.fixture(count=1,duration=1000,size=4096,video=f'cold-{seed}')
        save(directory/'manifest.json',manifest)
        path=f'/videos/{manifest["video_id"]}/manifest'
        assert metadata(c,path)==manifest
        pending=None
        if case=='E5b':
            sid=manifest['segments'][0]['segment_id'];loc=metadata(c,f'/segments/{sid}/locations')['locations']
            c.stop(loc[0]['peer_id']);journal.write('injections',{'kind':'first_peer_failure','node':loc[0]['peer_id']})
            deadline=time.monotonic()+18
            while time.monotonic()<deadline and pending is None:
                tasks=metadata(c,'/maintenance',journal)
                if tasks:
                    pending=next((t for t in (tasks.get('tasks') or {}).values() if t['state']=='pending'),None)
                time.sleep(.1)
            checks['pending_precondition']=pending is not None
            if pending is None:
                status='precondition_missed';return {'case':case,'seed':seed,'run_id':directory.name,'status':status,'checks':checks,'application_recovery_s':None}
            save(directory/'pending-task.json',pending)
        if case=='E5a':
            journal.epoch=time.monotonic()
            t=threading.Thread(target=start_viewer,args=(c,cfg,manifest['video_id'],{'viewer':'warm','arrival_s':0},journal,rows,stop));t.start();threads.append(t)
            time.sleep(4)
        leader=current_leader(c)
        if case in ('E5a','E5b'):
            c.stop(leader['id']);fault=journal.write('injections',{'kind':'leader_failure','node':leader['id'],'precondition_satisfied':True})['observer_s']
            if case=='E5a':
                t=threading.Thread(target=start_viewer,args=(c,cfg,cold['video_id'],{'viewer':'cold','arrival_s':0},journal,rows,stop));t.start();threads.append(t)
            app=application_until(c,journal,path,time.monotonic()+12)
            checks['fresh_application_recovered']=app is not None
            replacement=current_leader(c)
            checks['replacement_leader']=replacement['id']!=leader['id']
            if case=='E5b':
                completed=None;deadline=time.monotonic()+30
                while time.monotonic()<deadline:
                    state=metadata(c,'/maintenance',journal)
                    journal.write('tasks-and-copy-verification',{'state':state})
                    task=(state.get('tasks') or {}).get(pending['task_id']) if state else None
                    if task and task['state']=='completed':completed=task;break
                    if task and task['state']=='failed':break
                    time.sleep(.2)
                checks['same_task_completed']=completed is not None
                checks['committed_age_preserved']=completed is not None and completed.get('created_unix_ms')==pending.get('created_unix_ms') and pending.get('created_unix_ms',0)>0
                sid=pending['segment_id'];target=pending['target']
                res=probe(target['url']+f'/api/v1/segments/{sid}',binary=True,timeout=8,headers={'X-Transfer-Class':'probe'})
                data=res.pop('body',b'');journal.write('target-delivery',res)
                checks['verified_target_bytes']=res['ok'] and len(data)==pending['size_bytes'] and hashlib.sha256(data).hexdigest()==pending['sha256']
                loc=metadata(c,f'/segments/{sid}/locations',journal)
                checks['target_registration']=loc is not None and any(p['peer_id']==target['peer_id'] for p in loc['locations'])
                # Restart the interrupted controller with its original config;
                # quorum-backed task identity/age must remain present.
                c.start(leader['id']);time.sleep(.5)
                replay=metadata(c,'/maintenance',journal)
                checks['task_persists_after_restart']=replay is not None and (replay.get('tasks') or {}).get(pending['task_id'],{}).get('created_unix_ms')==pending.get('created_unix_ms')
                source=pending['source'];before=probe(source['url']+'/api/v1/stats')
                c.stop(source['peer_id']);c.start(source['peer_id'])
                wait(lambda:probe(source['url']+'/api/v1/stats')['ok'],timeout=6)
                after=probe(source['url']+'/api/v1/stats')
                journal.write('policy-restart',{'before':before,'after':after})
                checks['L_policy_and_counter_instance_restart']=before['ok'] and after['ok'] and after['body']['repair_policy']=='L' and after['body']['instance_id']!=before['body']['instance_id']
        else:
            # Leave the actual old leader alone; both followers stop. Wait for
            # stable isolation before submitting valid uniquely identified work.
            stopped=[m for m in c.members if m['id']!=leader['id']]
            for m in stopped:c.stop(m['id'])
            fault=journal.write('injections',{'kind':'majority_loss','nodes':[m['id'] for m in stopped],'precondition_satisfied':True})['observer_s']
            time.sleep(2.5)
            sid=f'isolated-{seed}--000000';raw=b'x'
            m={'schema_version':1,'video_id':f'isolated-{seed}','representation':'synthetic-bytes-v1','segments':[{'segment_id':sid,'index':0,'duration_ms':1000,'size_bytes':1,'sha256':hashlib.sha256(raw).hexdigest()}]}
            command={'operation_id':f'isolated-create-{seed}','manifest':m};attempts=[]
            for _ in range(3):
                res=probe(leader['http_url']+'/api/v1/videos','POST',command,timeout=1)
                journal.write('isolation-writes',res);attempts.append(res);time.sleep(.2)
            checks['no_success_ack_during_isolation']=all(not r['ok'] for r in attempts)
            restore=journal.write('injections',{'kind':'restore_quorum','requested_s':time.monotonic()-journal.epoch})['observer_s']
            for member in stopped:c.start(member['id'])
            app=application_until(c,journal,path,time.monotonic()+15)
            checks['baseline_state_retained']=metadata(c,path,journal)==manifest
            checks['fresh_application_recovered']=app is not None
            # Retry exactly the original operation; ambiguous requests are not
            # relabeled as lost. A post-restore commit is permitted.
            deadline=time.monotonic()+8;reconciled=False
            while time.monotonic()<deadline and not reconciled:
                for base in c.controllers:
                    res=probe(base+'/api/v1/videos','POST',command)
                    journal.write('reconciliation',res)
                    if res['ok']:reconciled=True;break
                time.sleep(.1)
            checks['original_operation_reconciled']=reconciled
            # plan is POST; verify actual application object existence.
            checks['reconciled_object_present']=any(probe(base+f'/api/v1/videos/{m["video_id"]}/placement-plan','POST',{})['ok'] for base in c.controllers)
            checks['quorum_restored']=app is not None
            save(directory/'quorum-timing.json',{'restore_s':restore,'isolation_duration_s':restore-fault})
        for t in threads:t.join(timeout=cfg['session_timeout_s']+3)
        if case=='E5a':
            checks['warm_completed']=any(r['viewer']=='warm' and r['status']=='completed' for r in rows)
            checks['cold_uncached_completed']=any(r['viewer']=='cold' and r['status']=='completed' for r in rows)
        status='passed' if all(checks.values()) else 'failed'
    except Exception as exc:
        checks['error']=f'{type(exc).__name__}: {exc}';status='failed'
    finally:
        stop.set()
        for t in threads:t.join(timeout=cfg['session_timeout_s']+3)
        archive_logs(c,directory);c.__exit__(None,None,None);journal.close()
        save(directory/'correctness.json',{'checks':checks,'status':status})
    result={'case':case,'seed':seed,'run_id':directory.name,'status':status,'checks':checks,
            'application_recovery_s':app-fault if app is not None and fault is not None else None,
            'sessions':rows,'formal_experiment':False}
    save(directory/'summary.json',result)
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output-dir',type=Path,required=True)
    p.add_argument('--config',type=Path,default=ROOT/'experiments/scenarios/compact-repair.json');p.add_argument('--resume',action='store_true');p.add_argument('--seeds');args=p.parse_args()
    cfg=json.loads(args.config.read_text());out=args.output_dir
    if not (out/'study-config.json').exists():provenance(out,cfg)
    assert cfg==json.loads((out/'study-config.json').read_text())
    fingerprints=json.loads((out/'environment-and-provenance.json').read_text())['binaries']
    assert all(hashlib.sha256((ROOT/'bin'/n).read_bytes()).hexdigest()==h for n,h in fingerprints.items()),'study binaries changed'
    prior=out/'controller-runs.jsonl'
    if prior.exists() and not args.resume:p.error('use --resume to continue controller runs')
    finished={r['run_id'] for r in [json.loads(l) for l in prior.read_text().splitlines()]} if prior.exists() else set()
    seeds=[int(s) for s in args.seeds.split(',')] if args.seeds else cfg['e5_seeds']
    with prior.open('a',buffering=1) as f:
        for seed in seeds:
            for case in ('E5a','E5b','E5c'):
                if f'{case}-{seed}' in finished:continue
                result=run_case(out,cfg,case,seed);f.write(json.dumps(result)+'\n');print(case,seed,result['status'],result['checks'],flush=True)
                subprocess.run([sys.executable,str(ROOT/'experiments/analyze_repair_playback.py'),str(out)],check=True)

if __name__=='__main__':main()
