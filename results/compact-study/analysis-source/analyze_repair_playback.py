#!/usr/bin/env python3
"""Rebuild slide exhibits from archived runs; failed/censored outcomes remain visible."""
import argparse
import csv
import json
from collections import defaultdict
from pathlib import Path
import statistics
import os
os.environ.setdefault('MPLCONFIGDIR','/tmp/cs512-matplotlib')
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

COLORS={'N':'#64748b','OFF':'#e69f00','S':'#d55e00','F25':'#56b4e9','F50':'#0072b2','L':'#009e73'}


def read(path):
    return [json.loads(l) for l in path.read_text().splitlines()] if path.exists() else []


def write_csv(path,rows):
    if not rows:return
    keys=list(dict.fromkeys(k for r in rows for k in r))
    with path.open('w',newline='') as f:
        writer=csv.DictWriter(f,fieldnames=keys);writer.writeheader();writer.writerows(rows)


def avg(values):
    values=[v for v in values if v is not None]
    return statistics.mean(values) if values else None


def main():
    p=argparse.ArgumentParser();p.add_argument('directory',type=Path);args=p.parse_args();out=args.directory
    rows=read(out/'runs.jsonl');cfg=json.loads((out/'study-config.json').read_text())
    figdir=out/'figures';figdir.mkdir(exist_ok=True)
    provenance=json.loads((out/'environment-and-provenance.json').read_text())
    binary=provenance['binaries']['peer'][:10]
    plt.rcParams.update({'font.size':11,'axes.spines.top':False,'axes.spines.right':False,'figure.dpi':110,'savefig.dpi':180})
    flat=[]
    for r in rows:
        flat.append({**{k:r.get(k) for k in ('run_id','experiments','case','seed','load','policy','grace','outage','delay','status','scheduled','completed','completion_rate','mean_startup_s_completed','mean_post_fault_stall_s_all_sessions','mean_rebuffering_ratio_completed','recovery_s','recovery_interval_lower_s','recovery_target_met','recovery_censored','censor_reason','one_copy_byte_seconds','zero_copy_byte_seconds','unknown_copy_byte_seconds','repair_tasks','failed_repair_tasks','post_fault_exposure','repair_foreground_overlap','injection_preconditions_met','metadata_probe_attempts','alternate_delivery_s')},
            **{k+'_bytes':v for k,v in r.get('traffic_bytes',{}).items()},
            'e4_correct_target':r.get('correctness',{}).get('second_fault_delivery',{}).get('hash_valid'),
            'e4_target_committed':r.get('correctness',{}).get('second_fault_delivery',{}).get('committed')})
    write_csv(out/'summary.csv',flat)
    groups=defaultdict(list)
    for r in rows:
        for e in r.get('experiments',[]):groups[(e,r['load'],r['policy'],r['grace'],r['outage'],r['delay'])].append(r)
    table=[]
    for (e,load,pol,g,off,delay),rs in sorted(groups.items()):
        good=[r for r in rs if r['status']=='completed'];recovered=[r for r in good if r.get('recovery_s') is not None]
        table.append({'experiment':e,'load':load,'policy':pol,'grace_s':g,'outage':off,'second_delay_s':delay,'runs':len(rs),'setup_failed':len(rs)-len(good),
            'sessions_completed':sum(r.get('completed',0) for r in rs),'sessions_scheduled':sum(r.get('scheduled',0) for r in rs),
            'completion_rate':sum(r.get('completed',0) for r in rs)/sum(r.get('scheduled',0) for r in rs) if sum(r.get('scheduled',0) for r in rs) else None,
            'mean_post_fault_stall_s_all':avg(r.get('mean_post_fault_stall_s_all_sessions') for r in good),
            'recovery_observed':len(recovered),'recovery_censored':sum(bool(r.get('recovery_censored')) for r in good),
            'recovery_target_met':sum(bool(r.get('recovery_target_met')) for r in good),
            'median_recovery_s_observed_only':statistics.median(r['recovery_s'] for r in recovered) if recovered else None,
            'mean_repair_MiB':avg(r.get('traffic_bytes',{}).get('repair',0)/(1<<20) for r in good),
            'mean_one_copy_MiB_s':avg(r['one_copy_byte_seconds']/(1<<20) for r in good),
            'exposed_runs':sum(bool(r.get('post_fault_exposure')) for r in good),
            'overlap_runs':sum(bool(r.get('repair_foreground_overlap')) for r in good),
            'correct_committed_target':sum(bool(r.get('correctness',{}).get('second_fault_delivery',{}).get('hash_valid')) and bool(r.get('correctness',{}).get('second_fault_delivery',{}).get('committed')) for r in good)})
    write_csv(out/'aggregate.csv',table)
    contrasts=[]
    for experiment,load,comparisons in [('E1',l,[('OFF','N'),('S','OFF'),('S','N')]) for l in ('low','high')]+[('E3',l,[('L','S'),('L','F25'),('L','F50')]) for l in ('high','burst')]:
        for a,b in comparisons:
            bypol={pol:{r['seed']:r for r in rows if experiment in r.get('experiments',[]) and r['load']==load and r['policy']==pol and r['status']=='completed'} for pol in (a,b)}
            for metric in ('completion_rate','mean_post_fault_stall_s_all_sessions','one_copy_byte_seconds'):
                diffs=[bypol[a][s][metric]-bypol[b][s][metric] for s in sorted(bypol[a].keys()&bypol[b].keys()) if bypol[a][s].get(metric) is not None and bypol[b][s].get(metric) is not None]
                if diffs:contrasts.append({'experiment':experiment,'load':load,'contrast':a+' minus '+b,'metric':metric,'paired_n':len(diffs),'mean_difference':avg(diffs),'median_difference':statistics.median(diffs),'minimum_difference':min(diffs),'maximum_difference':max(diffs),'individual_differences':json.dumps(diffs),'uncertainty':'individual paired points and range; no population claim'})
    write_csv(out/'paired-contrasts.csv',contrasts)
    def finish(fig,name,title):
        fig.suptitle(title,fontweight='bold')
        if name.startswith('E5'):
            note='Exploratory • 3 trials per case • direct application probes • E5c latency includes deliberate quorum isolation; see E5-safety-table for restoration-relative timing.'
        elif name.startswith('E4'):
            note=f"Exploratory stress test • two paired contexts per cell, seeds {','.join(map(str,cfg['e4_seeds']))} • 3 Controllers / 3 Peers\nReal HTTP targeted delivery after two process stops; 50 s horizon; stopped disks retained; counts are not real-world loss probabilities."
        elif name=='L-budget-trace':
            note=f"Trace: {lr['run_id']} • {lr['completed']}/{lr['scheduled']} viewers completed; foreground idle here follows viewer failures.\nLines show recorded 250 ms source control updates, aligned within the stats interval; all classes still obey the common source upload cap."
        else:
            note=f"Compact exploratory study • real HTTP / simulated consumption • {cfg['segments']} × {cfg['duration_ms']/1000:g}s × {cfg['segment_bytes']/1024:g} KiB • {cfg['horizon_s']}s horizon\nCaps: {'/'.join(str(int(v/1024)) for v in cfg['rates'])} KiB/s • seeds {','.join(str(v) for v in cfg['seeds'])} • data {out.name} • peer binary {binary}.\nPoints = run means; small samples; verification every {cfg['verify_interval_s']}s; failures retained in tables."
        fig.text(.01,.01,note,fontsize=8,color='#555')
        fig.tight_layout(rect=(0,.18 if name=='E3-tradeoff' else .115,1,.93))
        fig.savefig(figdir/(name+'.svg'));fig.savefig(figdir/(name+'.png'));plt.close(fig)
    e1=[r for r in rows if 'E1' in r.get('experiments',[]) and r['status']=='completed']
    if e1:
        fig,axs=plt.subplots(1,2,figsize=(11,4.8))
        for j,load in enumerate(('low','high')):
            for i,pol in enumerate(('N','OFF','S')):
                rs=[r for r in e1 if r['load']==load and r['policy']==pol]
                if rs:
                    xs=[i+(k-(len(rs)-1)/2)*.08 for k in range(len(rs))]
                    axs[j].scatter(xs,[r['mean_post_fault_stall_s_all_sessions'] for r in rs],c=COLORS[pol],s=55)
                    axs[j].plot([i-.2,i+.2],[avg(r['mean_post_fault_stall_s_all_sessions'] for r in rs)]*2,c=COLORS[pol],lw=3)
                    axs[j].text(i,.98,f"{sum(r['completed'] for r in rs)}/{sum(r['scheduled'] for r in rs)} done",transform=axs[j].get_xaxis_transform(),ha='center',va='top',fontsize=9)
            axs[j].set(xticks=range(3),xticklabels=['N','OFF','S'],ylabel='Post-fault stall per scheduled viewer (s)',title=load+' steady demand');axs[j].grid(axis='y',alpha=.2)
        finish(fig,'E1-contention','E1 • Peer loss and repair: playback outcomes')
    e2=[r for r in rows if 'E2' in r.get('experiments',[]) and r['status']=='completed']
    if e2:
        fig,axs=plt.subplots(1,2,figsize=(11,4.8))
        for j,metric in enumerate(('repair','exposure')):
            for outage,color in [('brief','#0072b2'),('persistent','#d55e00')]:
                for grace in cfg['graces']:
                    rs=[r for r in e2 if r['outage']==outage and r['grace']==grace]
                    vals=[r['traffic_bytes']['repair']/(1<<20) if metric=='repair' else r['one_copy_byte_seconds']/(1<<20) for r in rs]
                    if vals:
                        axs[j].scatter([grace+(.1 if outage=='brief' else -.1)]*len(vals),vals,color=color,alpha=.7)
                        axs[j].plot(grace,avg(vals),marker='_',ms=18,color=color)
                axs[j].plot([],[],marker='o',ls='',label=outage,color=color)
            axs[j].set(xlabel='Repair grace (s)',ylabel='Repair payload (MiB)' if metric=='repair' else 'One-copy exposure (MiB·s)');axs[j].legend();axs[j].grid(alpha=.2)
        finish(fig,'E2-grace','E2 • Grace trades repair traffic against reduced redundancy')
    e3=[r for r in rows if 'E3' in r.get('experiments',[]) and r['status']=='completed']
    if e3:
        fig,axs=plt.subplots(1,2,figsize=(12,5.8))
        for ax,load in zip(axs,('high','burst')):
            for pol in ('S','F25','F50','L'):
                rs=[r for r in e3 if r['load']==load and r['policy']==pol]
                for r in rs:
                    ax.scatter(r['recovery_s'] if r['recovery_s'] is not None else cfg['horizon_s']-cfg['fault_s']+2,r['mean_post_fault_stall_s_all_sessions'],color=COLORS[pol],marker='o' if r['recovery_s'] is not None else 'x',s=55)
                ax.plot([],[],color=COLORS[pol],marker='o',ls='',label=pol)
            counts=[(sum(r['completed'] for r in e3 if r['load']==load and r['policy']==p),sum(r['scheduled'] for r in e3 if r['load']==load and r['policy']==p)) for p in ('S','F25','F50','L')]
            completion=(f'{counts[0][0]}/{counts[0][1]} complete per policy' if len(set(counts))==1 else 'completion shown in policy table')
            ax.axvline(cfg['recovery_target_s'],color='#888',ls='--',lw=1);ax.set(title=load+' demand • '+completion,xlabel='Verified restoration after fault (s)\n× = unobserved by horizon (right boundary)',ylabel='Post-fault stall per viewer (s)');ax.legend();ax.grid(alpha=.2)
        finish(fig,'E3-tradeoff','E3 • Playback–recovery compromise')
    e4=[r for r in rows if 'E4' in r.get('experiments',[]) and r['status']=='completed']
    if e4:
        fig,ax=plt.subplots(figsize=(10,4.8));pols=['S',cfg['fixed_e4_policy'],'L']
        for i,pol in enumerate(pols):
            vals=[]
            for delay in cfg['second_delays']:
                rs=[r for r in e4 if r['policy']==pol and r['delay']==delay]
                vals.append(avg(int(r['correctness']['second_fault_delivery']['hash_valid'] and r['correctness']['second_fault_delivery']['committed']) for r in rs))
            ax.plot(cfg['second_delays'],[float('nan') if v is None else v for v in vals],marker='o',label=pol,color=COLORS[pol])
        ax.set(xlabel='Second-holder failure delay (s after first)',ylabel='Hash-valid and committed target delivery fraction',ylim=(-.05,1.08));ax.legend();ax.grid(alpha=.2)
        finish(fig,'E4-second-failure','E4 • Conditional availability under a second holder failure')
    e5=read(out/'controller-runs.jsonl')
    if e5:
        write_csv(out/'controller-summary.csv',[{**{k:v for k,v in r.items() if not isinstance(v,(list,dict))},'checks':json.dumps(r.get('checks',{}))} for r in e5])
        fig,axs=plt.subplots(1,2,figsize=(11,4.8))
        cases=['E5a','E5b','E5c']
        for i,case in enumerate(cases):
            rs=[r for r in e5 if r['case']==case]
            vals=[r['application_recovery_s'] for r in rs if r.get('application_recovery_s') is not None]
            axs[0].scatter([i]*len(vals),vals,color='#0072b2',s=55)
            axs[1].bar(i,sum(r['status']=='passed' for r in rs),color='#009e73');axs[1].bar(i,sum(r['status']!='passed' for r in rs),bottom=sum(r['status']=='passed' for r in rs),color='#d55e00')
        axs[0].set(xticks=range(3),xticklabels=cases,ylabel='Fresh application metadata recovery (s)');axs[1].set(xticks=range(3),xticklabels=cases,ylabel='Runs (green pass / orange failed or missed)')
        finish(fig,'E5-controller','E5 • Application recovery and correctness outcomes')
    # One source budget trace, with control-update points deduplicated by instance.
    lr=next((r for r in rows if r['policy']=='L' and r['status']=='completed'),None)
    if lr:
        samples={}
        for r in read(out/lr['run_id']/'peer-traffic-and-policy.jsonl'):
            if not r.get('ok'):continue
            for h in r['body'].get('policy_history',[]):
                samples[(r['peer_id'],r['body']['instance_id'],h['sequence'])]=(r['observer_s']+(h['unix_ms']-r['body']['policy_history'][-1]['unix_ms'])/1000,h)
        fig,axs=plt.subplots(2,1,figsize=(10,6),sharex=True)
        for peer in ('peer-1','peer-2','peer-3'):
            vals=sorted(v for k,v in samples.items() if k[0]==peer)
            axs[0].step([v[0] for v in vals],[v[1]['repair_bytes_per_second']/1024 for v in vals],where='post',label=peer)
            axs[1].step([v[0] for v in vals],[v[1]['active_foreground'] for v in vals],where='post',label=peer)
        axs[0].set(ylabel='Repair cap (KiB/s)');axs[0].legend();axs[1].set(ylabel='Active foreground GETs',xlabel='Observer-aligned time (s; within stats polling interval)')
        finish(fig,'L-budget-trace','L • Observed repair budget and foreground activity')
    expected=len(json.loads((out/'matrix.json').read_text())) if (out/'matrix.json').exists() else 0
    lines=['# Compact repair study results','',f'Archived E1–E4 runs: **{len(rows)}/{expected}**. Controller runs: **{len(e5)}**.',
        '',f'Exploratory study: {len(cfg["seeds"])} paired workload seeds, 3 Controllers, 3 Peers, {cfg["horizon_s"]} s fixed horizon. Real HTTP transfer with headless simulated media consumption. No browser performance claim.',
        '', '## Slide-ready evidence', '', '| Experiment | Group | Runs | Complete viewers | Mean post-fault stall (s) | Recovery observed | Target met | Repair MiB |', '|---|---|---:|---:|---:|---:|---:|---:|']
    def fmt(v):return '—' if v is None else f'{v:.2f}'
    for r in table:
        lines.append(f'| {r["experiment"]} | {r["policy"]}, {r["load"]}, grace {r["grace_s"]}, {r["outage"]}, Δ2={r["second_delay_s"]} | {r["runs"]} | {r["sessions_completed"]}/{r["sessions_scheduled"]} | {fmt(r["mean_post_fault_stall_s_all"])} | {r["recovery_observed"]}/{r["runs"]} | {r["recovery_target_met"]}/{r["runs"]} | {fmt(r["mean_repair_MiB"])} |')
    lines+=['','## Interpretation boundaries','','Small samples characterize these workloads; individual paired differences and ranges are in `paired-contrasts.csv`. They do not establish population noninferiority or precise tail latency. Unrecovered runs remain censored; no zero or fabricated recovery time is substituted.',
        '', 'Copy exposure is sampled from hash-valid inventories intersected with quorum-backed registrations. Verification failure is unknown/unreachable, not proof of destroyed data. The first post-fault sample is interval censored at the verification cadence. Repair bytes are ResponseWriter payload, including retries; probe traffic has a separate class. Counter instances are tracked across restart.',
        '', 'See `summary.csv` for every run, `aggregate.csv` for slide tables, `controller-summary.csv` for controller checks, and `figures/` for SVG and PNG exports. Raw event journals, injections, manifests, task/copy observations and logs are under each run directory.',
        '', f'Time budget reduction: grace 1/3/8 s, brief outage {cfg["brief_s"]} s, first fault 5 s, second fault delays 4/14/30 s, recovery objective 40 s. L retains the original 250 ms / 2 s idle / 60 s aging rule; compact runs do not test the age branch statistically. Regression tests exercise that branch.',
        '', 'The final scientific verdict must consider completion, recovery censoring, traffic and exposure together; a completed-only stall statistic cannot establish improvement.']
    (out/'RESULTS.md').write_text('\n'.join(lines)+'\n')

if __name__=='__main__':main()
