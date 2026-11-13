// TODO(Yuquan): Add keeper preferences when placement policies expose stable rankings.
import { useState } from 'react';
interface Task { task_id: string; segment_id: string; size_bytes: number; target: { peer_id: string }; keep: { peer_id: string }[] }
export default function ReplicaCleanup({ video }: { video: string }) {
 const [tasks,setTasks]=useState<Task[]>([]);
 const [busy,setBusy]=useState(false);
 const [message,setMessage]=useState('');
 async function run(execute: boolean) {
  setBusy(true); setMessage('');
  try {
   const response=await fetch(`/api/local/videos/${encodeURIComponent(video)}/cleanup`, execute ? {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({tasks})} : undefined);
   const body=await response.json(); if(!response.ok) throw new Error(body.error?.message ?? body.error ?? `HTTP ${response.status}`);
   if(execute) { setTasks([]); setMessage('清理任务已提交。在故障测试台查看任务状态；离线节点恢复后会继续清理。'); }
   else { setTasks(body.tasks); setMessage(body.tasks.length ? '请核对下面的保留与删除计划，再执行清理。' : '没有可安全清理的副本，或已有任务正在进行。'); }
  } catch(error) { setTasks([]); setMessage(String(error)); } finally { setBusy(false); }
 }
 return <section><details><summary>清理当前视频的冗余副本</summary>
 <p className="muted">优先清理预计存储负载较高节点的冗余副本，尽量保持分布均衡。每个分片保留两份已验证副本，每次最多清理一份多余副本。共享文件可能不会立即释放磁盘空间。</p>
 <button disabled={busy} onClick={()=>void run(false)}>预览清理计划</button>
 {tasks.length>0 && <><p>待清理 {tasks.length} 份副本，逻辑大小 {(tasks.reduce((n,t)=>n+t.size_bytes,0)/1048576).toFixed(2)} MiB。</p>
 <div style={{maxHeight:280,overflow:'auto'}}><table><thead><tr><th>分片</th><th>保留</th><th>删除位置</th></tr></thead><tbody>{tasks.map(t=><tr key={t.task_id}><td>{t.segment_id}</td><td>{t.keep.map(p=>p.peer_id).join('、')}</td><td>{t.target.peer_id}</td></tr>)}</tbody></table></div>
 <button className="danger" disabled={busy} onClick={()=>void run(true)}>执行此清理计划</button></>}
 <p role="status">{message}</p></details></section>;
}
