// TODO(Yuquan): Add replayable network-fault traces after process-fault workflows are validated.
import { useEffect, useRef, useState } from 'react';
import EventLog from './EventLog';
import { faultEntry } from '../logging/events';
import type { LogEntry } from '../logging/events';
import { responseError, serverMessage } from '../api/messages';

interface NodeInfo { id: string; kind: string; status: string; port: number; details?: { state?: string } }
interface Task { task_id: string; segment_id: string; state: string; source: { peer_id: string }; target: { peer_id: string } }
interface FaultEvent { sequence: number; node_id: string; action: string; at: string }
interface Maintenance { available: boolean; leader_id?: string; error?: string; tasks: Record<string, Task> | null; events: FaultEvent[] }
interface Props { nodes: NodeInfo[]; busy: boolean; disabled: boolean; connected: boolean; onPending: (value: boolean) => void; onRefresh: () => Promise<void> }
const actions: Record<string, string> = { stop: 'Stopped', start: 'Recovered', restart: 'Restarted' };
const taskStates: Record<string, string> = { pending: 'Pending', completed: 'Completed', failed: 'Failed; retryable' };

export default function FaultLab({ nodes, busy, disabled, connected, onPending, onRefresh }: Props) {
  const [data, setData] = useState<Maintenance>({ available: false, tasks: {}, events: [] });
  const [message, setMessage] = useState(''); const [error, setError] = useState('');
  const [working, setWorking] = useState(''); const [observedAt, setObservedAt] = useState('');
  const [observations, setObservations] = useState<LogEntry[]>([]);
  const lastAvailable = useRef<boolean | undefined>(undefined); const sequence = useRef(0);
  const alive = useRef(true); const fetching = useRef(false); const lock = useRef(false);
  async function refresh() {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const response = await fetch('/api/local/maintenance', { signal: AbortSignal.timeout(5000) });
      const body = await response.json(); if (!response.ok) throw new Error(body.error?.message ?? 'Unable to read status');
      if (alive.current) {
        setData({ ...body, error: body.error ? serverMessage(body.error) : '' }); setObservedAt(new Date().toLocaleTimeString('en-US'));
        if (lastAvailable.current !== body.available) {
          const entry: LogEntry = { id: `monitor-${++sequence.current}`, time: new Date().toISOString(), level: body.available ? 'success' : 'warning', category: 'Monitor', node: body.leader_id,
            message: body.available ? 'Metadata is readable from the leader.' : 'Metadata is unavailable. Node recovery controls remain available.', details: { available: body.available, error: body.error } };
          setObservations(previous => [...previous, entry].slice(-100)); lastAvailable.current = body.available;
        }
      }
    } catch { if (alive.current) setData(previous => ({ ...previous, available: false, error: 'Cannot reach the fault management service. Check the console connection.' })); }
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
    lock.current = true; onPending(true); setWorking(targets.map(node => node.id).join(', ')); setError(''); setMessage('');
    const done: string[] = [];
    try {
      for (const node of targets) {
        const response = await fetch(`/api/local/nodes/${encodeURIComponent(node.id)}/${action}`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(leaderOnly ? { expected_role: 'Leader' } : {}), signal: AbortSignal.timeout(10000),
        });
        const body = await response.json(); if (!response.ok) throw responseError(body.error?.message ?? 'Operation failed', response.status);
        done.push(node.id);
      }
      if (alive.current) setMessage(`${done.join(', ')}: ${actions[action]}. ${action === 'stop' ? 'Observe download sources, elections, and maintenance tasks.' : 'Wait for health checks and leader election.'} Video data is retained.`);
    } catch (reason) {
      if (alive.current) {
        const message = `${done.length ? `Already updated ${done.join(', ')}; ` : ''}${reason instanceof Error ? reason.message : String(reason)}`;
        setError(message); setObservations(previous => [...previous, { id: `monitor-${++sequence.current}`, time: new Date().toISOString(), level: 'error', category: 'Fault control', message, details: { original_error: reason instanceof Error ? reason.cause ?? reason.message : reason } } as LogEntry].slice(-100));
      }
    } finally {
      lock.current = false; onPending(false); if (alive.current) { setWorking(''); await onRefresh(); await refresh(); }
    }
  }
  function download() {
    const url = URL.createObjectURL(new Blob([JSON.stringify({ exported_at: new Date().toISOString(), nodes, maintenance: data }, null, 2)], { type: 'application/json' }));
    const link = document.createElement('a'); link.href = url; link.download = `fault-test-${Date.now()}.json`; link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return <section id="fault-lab" className="fault-lab" aria-label="Fault lab">
    <div className="section-title"><div><span className="step">FAULT LAB</span><h2>Fault lab</h2></div><span className={`badge ${data.available ? 'good' : ''}`}>{data.available ? 'Metadata available' : 'Metadata unavailable'}</span></div>
    <p className="muted">Stop nodes during playback to observe failover, elections, and replica repair. Controls affect only this console’s processes and retain video data.</p>
    <div className="fault-stats">
      <div><span>Running peers</span><strong>{peers.filter(node => node.status === 'running').length} / {peers.length}</strong></div>
      <div><span>Running controllers</span><strong>{controllers.filter(node => node.status === 'running').length} / {controllers.length}</strong></div>
      <div><span>Current leader</span><strong>{data.available ? data.leader_id : 'No readable leader'}</strong></div>
      <div><span>Controller quorum</span><strong>{controllers.length ? `${quorum} nodes` : 'Not started'}</strong></div>
    </div>
    <div className="actions fault-actions">
      <button className="danger" disabled={blocked || !leader} onClick={() => leader && void mutate([leader], 'stop', true)}>Stop current leader</button>
      <button className="secondary" disabled={blocked || !stopped.length} onClick={() => void mutate(stopped, 'start')}>Recover exited nodes</button>
      <button className="secondary" disabled={!connected} onClick={() => { void onRefresh(); void refresh(); }}>Refresh status</button>
      <button className="secondary" disabled={!nodes.length && !data.events.length} onClick={download}>Export test snapshot</button>
    </div>
    {controllers.length === 1 && <p className="fault-warning">This cluster has one controller. Stopping it makes metadata unavailable until it recovers. Use a 3-controller cluster to test leader election.</p>}
    {controllers.length > 1 && controllers.filter(node => node.status !== 'exited').length < quorum && <p className="fault-warning">Fewer controllers are running than required for quorum. Recover nodes before committing replicas or maintenance results.</p>}
    {busy && <p className="notice">Fault controls are disabled while a video is processing. Test during playback after import finishes.</p>}
    {!nodes.length && <p className="notice">Start the cluster above to enable node controls.</p>}
    {working && <p role="status" className="notice">Updating {working}…</p>}
    {message && <p role="status" className="notice">{message}</p>}{error && <p role="alert" className="notice failed">{error}</p>}
    <div className="table-scroll"><table><caption className="sr-only">Node fault controls</caption><thead><tr><th>Node</th><th>Role / status</th><th>Port</th><th>Actions</th></tr></thead><tbody>
      {nodes.map(node => <tr key={node.id}><td><strong>{node.id}</strong><small className="node-kind">{node.kind === 'peer' ? 'Segment storage' : node.kind === 'controller' ? 'Metadata and consensus' : 'Playback API'}</small></td>
        <td><span className={`badge ${node.status === 'running' ? 'good' : ''}`}>{node.status === 'exited' ? 'Exited' : node.status === 'running' ? node.details?.state ?? 'Running' : 'Checking'}</span></td><td>{node.port}</td>
        <td><div className="node-actions"><button className="danger" aria-label={`Stop ${node.id}`} disabled={blocked || node.status === 'exited'} onClick={() => void mutate([node], 'stop')}>Stop</button>
          <button className="secondary" aria-label={`Recover ${node.id}`} disabled={blocked || node.status !== 'exited'} onClick={() => void mutate([node], 'start')}>Recover</button>
          <button className="secondary" aria-label={`Restart ${node.id}`} disabled={blocked || node.status === 'exited'} onClick={() => void mutate([node], 'restart')}>Restart</button></div></td></tr>)}
    </tbody></table></div>
    <details className="fault-guide"><summary>Test guide and expected results</summary><ol>
      <li>Play a longer video, then stop a serving peer. Check that later segments use another peer; buffered segments are unaffected.</li>
      <li>Wait for repair: the default grace period is 6 seconds, followed by detection and copying. A completed task confirms the new replica; uninterrupted playback alone does not.</li>
      <li>With 3 controllers, stop the leader and observe election and metadata recovery. Stop a majority to test loss of quorum.</li>
      <li>Stop the playback API to observe request failures. Recover it and restart playback. Export a snapshot after restoring all nodes.</li>
    </ol></details>
    <div className="section-title"><h3>Replica maintenance</h3><span className="muted small">Polled every 2 seconds · {observedAt ? `Updated ${observedAt}` : 'Waiting for status'}</span></div>
    {!data.available ? <p className="muted" role="status">{data.error || 'No readable leader. Task status is currently unavailable.'} Node recovery controls remain available.</p> : !tasks.length ? <p className="muted">No maintenance tasks. Repair starts after the grace period when a healthy source and available target exist.</p> : <>
      <p className="muted small">Pending {tasks.filter(task => task.state === 'pending').length} · Completed {tasks.filter(task => task.state === 'completed').length} · Failed {tasks.filter(task => task.state === 'failed').length}; Showing up to 12 tasks.</p>
      <div className="table-scroll"><table><thead><tr><th>Segment</th><th>Action</th><th>Task status</th></tr></thead><tbody>{tasks.slice(-12).reverse().map(task => <tr key={task.task_id}><td className="segment-cell">{task.segment_id}</td><td>{task.task_id.startsWith('delete-') ? `Delete replica on ${task.target.peer_id}` : task.task_id.startsWith('cleanup-') ? `Clean up ${task.target.peer_id}` : `${task.source.peer_id} → ${task.target.peer_id}`}</td><td><span className={`badge ${task.state === 'completed' ? 'good' : ''}`}>{taskStates[task.state] ?? task.state}</span></td></tr>)}</tbody></table></div>
    </>}
    <EventLog entries={[...data.events.map(faultEntry), ...observations]} title="Cluster activity" fileName="cluster-events"
      description="Server-confirmed fault actions and browser-observed metadata changes. Pause freezes this view, not the cluster. Console history resets when the manager restarts."/>
  </section>;
}
