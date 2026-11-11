// TODO(Tian): Expand this minimal single-page dashboard after baseline experiments work.
import { useState } from 'react';
import { request } from '../api/client';
import { useSession } from '../hooks/useSession';
import MetricsPanel from '../components/MetricsPanel';
export default function Dashboard() {
  const [video, setVideo] = useState('fixture'); const [id, setID] = useState(''); const [error, setError] = useState(''); const [busy, setBusy] = useState(false);
  const { session, error: connectionError } = useSession(id);
  async function start() { setError(''); setBusy(true); try { const out = await request<{ session_id: string }>('/sessions', 'POST', { video_id: video, strategy: 'round_robin' }); setID(out.session_id); } catch (err) { setError(String(err)); } finally { setBusy(false); } }
  async function stop() { try { await request(`/sessions/${id}/stop`, 'POST', {}); } catch (err) { setError(String(err)); } }
  return <main><p className="eyebrow">COMPSCI 512 · Starter</p><h1>Cooperative Video</h1><p>真实 HTTP 下载 · 模拟媒体时钟 · Round Robin 基线</p>
    <section><label htmlFor="video">已导入的视频 ID</label><input id="video" value={video} onChange={event => setVideo(event.target.value)} /><button disabled={busy || session?.summary.status === 'running'} onClick={start}>开始 Session</button><button disabled={!id || session?.summary.status !== 'running'} onClick={stop}>停止</button></section>
    {(error || connectionError) && <p role="alert">{error || connectionError}</p>}
    {session && <><MetricsPanel summary={session.summary}/><section><h2>真实后端事件</h2><ol>{session.events.slice(-30).map(event => <li key={event.sequence}>{event.type} {event.segment_id} {event.peer_id} {event.message}</li>)}</ol></section></>}
    <p>Raft/Peer 面板、Buffer 时间轴和优化策略尚待实现；当前不显示替代数据。</p>
  </main>;
}
