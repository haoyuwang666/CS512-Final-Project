// TODO(Yuquan): Add replayable network-fault traces after process-fault workflows are validated.
import { useEffect, useRef, useState } from 'react';

interface NodeInfo { id: string; kind: string; status: string; port: number; details?: { state?: string } }
interface Task { task_id: string; segment_id: string; state: string; source: { peer_id: string }; target: { peer_id: string } }
interface FaultEvent { sequence: number; node_id: string; action: string; at: string }
interface Maintenance { available: boolean; leader_id?: string; error?: string; tasks: Record<string, Task> | null; events: FaultEvent[] }
interface Props { nodes: NodeInfo[]; busy: boolean; disabled: boolean; connected: boolean; onPending: (value: boolean) => void; onRefresh: () => Promise<void> }
const actions: Record<string, string> = { stop: '强制挂掉', start: '恢复', restart: '重启' };
const taskStates: Record<string, string> = { pending: '等待复制', completed: '已完成', failed: '失败，可重试' };

export default function FaultLab({ nodes, busy, disabled, connected, onPending, onRefresh }: Props) {
  const [data, setData] = useState<Maintenance>({ available: false, tasks: {}, events: [] });
  const [message, setMessage] = useState(''); const [error, setError] = useState('');
  const [working, setWorking] = useState(''); const [observedAt, setObservedAt] = useState('');
  const alive = useRef(true); const fetching = useRef(false); const lock = useRef(false);
  async function refresh() {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const response = await fetch('/api/local/maintenance', { signal: AbortSignal.timeout(5000) });
      const body = await response.json(); if (!response.ok) throw new Error(body.error?.message ?? '状态读取失败');
      if (alive.current) { setData(body); setObservedAt(new Date().toLocaleTimeString()); }
    } catch { if (alive.current) setData(previous => ({ ...previous, available: false, error: '无法连接故障管理服务，请检查控制台连接。' })); }
    finally { fetching.current = false; }
  }
  useEffect(() => {
    alive.current = true; void refresh(); const timer = window.setInterval(() => void refresh(), 2000);
    return () => { alive.current = false; window.clearInterval(timer); };
  }, []);
  const peers = nodes.filter(node => node.kind === 'peer');
  const controllers = nodes.filter(node => node.kind === 'controller');
  const quorum = Math.floor(controllers.length / 2) + 1;
  const leader = controllers.find(node => node.status === 'running' && node.details?.state === 'Leader');
  const stopped = nodes.filter(node => node.status === 'exited');
  const blocked = disabled || busy || !connected || !!working;
  const tasks = Object.values(data.tasks ?? {});
  async function mutate(targets: NodeInfo[], action: 'stop' | 'start' | 'restart', leaderOnly = false) {
    if (lock.current || blocked) return;
    lock.current = true; onPending(true); setWorking(targets.map(node => node.id).join('、')); setError(''); setMessage('');
    const done: string[] = [];
    try {
      for (const node of targets) {
        const response = await fetch(`/api/local/nodes/${encodeURIComponent(node.id)}/${action}`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(leaderOnly ? { expected_role: 'Leader' } : {}), signal: AbortSignal.timeout(10000),
        });
        const body = await response.json(); if (!response.ok) throw new Error(body.error?.message ?? '操作失败');
        done.push(node.id);
      }
      if (alive.current) setMessage(`${done.join('、')}：已${actions[action]}。${action === 'stop' ? '观察后续下载来源、选主和修复任务。' : '等待健康检查和选主恢复。'}视频数据保留。`);
    } catch (reason) {
      if (alive.current) setError(`${done.length ? `已操作 ${done.join('、')}；` : ''}${reason instanceof Error ? reason.message : String(reason)}`);
    } finally {
      lock.current = false; onPending(false); if (alive.current) { setWorking(''); await onRefresh(); await refresh(); }
    }
  }
  function download() {
    const url = URL.createObjectURL(new Blob([JSON.stringify({ exported_at: new Date().toISOString(), nodes, maintenance: data }, null, 2)], { type: 'application/json' }));
    const link = document.createElement('a'); link.href = url; link.download = `fault-test-${Date.now()}.json`; link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return <section id="fault-lab" className="fault-lab" aria-label="故障测试台">
    <div className="section-title"><div><span className="step">FAULT LAB</span><h2>故障测试台</h2></div><span className={`badge ${data.available ? 'good' : ''}`}>{data.available ? '元数据可读' : '元数据不可读'}</span></div>
    <p className="muted">播放视频时，直接挂掉节点并观察切源、选主与副本修复。仅操作本控制台启动的进程，停止和重启均保留视频数据。</p>
    <div className="fault-stats">
      <div><span>运行中的 Peer</span><strong>{peers.filter(node => node.status === 'running').length} / {peers.length}</strong></div>
      <div><span>运行中的 Controller</span><strong>{controllers.filter(node => node.status === 'running').length} / {controllers.length}</strong></div>
      <div><span>当前 Leader</span><strong>{data.available ? data.leader_id : '暂无可读 Leader'}</strong></div>
      <div><span>Controller 多数派门槛</span><strong>{controllers.length ? `${quorum} 个节点` : '未启动'}</strong></div>
    </div>
    <div className="actions fault-actions">
      <button className="danger" disabled={blocked || !leader} onClick={() => leader && void mutate([leader], 'stop', true)}>挂掉当前 Leader</button>
      <button className="secondary" disabled={blocked || !stopped.length} onClick={() => void mutate(stopped, 'start')}>恢复所有已退出节点</button>
      <button className="secondary" disabled={!connected} onClick={() => { void onRefresh(); void refresh(); }}>刷新状态</button>
      <button className="secondary" disabled={!nodes.length && !data.events.length} onClick={download}>导出测试记录</button>
    </div>
    {controllers.length === 1 && <p className="fault-warning">当前是单 Controller：挂掉它后不会自动选出新 Leader，恢复它后才能重新查询元数据。测试选主请先停止集群，再选择 3 个 Controller 启动。</p>}
    {controllers.length > 1 && controllers.filter(node => node.status !== 'exited').length < quorum && <p className="fault-warning">运行中的 Controller 已少于多数派门槛。请先恢复节点；没有多数派时不能提交新的副本或修复结果。</p>}
    {busy && <p className="notice">视频正在处理，故障操作暂时禁用；导入完成后可以在播放期间测试。</p>}
    {!nodes.length && <p className="notice">请先在上方启动集群，节点操作会显示在这里。</p>}
    {working && <p role="status" className="notice">正在操作 {working}…</p>}
    {message && <p role="status" className="notice">{message}</p>}{error && <p role="alert" className="notice failed">{error}</p>}
    <div className="table-scroll"><table><caption className="sr-only">节点故障操作</caption><thead><tr><th>节点</th><th>角色 / 状态</th><th>端口</th><th>故障操作</th></tr></thead><tbody>
      {nodes.map(node => <tr key={node.id}><td><strong>{node.id}</strong><small className="node-kind">{node.kind === 'peer' ? '分片存储' : node.kind === 'controller' ? '元数据与共识' : '播放 API'}</small></td>
        <td><span className={`badge ${node.status === 'running' ? 'good' : ''}`}>{node.status === 'exited' ? '已退出' : node.status === 'running' ? node.details?.state ?? '运行中' : '检查中'}</span></td><td>{node.port}</td>
        <td><div className="node-actions"><button className="danger" aria-label={`挂掉 ${node.id}`} disabled={blocked || node.status === 'exited'} onClick={() => void mutate([node], 'stop')}>挂掉</button>
          <button className="secondary" aria-label={`恢复 ${node.id}`} disabled={blocked || node.status !== 'exited'} onClick={() => void mutate([node], 'start')}>恢复</button>
          <button className="secondary" aria-label={`重启 ${node.id}`} disabled={blocked || node.status === 'exited'} onClick={() => void mutate([node], 'restart')}>重启</button></div></td></tr>)}
    </tbody></table></div>
    <details className="fault-guide"><summary>推荐测试步骤与判定方式</summary><ol>
      <li>先播放较长的视频，再挂掉一个正在供片的 Peer。确认新分片改由其他 Peer 下载；已缓冲片段不受影响。</li>
      <li>等待副本修复：默认宽限期 6 秒，之后还需要检测和复制时间。任务“已完成”表示新副本已确认；播放继续本身不能证明修复成功。</li>
      <li>使用 3 个 Controller 时挂掉 Leader，观察其他节点选主和元数据恢复；停止多数派时可验证元数据不可用。</li>
      <li>挂掉播放 API 可观察请求失败，再恢复并重新开始播放。恢复所有节点后，可导出本次测试记录。</li>
    </ol></details>
    <div className="section-title"><h3>自动修复任务</h3><span className="muted small">每 2 秒读取 · {observedAt ? `更新于 ${observedAt}` : '等待读取'}</span></div>
    {!data.available ? <p className="muted" role="status">{data.error || '暂无可读 Leader，任务状态暂不可用。'}管理操作仍可使用。</p> : !tasks.length ? <p className="muted">暂无修复任务。欠副本经过宽限期、且有健康来源和空闲目标后才会创建任务。</p> : <>
      <p className="muted small">等待 {tasks.filter(task => task.state === 'pending').length} · 完成 {tasks.filter(task => task.state === 'completed').length} · 失败 {tasks.filter(task => task.state === 'failed').length}；显示最多 12 项。</p>
      <div className="table-scroll"><table><thead><tr><th>分片</th><th>复制路径</th><th>任务状态</th></tr></thead><tbody>{tasks.slice(-12).reverse().map(task => <tr key={task.task_id}><td className="segment-cell">{task.segment_id}</td><td>{task.source.peer_id} → {task.target.peer_id}</td><td><span className={`badge ${task.state === 'completed' ? 'good' : ''}`}>{taskStates[task.state] ?? task.state}</span></td></tr>)}</tbody></table></div>
    </>}
    <details><summary>本次控制台的操作记录（最多 100 条）</summary>{data.events.length ? <ol className="fault-log">{data.events.slice().reverse().map(event => <li key={event.sequence}><time>{new Date(event.at).toLocaleTimeString()}</time> {event.node_id} · {actions[event.action] ?? event.action}</li>)}</ol> : <p className="muted small">尚无故障操作。记录在控制台重启后清空，重要测试请导出。</p>}</details>
  </section>;
}
