// TODO(Haoyu): Keep local lifecycle and upload feedback tied to actual server responses.
import { useEffect, useState } from 'react';
import FaultLab from './FaultLab';
import { responseError, serverMessage } from '../api/messages';

interface Node { id: string; kind: string; port: number; pid: number; status: string; details?: { state?: string; objects?: number; stored_bytes?: number; uploaded_bytes?: number } }
interface Job { id: string; name: string; stage: string; segments: number; error?: string }
interface Status { nodes: Node[]; busy: boolean; job?: Job }
const stages: Record<string, string> = { uploaded: 'Upload complete', trimming: 'Trimming video', segmenting: 'Encoding and segmenting', manifest: 'Building the verification manifest', importing: 'Distributing segments and confirming replicas', ready: 'Ready to play', failed: 'Processing failed' };
async function api<T>(path: string, body?: object): Promise<T> {
  const res = await fetch(`/api/local/${path}`, body ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } : undefined);
  const data = await res.json(); if (!res.ok) throw responseError(data.error?.message ?? 'Request failed', res.status); return data;
}
const size = (n = 0) => `${(n / 1024 / 1024).toFixed(2)} MB`;
export default function LocalConsole({ onSelect, onReset }: { onSelect: (id: string, name: string) => void; onReset: () => void }) {
  const [status, setStatus] = useState<Status>({ nodes: [], busy: false });
  const [videos, setVideos] = useState<Job[]>([]);
  const [peers, setPeers] = useState(3); const [controllers, setControllers] = useState(1);
  const [pending, setPending] = useState(false); const [error, setError] = useState(''); const [connected, setConnected] = useState(false);
  const [file, setFile] = useState<File>(); const [start, setStart] = useState('0'); const [duration, setDuration] = useState('');
  const [progress, setProgress] = useState<number | null>(null); const [notice, setNotice] = useState('');
  async function refresh() {
    try { const [s, v] = await Promise.all([api<Status>('status'), api<Job[]>('videos')]); setStatus(s); setVideos(v); setConnected(true); }
    catch { setConnected(false); }
  }
  useEffect(() => { void refresh(); const timer = window.setInterval(() => void refresh(), 1500); return () => window.clearInterval(timer); }, []);
  const active = status.nodes.length > 0;
  const ready = connected && active && status.nodes.every(n => n.status === 'running') && status.nodes.some(n => n.kind === 'controller' && n.details?.state === 'Leader');
  async function cluster(action: 'start' | 'stop') {
    setPending(true); setError(''); setNotice('');
    try { await api(action, action === 'start' ? { peers, controllers } : {}); onReset(); setNotice(action === 'start' ? 'Cluster startup requested. Waiting for healthy nodes and leader election.' : 'Cluster stopped. Video data has been retained.'); await refresh(); }
    catch (err) { setError(String(err)); } finally { setPending(false); }
  }
  async function deleteVideo(video: Job) {
    if (!window.confirm(`Delete "${video.name}"? All registered replicas, local source files, and the library record will be permanently removed.`)) return;
    setPending(true); setError('');
    try { await api<{message:string}>(`videos/${encodeURIComponent(video.id)}/delete`, {}); onReset(); setNotice('Video deletion request processed. Pending work is shown in the library and maintenance tasks.'); await refresh(); }
    catch(err) { setError(String(err)); } finally { setPending(false); }
  }
  function upload() {
    if (!file) return;
    setPending(true); setError(''); setNotice(''); setProgress(0);
    const data = new FormData(); data.append('video', file); data.append('start', start); data.append('duration', duration);
    const xhr = new XMLHttpRequest(); xhr.open('POST', '/api/local/upload');
    xhr.upload.onprogress = event => { if (event.lengthComputable) setProgress(Math.round(event.loaded / event.total * 100)); };
    xhr.onerror = () => { setError('Upload connection failed. Check that the local console is running.'); setPending(false); setProgress(null); };
    xhr.onload = () => {
      try { const out = JSON.parse(xhr.responseText); if (xhr.status >= 400) throw responseError(out.error?.message ?? 'Upload failed', xhr.status); setNotice('Upload complete. Background processing progress is shown below.'); }
      catch (err) { setError(String(err)); } finally { setPending(false); setProgress(null); void refresh(); }
    };
    xhr.send(data);
  }
  return <>
    <nav className="workflow" aria-label="Workflow"><div className={ready ? 'complete' : 'current'}><b>1</b><span>Start cluster<small>Prepare storage nodes first</small></span></div><span className="flow-arrow">→</span><div className={ready && !videos.length ? 'current' : ''}><b>2</b><span>Upload and process<small>Optional trim → Segment → Distribute</small></span></div><span className="flow-arrow">→</span><div className={ready && videos.length ? 'current' : ''}><b>3</b><span>Choose a video<small>Play directly from the library</small></span></div></nav>
    <p className="workflow-hint">{!active ? 'Start the cluster first. You can upload and trim a video afterwards.' : !ready ? 'Waiting for healthy nodes and a controller leader. Check the fault lab if a node has exited.' : status.busy ? 'Video processing is in progress. Wait for import to finish.' : videos.length ? 'Cluster ready. Upload another video or play one from the library.' : 'Cluster ready. Choose a video and upload it. Keep the trim settings unchanged to use the full video.'}</p>
    <div className="console-grid">
      <section className="panel"><div className="section-title"><div><span className="step">01 / CLUSTER</span><h2>① Start cluster</h2></div><span className={`badge ${ready ? 'good' : ''}`}>{!connected ? 'Console disconnected' : ready ? 'Ready' : active ? 'Checking nodes' : 'Not started'}</span></div>
        <p className="muted">Peers store and serve video segments. Controllers track replica locations. Start the cluster before importing a video.</p>
        <div className="form-row"><label>Peers<select value={peers} disabled={active || pending} onChange={e => setPeers(+e.target.value)}>{[2,3,4,5,6,7,8].map(n => <option key={n}>{n}</option>)}</select></label>
          <label>Controller<select value={controllers} disabled={active || pending} onChange={e => setControllers(+e.target.value)}><option value={1}>1 · Local development</option><option value={3}>3 · Raft cluster</option></select></label></div>
        <div className="actions"><button disabled={!connected || active || pending} onClick={() => void cluster('start')}>Start cluster</button><button className="secondary" disabled={!active || pending || status.busy} onClick={() => void cluster('stop')}>Stop cluster</button></div>
        <p className="muted small">Each segment has two replicas, requiring at least 2 peers. Stop the cluster before changing its size. Each topology has separate data.</p>
      </section>
      <section className="panel"><div className="section-title"><div><span className="step">02 / INGEST</span><h2>② Upload and process</h2></div><span className="badge">Up to 512 MB</span></div>
        <p className="muted">Trimming is optional. Segmentation runs automatically before distributing the video to peers.</p>
        <label className="file-picker">{file ? file.name : 'Choose a video file'}<input type="file" aria-label="Choose video file" accept="video/*,.mkv,.mov,.mp4,.webm" disabled={!ready || pending || status.busy} onChange={e => setFile(e.target.files?.[0])}/></label>
        <details><summary>Optional trim · Full video by default</summary><div className="form-row"><label>Start time (seconds)<input type="number" min="0" max="86400" value={start} disabled={!ready || pending || status.busy} onChange={e => setStart(e.target.value)}/></label><label>Duration (seconds)<input type="number" min="0" max="86400" placeholder="Leave empty to keep the rest" value={duration} disabled={!ready || pending || status.busy} onChange={e => setDuration(e.target.value)}/></label></div><p className="field-help">For example, start at 10 seconds with a duration of 5 to keep seconds 10–15. Start at 0 and leave duration empty to keep the full video.</p></details>
        <button disabled={!ready || !file || pending || status.busy} onClick={upload}>{progress !== null ? `Uploading ${progress}%` : status.busy ? 'Processing…' : 'Upload and process'}</button>
        {progress !== null && <progress value={progress} max={100} aria-label="Upload progress"/>}
        {!ready && <p className="muted small">Start the cluster in step 1 to enable uploads.</p>}
        {status.job && <div className={`job ${status.job.stage === 'failed' ? 'failed' : ''}`} role="status"><strong>{status.job.stage === 'ready' && !active ? 'Previously imported. Start the cluster to play.' : stages[status.job.stage] ?? status.job.stage}</strong><span>{status.job.name}{status.job.segments > 0 ? ` · ${status.job.segments} segments` : ''}</span>{status.job.error && <details><summary>Original server details</summary><pre>{status.job.error}</pre></details>}</div>}
      </section>
    </div>
    {!connected && <p className="notice">The local console is disconnected. In the project directory, run <code>make console</code> and open http://127.0.0.1:8090.</p>}
    {notice && <p className="notice" role="status">{notice}</p>}{error && <p className="notice failed" role="alert">{error}</p>}
    {active && <section aria-labelledby="peer-segments-title">
      <div className="section-title"><h2 id="peer-segments-title">Segments by peer</h2><span className="muted small">Updated every 1.5 seconds</span></div>
      <p className="muted small">Registered segments across all videos. Each peer replica is counted separately. Offline or unreadable nodes show a dash.</p>
      <div className="peer-segment-grid">{status.nodes.filter(n => n.kind === 'peer').map(n => {
        // An unavailable sample is unknown, not an empty Peer. Keep real zeroes visible.
        const available = connected && n.status === 'running' && n.details?.objects !== undefined;
        return <article className="peer-segment-card" key={n.id} aria-label={`${n.id} segment statistics`}>
          <div className="section-title"><strong>{n.id}</strong><span className={`badge ${available ? 'good' : ''}`}>{available ? 'Online' : n.status === 'exited' ? 'Exited' : 'Unavailable'}</span></div>
          <p className="peer-segment-count">{available ? n.details!.objects!.toLocaleString() : '—'} <span>segments</span></p>
          <p className="muted small">Stored {available && n.details?.stored_bytes !== undefined ? size(n.details.stored_bytes) : '—'}</p>
        </article>;
      })}</div>
    </section>}
    {active && <details className="node-details"><summary>Node status, ports, and traffic</summary><section><div className="section-title"><h2>Nodes and traffic</h2><span className="muted small">Polled every 1.5 seconds · Sent bytes include all segment requests in this process</span></div><div className="table-scroll"><table><thead><tr><th>Node</th><th>Status</th><th>Local port</th><th>Segments</th><th>Stored</th><th>Bytes sent</th></tr></thead><tbody>{status.nodes.map(n => <tr key={n.id}><td>{n.id}</td><td><span className={`badge ${n.status === 'running' ? 'good' : ''}`}>{n.status === 'running' ? n.details?.state ?? 'Running' : n.status === 'exited' ? 'Exited' : 'Not ready'}</span></td><td><code>{n.port}</code></td><td>{n.details?.objects ?? '—'}</td><td>{n.kind === 'peer' ? size(n.details?.stored_bytes) : '—'}</td><td>{n.kind === 'peer' ? size(n.details?.uploaded_bytes) : '—'}</td></tr>)}</tbody></table></div></section></details>}
    <section><div className="section-title"><div><span className="step">03 / LIBRARY</span><h2>③ Choose a video</h2></div><span className="badge">{videos.length} imported</span></div>{videos.length ? <div className="library">{videos.map(v => <div className="video-item" key={v.id}><div><strong>{v.name}</strong><p className="muted small">{v.stage === 'deleting' ? `Deleting: ${v.error ? serverMessage(v.error) : 'Waiting for node acknowledgements'}` : `${v.segments} segments · Two replicas confirmed`}</p><details><summary>Show video ID</summary><code>{v.id}</code></details></div><button className="secondary" disabled={!active || !connected || v.stage==='deleting'} onClick={() => onSelect(v.id, v.name)}>Play</button><button className="danger" disabled={!active || !connected || pending || status.busy || v.stage==='deleting'} onClick={()=>void deleteVideo(v)}>Delete video</button></div>)}</div> : <p className="muted">Imported videos appear here. Select Play to start without entering an ID.</p>}</section>
    <FaultLab nodes={status.nodes} busy={status.busy} disabled={pending} connected={connected} onPending={setPending} onRefresh={refresh}/>
  </>;
}
