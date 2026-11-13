// TODO(Yuquan): Add keeper preferences when placement policies expose stable rankings.
import { useState } from 'react';
import { responseError } from '../api/messages';
interface Task { task_id: string; segment_id: string; size_bytes: number; target: { peer_id: string }; keep: { peer_id: string }[] }
export default function ReplicaCleanup({ video }: { video: string }) {
 const [tasks,setTasks]=useState<Task[]>([]);
 const [busy,setBusy]=useState(false);
 const [message,setMessage]=useState('');
 async function run(execute: boolean) {
  setBusy(true); setMessage('');
  try {
   const response=await fetch(`/api/local/videos/${encodeURIComponent(video)}/cleanup`, execute ? {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({tasks})} : undefined);
   const body=await response.json(); if(!response.ok) throw responseError(body.error?.message ?? body.error ?? `HTTP ${response.status}`,response.status);
   if(execute) { setTasks([]); setMessage('Cleanup submitted. Follow task status in the fault lab. Offline targets are retried after recovery.'); }
   else { setTasks(body.tasks); setMessage(body.tasks.length ? 'Review which replicas to keep and remove before applying the plan.' : 'No replicas can be safely removed, or a maintenance task is already running.'); }
  } catch(error) { setTasks([]); setMessage(String(error)); } finally { setBusy(false); }
 }
 return <section><details><summary>Clean up redundant replicas</summary>
 <p className="muted">Prefer removing replicas from more heavily loaded peers. Keep two verified replicas per segment and remove at most one extra copy per plan. Shared blobs may not immediately free disk space.</p>
 <button disabled={busy} onClick={()=>void run(false)}>Preview cleanup</button>
 {tasks.length>0 && <><p>{tasks.length} replicas selected; logical size {(tasks.reduce((n,t)=>n+t.size_bytes,0)/1048576).toFixed(2)} MiB.</p>
 <div style={{maxHeight:280,overflow:'auto'}}><table><thead><tr><th>Segment</th><th>Keep</th><th>Remove from</th></tr></thead><tbody>{tasks.map(t=><tr key={t.task_id}><td>{t.segment_id}</td><td>{t.keep.map(p=>p.peer_id).join(', ')}</td><td>{t.target.peer_id}</td></tr>)}</tbody></table></div>
 <button className="danger" disabled={busy} onClick={()=>void run(true)}>Apply cleanup plan</button></>}
 <p role="status">{message}</p></details></section>;
}
