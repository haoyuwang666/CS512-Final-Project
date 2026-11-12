// TODO(Tian): Keep playback controls aligned with reproducible CLI configurations.
import { useState } from 'react';
import { request } from '../api/client';
import { useSession } from '../hooks/useSession';
import MetricsPanel from '../components/MetricsPanel';
import VideoPlayer from '../components/VideoPlayer';
import type { MediaConfig } from '../types/api';
export default function Dashboard() {
  const [video, setVideo] = useState('my-video');
  const [mode, setMode] = useState('video');
  const [media, setMedia] = useState<{ config: MediaConfig; key: number }>();
  const [strategy, setStrategy] = useState('round_robin');
  const [window, setWindow] = useState(3);
  const [concurrency, setConcurrency] = useState(2);
  const [startup, setStartup] = useState(0);
  const [id, setID] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const { session, error: connectionError } = useSession(id);
  const running = session?.summary.status === 'running';
  async function start() {
    setError('');
    if (mode === 'video') {
      setMedia({ config: { video_id: video, strategy, window, concurrency, startup_buffer_ms: startup }, key: Date.now() });
      return;
    }
    setBusy(true);
    try {
      const out = await request<{ session_id: string }>('/sessions', 'POST', {
        video_id: video, strategy, window, concurrency, startup_buffer_ms: startup,
      });
      setID(out.session_id);
    } catch (err) { setError(String(err)); } finally { setBusy(false); }
  }
  async function stop() {
    if (mode === 'video') { setMedia(undefined); return; }
    try { await request(`/sessions/${id}/stop`, 'POST', {}); }
    catch (err) { setError(String(err)); }
  }
  return <main><p className="eyebrow">COMPSCI 512 · Cooperative Video</p><h1>Cooperative Video</h1>
    <p>分布式分片下载 · 浏览器视频播放 · Round Robin / Deadline</p>
    <section><label htmlFor="mode">运行模式</label>
      <select id="mode" value={mode} disabled={running} onChange={event => { setMedia(undefined); setMode(event.target.value); }}>
        <option value="video">视频播放（画面与声音）</option><option value="simulation">Headless 模拟实验</option>
      </select>
      <label htmlFor="video">已导入的视频 ID</label>
      <input id="video" value={video} onChange={event => setVideo(event.target.value)} />
      <label htmlFor="strategy">下载策略</label>
      <select id="strategy" value={strategy} onChange={event => setStrategy(event.target.value)}>
        <option value="round_robin">Round Robin</option><option value="deadline">Deadline</option>
      </select>
      <label htmlFor="window">预取窗口（分片）</label>
      <input id="window" type="number" min={1} max={64} value={window} onChange={event => setWindow(Number(event.target.value))}/>
      <label htmlFor="concurrency">下载并发</label>
      <input id="concurrency" type="number" min={1} max={window} value={concurrency} onChange={event => setConcurrency(Number(event.target.value))}/>
      <label htmlFor="startup">启动缓冲（ms，0 表示首段即可启动；须能放入预取窗口）</label>
      <input id="startup" type="number" min={0} max={3600000} value={startup} onChange={event => setStartup(Number(event.target.value))}/>
      <button disabled={busy || running} onClick={start}>{mode === 'video' ? '播放视频 / 重新加载' : '开始模拟 Session'}</button>
      <button disabled={mode === 'video' ? !media : !id || !running} onClick={stop}>停止</button>
    </section>
    {(error || connectionError) && <p role="alert">{error || connectionError}</p>}
    {mode === 'video' && media && <VideoPlayer key={media.key} config={media.config}/>}
    {mode === 'simulation' && session && <><MetricsPanel session={session}/><section><h2>后端事件</h2><ol>
      {session.events.slice(-30).map(event => <li key={event.sequence}>
        {event.type} {event.segment_id} {event.peer_id} {event.message}
      </li>)}
    </ol></section></>}
  </main>;
}
