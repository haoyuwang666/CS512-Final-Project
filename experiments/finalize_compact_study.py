#!/usr/bin/env python3
"""TODO(team): Maintain this experiment tool. Final audit and slide-focused exhibits. Raw event journals are never changed."""
import csv
import hashlib
import json
import os
from collections import defaultdict
from pathlib import Path
import statistics
import subprocess
import sys
os.environ.setdefault('MPLCONFIGDIR','/tmp/cs512-matplotlib')
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

ROOT=Path(__file__).resolve().parents[1]
out=Path(sys.argv[1]) if len(sys.argv)>1 else ROOT/'results/compact-study'
cfg=json.loads((out/'study-config.json').read_text())
def read(p):return [json.loads(l) for l in p.read_text().splitlines()] if p.exists() else []
def save(p,v):p.write_text(json.dumps(v,indent=2)+'\n')
rows=read(out/'runs.jsonl');controllers=read(out/'controller-runs.jsonl')
if not (out/'runs.observed.jsonl').exists():
    (out/'runs.observed.jsonl').write_bytes((out/'runs.jsonl').read_bytes())
changes=[];audit=[]
if not (out/'controller-runs.observed.jsonl').exists():
    (out/'controller-runs.observed.jsonl').write_bytes((out/'controller-runs.jsonl').read_bytes())
for r in controllers:
    if r['case']=='E5c' and r.get('application_recovery_s') is not None:
        timing=json.loads((out/r['run_id']/'quorum-timing.json').read_text())
        r['application_recovery_after_quorum_restore_s']=r['application_recovery_s']-timing['isolation_duration_s']
(out/'controller-runs.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in controllers))
for r in rows:
    directory=out/r['run_id'];traffic=read(directory/'peer-traffic-and-policy.jsonl')
    instances=defaultdict(lambda:defaultdict(int));policy_errors=[];last_counter_s=0
    for t in traffic:
        if not t.get('ok'):continue
        b=t['body'];last_counter_s=max(last_counter_s,t['observer_s'])
        for cl,n in b.get('class_bytes',{}).items():
            key=(t['peer_id'],b['instance_id']);instances[key][cl]=max(instances[key][cl],n)
        for h in b.get('policy_history',[]):
            f=h['repair_bytes_per_second']/b['upload_bytes_per_second']
            pol=b['repair_policy'];expected={'S':1,'F25':.25,'F50':.5}.get(pol)
            bad=abs(f-expected)>1e-8 if expected is not None else f not in (.25,.5,1)
            if pol=='L' and h['active_foreground']>0:
                expected=.5 if h['oldest_repair_age_ms']>60000 else .25
                bad=bad or abs(f-expected)>1e-8
            if bad:policy_errors.append(h)
    totals={cl:sum(m.get(cl,0) for m in instances.values()) for cl in ('repair','foreground','probe')}
    if totals!=r.get('traffic_bytes'):
        changes.append({'run_id':r['run_id'],'before':r.get('traffic_bytes'),'after':totals,
            'reason':'Use the monotonic maximum for each class/instance, robust to response/journal ordering.'})
        r['traffic_bytes']=totals
    if r['policy']=='N':r['recovery_target_met']=None
    r['last_traffic_counter_observed_s']=last_counter_s
    r['policy_sample_rule_violations']=len(policy_errors)
    fault=next((e['observer_s'] for e in read(directory/'injections.jsonl') if e['kind'] in ('first_failure','no_fault_marker')),None)
    repair_samples=[e['observer_s'] for e in traffic if e.get('ok') and e['body'].get('class_bytes',{}).get('repair',0)>0]
    r['first_repair_payload_observed_after_fault_s']=min(repair_samples)-fault if repair_samples and fault is not None else None
    tasks=read(directory/'tasks.jsonl');first_task=next((e['observer_s'] for e in tasks if e.get('state') and e['state'].get('tasks')),None)
    r['first_task_observed_after_fault_s']=first_task-fault if first_task is not None and fault is not None else None
    if r.get('recovery_s') is not None:
        restored=next((e for e in read(directory/'tasks-and-copy-verification.jsonl') if e['observer_s']>=fault+r['recovery_s']-.0001),None)
        r['recovery_mechanism']='original_return' if restored and all(r['correctness']['failed_peer'] in (restored['copies'].get(s) or []) for s in r['correctness']['affected']) else 'replacement'
    else:r['recovery_mechanism']=None
    sessions=read(directory/'sessions.jsonl')
    audit.append({'run_id':r['run_id'],'status':r['status'],'injections_valid':r.get('injection_preconditions_met'),
        'post_fault_exposure':r.get('post_fault_exposure'),'all_scheduled_outcomes_present':len(sessions)==r.get('scheduled'),
        'policy_sample_rule_violations':len(policy_errors),'counter_instances':len(instances),
        'traffic_endpoint_s':last_counter_s,
        'all_session_outcomes_by_horizon':all(v['ended_s']<=cfg['horizon_s'] for v in sessions),
        'last_session_terminal_s':max((v['ended_s'] for v in sessions),default=None)})
    save(directory/'summary.json',r)
(out/'runs.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
save(out/'analysis-audit.json',{'performance_runs':len(rows),'controller_runs':len(controllers),'runs':audit,
    'counter_ordering_corrections':changes,'postprocessing_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
    'raw_journals_unchanged':True,'matplotlib_version':matplotlib.__version__})
subprocess.run([sys.executable,str(ROOT/'experiments/analyze_repair_playback.py'),str(out)],check=True)
figdir=out/'figures';plt.rcParams.update({'font.size':11,'axes.spines.top':False,'axes.spines.right':False})
colors={'N':'#64748b','OFF':'#e69f00','S':'#d55e00','F25':'#56b4e9','F50':'#0072b2','L':'#009e73'}
footer=f"Exploratory • paired seeds {','.join(map(str,cfg['seeds']))} • 3 Controllers / 3 Peers • real HTTP, simulated consumption\n50 s nominal horizon; 2 s verification cadence; payload counters include retries; see raw journals and provenance."
def finish(fig,name,title):
    fig.suptitle(title,fontweight='bold')
    note=(f"Exploratory stress test • two paired contexts per cell, seeds {','.join(map(str,cfg['e4_seeds']))} • real HTTP targeted delivery\n50 s horizon; late-queue sentinel; stopped disks retained; these counts are not real-world loss probabilities." if name.startswith('E4') else footer) if not name.startswith('E5') else 'Exploratory • 3 trials per case • direct one-attempt quorum-backed application probes\nE5a/b: time from Leader stop. E5c table: time from quorum restoration. Failures and retry attempts retained in raw journals.'
    fig.text(.02,.015,note,fontsize=8,color='#555');fig.tight_layout(rect=(0,.09,1,.92))
    for ext in ('png','svg'):fig.savefig(figdir/(name+'.'+ext),dpi=180)
    plt.close(fig)
def mean(v):return statistics.mean(v) if v else None

def table_figure(name,title,headers,data,widths=None):
    fig,ax=plt.subplots(figsize=(12,max(3.6,.44*len(data)+1.7)));ax.axis('off')
    table=ax.table(cellText=data,colLabels=headers,loc='center',cellLoc='center',colWidths=widths)
    table.auto_set_font_size(False);table.set_fontsize(10);table.scale(1,1.65)
    for (i,j),cell in table.get_celld().items():
        cell.set_edgecolor('#d1d5db')
        if i==0:
            cell.set_facecolor('#1e293b');cell.set_text_props(color='white',weight='bold',fontsize=9.5)
            cell.set_height(cell.get_height()*2)
        elif i%2==0:cell.set_facecolor('#f1f5f9')
    finish(fig,name,title)

e3=[]
for load in ('high','burst'):
    for pol in ('S','F25','F50','L'):
        rs=[r for r in rows if 'E3' in r['experiments'] and r['load']==load and r['policy']==pol]
        if not rs:continue
        recovered=[r['recovery_s'] for r in rs if r.get('recovery_s') is not None]
        e3.append([load,pol,f"{sum(r['completed'] for r in rs)}/{sum(r['scheduled'] for r in rs)}",
            f"{mean([r['mean_post_fault_stall_s_all_sessions'] for r in rs]):.2f}",
            f"{sum(bool(r['recovery_target_met']) for r in rs)}/{len(rs)}",
            f"{statistics.median(recovered):.2f}" if recovered else '—',
            str(sum(r['recovery_censored'] for r in rs)),f"{mean([r['one_copy_byte_seconds']/1048576 for r in rs]):.1f}"])
table_figure('E3-policy-table','E3 • Completion and recovery must be judged together',
    ['Demand','Policy','Viewers\ncompleted','Post-fault\nstall (s)','Target\nmet / runs','Median recovery\nobserved only (s)','Censored\nruns','Exposure\n(MiB·s)'],e3)

fig,axs=plt.subplots(1,2,figsize=(11,5),sharey=True)
for ax,load in zip(axs,('low','high')):
    for i,(a,b) in enumerate([('OFF','N'),('S','OFF'),('S','N')]):
        diffs=[]
        for seed in cfg['seeds']:
            pair={p:next((r for r in rows if 'E1' in r['experiments'] and r['load']==load and r['seed']==seed and r['policy']==p),None) for p in (a,b)}
            if all(pair.values()):diffs.append(pair[a]['mean_post_fault_stall_s_all_sessions']-pair[b]['mean_post_fault_stall_s_all_sessions'])
        ax.scatter([i+(k-(len(diffs)-1)/2)*.06 for k in range(len(diffs))],diffs,color='#0072b2',s=55)
        if diffs:ax.plot([i-.18,i+.18],[mean(diffs)]*2,color='#d55e00',lw=3)
    ax.axhline(0,color='#555',lw=1);ax.set(xticks=range(3),xticklabels=['OFF − N','S − OFF','S − N'],title=f'{load} demand',ylabel='Paired post-fault stall difference (s/viewer)');ax.grid(axis='y',alpha=.2)
finish(fig,'E1-paired-effects','E1 • Separate Peer-loss and repair effects (points = seeds)')

# Make E4 outcome counts legible even when all policies have the same result.
pols=['S',cfg['fixed_e4_policy'],'L'];values=[];labels=[]
for pol in pols:
    row=[];text=[]
    for delay in cfg['second_delays']:
        rs=[r for r in rows if 'E4' in r['experiments'] and r['policy']==pol and r['delay']==delay]
        n=sum(r['correctness']['second_fault_delivery']['hash_valid'] and r['correctness']['second_fault_delivery']['committed'] for r in rs)
        row.append(n/len(rs) if rs else 0);text.append(f'{n}/{len(rs)}')
    values.append(row);labels.append(text)
fig,ax=plt.subplots(figsize=(9,4.5));ax.imshow(values,vmin=0,vmax=1,cmap='RdYlGn',aspect='auto')
for i in range(len(pols)):
    for j in range(len(cfg['second_delays'])):ax.text(j,i,labels[i][j],ha='center',va='center',fontsize=20,color='white' if values[i][j]<.25 else '#111',fontweight='bold')
ax.set(xticks=range(len(cfg['second_delays'])),xticklabels=[f'{d} s' for d in cfg['second_delays']],yticks=range(3),yticklabels=pols,xlabel='Delay between first and second holder failure')
finish(fig,'E4-target-delivery-counts','E4 • Correct, committed replacement deliveries / attempts')

checks=[]
for case in ('E5a','E5b','E5c'):
    rs=[r for r in controllers if r['case']==case]
    vals=[r.get('application_recovery_after_quorum_restore_s',r['application_recovery_s']) for r in rs if r.get('application_recovery_s') is not None]
    checks.append([case,{'E5a':'Leader loss during delivery','E5b':'Leader loss during repair','E5c':'Majority loss / restoration'}[case],
        f"{sum(r['status']=='passed' for r in rs)}/{len(rs)}",f'{statistics.median(vals):.2f}' if vals else '—',
        f'{min(vals):.2f}–{max(vals):.2f}' if vals else '—'])
table_figure('E5-safety-table','E5 • Observed application recovery and safety checks',
    ['Case','Injection','Passes / attempts','Median metadata recovery\nafter fault / quorum restore (s)','Observed range (s)'],checks,[.07,.37,.16,.22,.18])

# Timeline shows service and playback separately, rather than treating a role
# announcement as proof of application availability.
a=next((r for r in controllers if r['case']=='E5a' and r['status']=='passed'),None)
if a:
    directory=out/a['run_id'];injections=read(directory/'injections.jsonl');fault=next(e['observer_s'] for e in injections if e['kind']=='leader_failure')
    events=read(directory/'client-events.jsonl');fig,ax=plt.subplots(figsize=(11,4))
    for i,viewer in enumerate(('warm','cold')):
        es=[e for e in events if e.get('viewer')==viewer]
        if es:
            start=es[0]['observer_s']-fault;end=es[-1]['observer_s']-fault
            ax.plot([start,end],[i,i],lw=9,color='#0072b2' if viewer=='warm' else '#009e73',solid_capstyle='round')
            ax.text(end+.3,i,viewer+' completed',va='center')
    ax.axvline(0,color='#d55e00',ls='--',label='actual Leader stop')
    ax.axvline(a['application_recovery_s'],color='#009e73',ls=':',label='first fresh quorum-backed response')
    ax.set_xlim(-5,max(e['observer_s']-fault for e in events)+4.5)
    ax.set(yticks=[0,1],yticklabels=['Warm session','Cold uncached session'],xlabel='Time relative to Leader failure (s)',ylim=(-.8,1.8));ax.legend(loc='upper left')
    finish(fig,'E5-annotated-timeline','E5a • Playback outcomes and fresh metadata recovery')

# Machine-readable enriched run metrics (postprocessing-only additions).
keys=['run_id','policy','seed','first_task_observed_after_fault_s','first_repair_payload_observed_after_fault_s','recovery_mechanism','last_traffic_counter_observed_s','policy_sample_rule_violations']
with (out/'measurement-audit.csv').open('w',newline='') as f:
    w=csv.DictWriter(f,fieldnames=keys);w.writeheader();w.writerows({k:r.get(k) for k in keys} for r in rows)
print('Final audit and slide tables saved:',out,flush=True)

all_sessions=[]
for r in rows:
    for session in read(out/r["run_id"]/"sessions.jsonl"):
        all_sessions.append({**session,"load":r["load"],"outage":r["outage"],"grace":r["grace"],"second_delay_s":r["delay"]})
if all_sessions:
    fields=list(dict.fromkeys(k for r in all_sessions for k in r))
    with (out/"sessions.csv").open("w",newline="") as f:
        w=csv.DictWriter(f,fieldnames=fields);w.writeheader();w.writerows(all_sessions)

for load in ('high','burst'):
    data=[]
    for peer in ('peer-1','peer-2','peer-3'):
        for pol in ('S','F25','F50','L'):
            r=next(r for r in rows if 'E3' in r['experiments'] and r['load']==load and r['policy']==pol and r['correctness']['failed_peer']==peer)
            lower=r.get('recovery_interval_lower_s');upper=r.get('recovery_s')
            data.append([peer,pol,f"{r['completed']}/{r['scheduled']}",f"{r['mean_post_fault_stall_s_all_sessions']:.2f}",
                f"({lower:.2f}, {upper:.2f}]" if upper is not None else 'censored',
                'yes' if r['recovery_target_met'] else 'no'])
    table_figure('E3-'+load+'-by-failed-peer','E3 • '+load+' demand: retain the failed-source context',
        ['Failed Peer','Policy','Viewers done','Observed stall\n(s / scheduled viewer)','Restoration interval\n(s after fault)','Verified by\n40 s'],data)
analysis_source=out/'analysis-source';analysis_source.mkdir(exist_ok=True)
for name in ('finalize_compact_study.py','analyze_repair_playback.py'):
    (analysis_source/name).write_bytes((ROOT/'experiments'/name).read_bytes())
