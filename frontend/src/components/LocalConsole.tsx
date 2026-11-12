// TODO(Haoyu): Keep local lifecycle and upload feedback tied to actual server responses.
import { useEffect, useState } from 'react';

interface Node { id: string; kind: string; port: number; pid: number; status: string; details?: { state?: string; objects?: number; stored_bytes?: number; uploaded_bytes?: number } }
interface Job { id: string; name: string; stage: string; segments: number; error?: string }
interface Status { nodes: Node[]; busy: boolean; job?: Job }
const stages: Record<string, string> = { uploaded: '上传完成', trimming: '正在裁剪', segmenting: '正在编码并切片', manifest: '切片成功，正在生成校验清单', importing: '正在分发至 Peer 并确认双副本', ready: '导入成功，可以播放', failed: '处理失败' };
async function api<T>(path: string, body?: object): Promise<T> {
  const res = await fetch(`/api/local/${path}`, body ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } : undefined);
  const data = await res.json(); if (!res.ok) throw new Error(data.error?.message ?? '请求失败'); return data;
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
    try { await api(action, action === 'start' ? { peers, controllers } : {}); onReset(); setNotice(action === 'start' ? '已提交启动请求，等待节点健康检查与选主。' : '集群已停止，视频数据已保留。'); await refresh(); }
    catch (err) { setError(String(err)); } finally { setPending(false); }
  }
  function upload() {
    if (!file) return;
    setPending(true); setError(''); setNotice(''); setProgress(0);
    const data = new FormData(); data.append('video', file); data.append('start', start); data.append('duration', duration);
    const xhr = new XMLHttpRequest(); xhr.open('POST', '/api/local/upload');
    xhr.upload.onprogress = event => { if (event.lengthComputable) setProgress(Math.round(event.loaded / event.total * 100)); };
    xhr.onerror = () => { setError('上传连接失败，请检查本地控制台是否运行。'); setPending(false); setProgress(null); };
    xhr.onload = () => {
      try { const out = JSON.parse(xhr.responseText); if (xhr.status >= 400) throw new Error(out.error?.message ?? '上传失败'); setNotice('文件已上传，正在后台处理；下方会显示实际进度。'); }
      catch (err) { setError(String(err)); } finally { setPending(false); setProgress(null); void refresh(); }
    };
    xhr.send(data);
  }
  return <>
    <nav className="workflow" aria-label="操作步骤"><div className={ready ? 'complete' : 'current'}><b>1</b><span>启动集群<small>先准备存放分片的节点</small></span></div><span className="flow-arrow">→</span><div className={ready && !videos.length ? 'current' : ''}><b>2</b><span>上传并自动处理<small>可选裁剪 → 自动切片 → 分发</small></span></div><span className="flow-arrow">→</span><div className={ready && videos.length ? 'current' : ''}><b>3</b><span>选择视频播放<small>从视频库一键开始</small></span></div></nav>
    <p className="workflow-hint">{!active ? '下一步：点击“启动集群”。无需先上传或裁剪视频。' : !ready ? '等待节点就绪和 Controller 选主；若节点退出，请检查状态。' : status.busy ? '视频正在自动处理，请等待导入完成。' : videos.length ? '集群已就绪。可以继续上传，或在视频库直接播放已有视频。' : '集群已就绪。现在选择视频并上传；不需要裁剪时保持默认即可。'}</p>
    <div className="console-grid">
      <section className="panel"><div className="section-title"><div><span className="step">01 / CLUSTER</span><h2>① 启动集群</h2></div><span className={`badge ${ready ? 'good' : ''}`}>{!connected ? '未连接管理服务' : ready ? '就绪' : active ? '检查节点中' : '未启动'}</span></div>
        <p className="muted">Peer 负责存放和传输视频分片；Controller 负责记录分片位置。先启动它们，视频才能分发。</p>
        <div className="form-row"><label>Peer 数量<select value={peers} disabled={active || pending} onChange={e => setPeers(+e.target.value)}>{[2,3,4,5,6,7,8].map(n => <option key={n}>{n}</option>)}</select></label>
          <label>Controller<select value={controllers} disabled={active || pending} onChange={e => setControllers(+e.target.value)}><option value={1}>1 · 本机联调</option><option value={3}>3 · Raft 集群</option></select></label></div>
        <div className="actions"><button disabled={!connected || active || pending} onClick={() => void cluster('start')}>启动集群</button><button className="secondary" disabled={!active || pending || status.busy} onClick={() => void cluster('stop')}>停止集群</button></div>
        <p className="muted small">每片保存两份副本，至少需要 2 个 Peer。更改数量前需停止集群；不同规模的数据分别保存。</p>
      </section>
      <section className="panel"><div className="section-title"><div><span className="step">02 / INGEST</span><h2>② 上传与处理</h2></div><span className="badge">最大 512 MB</span></div>
        <p className="muted">裁剪是可选项，用于截取一段视频。切片是自动步骤，用于把视频分发到不同 Peer。</p>
        <label className="file-picker">{file ? file.name : '选择视频文件'}<input type="file" accept="video/*,.mkv,.mov,.mp4,.webm" disabled={!ready || pending || status.busy} onChange={e => setFile(e.target.files?.[0])}/></label>
        <details><summary>可选裁剪 · 默认使用完整视频</summary><div className="form-row"><label>起始位置（秒）<input type="number" min="0" max="86400" value={start} disabled={!ready || pending || status.busy} onChange={e => setStart(e.target.value)}/></label><label>保留时长（秒）<input type="number" min="0" max="86400" placeholder="留空至视频结尾" value={duration} disabled={!ready || pending || status.busy} onChange={e => setDuration(e.target.value)}/></label></div><p className="field-help">例如起始 10 秒、保留 5 秒，会截取原视频的第 10–15 秒。默认起始 0、时长留空，使用完整视频。</p></details>
        <button disabled={!ready || !file || pending || status.busy} onClick={upload}>{progress !== null ? `上传 ${progress}%` : status.busy ? '正在处理…' : '上传并自动处理'}</button>
        {progress !== null && <progress value={progress} max={100} aria-label="上传进度"/>}
        {!ready && <p className="muted small">请先完成左侧第 ① 步。集群就绪后，这里的上传操作会解锁。</p>}
        {status.job && <div className={`job ${status.job.stage === 'failed' ? 'failed' : ''}`} role="status"><strong>{status.job.stage === 'ready' && !active ? '上次已导入，启动集群后可播放' : stages[status.job.stage] ?? status.job.stage}</strong><span>{status.job.name}{status.job.segments > 0 ? ` · ${status.job.segments} 个分片` : ''}</span>{status.job.error && <pre>{status.job.error}</pre>}</div>}
      </section>
    </div>
    {!connected && <p className="notice">本地管理服务尚未连接。请在项目根目录运行 <code>make console</code>，并打开 http://127.0.0.1:8090。</p>}
    {notice && <p className="notice" role="status">{notice}</p>}{error && <p className="notice failed" role="alert">{error}</p>}
    {active && <details className="node-details"><summary>查看节点状态、端口与传输量</summary><section><div className="section-title"><h2>节点与传输</h2><span className="muted small">每 1.5 秒查询实际状态 · Peer 发送量包含本次进程全部请求</span></div><div className="table-scroll"><table><thead><tr><th>节点</th><th>状态</th><th>本机端口</th><th>分片对象</th><th>已存储</th><th>累计发送</th></tr></thead><tbody>{status.nodes.map(n => <tr key={n.id}><td>{n.id}</td><td><span className={`badge ${n.status === 'running' ? 'good' : ''}`}>{n.status === 'running' ? n.details?.state ?? '运行中' : n.status === 'exited' ? '已退出' : '未就绪'}</span></td><td><code>{n.port}</code></td><td>{n.details?.objects ?? '—'}</td><td>{n.kind === 'peer' ? size(n.details?.stored_bytes) : '—'}</td><td>{n.kind === 'peer' ? size(n.details?.uploaded_bytes) : '—'}</td></tr>)}</tbody></table></div></section></details>}
    <section><div className="section-title"><div><span className="step">03 / LIBRARY</span><h2>③ 选择视频播放</h2></div><span className="badge">{videos.length} 个已导入</span></div>{videos.length ? <div className="library">{videos.map(v => <div className="video-item" key={v.id}><div><strong>{v.name}</strong><p className="muted small">{v.segments} 个分片 · 双副本已确认</p><details><summary>查看视频 ID</summary><code>{v.id}</code></details></div><button className="secondary" disabled={!active || !connected} onClick={() => onSelect(v.id, v.name)}>播放</button></div>)}</div> : <p className="muted">完成前两步后，视频会显示在这里。点击“播放”即可，不需要填写参数或视频 ID。</p>}</section>
  </>;
}
