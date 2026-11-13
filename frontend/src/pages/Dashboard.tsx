// TODO(Tian): Keep playback controls aligned with reproducible CLI configurations.
import { useState } from 'react';
import { request } from '../api/client';
import { useSession } from '../hooks/useSession';
import MetricsPanel from '../components/MetricsPanel';
import VideoPlayer from '../components/VideoPlayer';
import LocalConsole from '../components/LocalConsole';
import ReplicaCleanup from '../components/ReplicaCleanup';
import type { MediaConfig } from '../types/api';
export default function Dashboard() {
  const [video, setVideo] = useState('');
  const [videoName, setVideoName] = useState('');
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
    if (!video.trim()) { setError('请先从视频库选择视频。'); return; }
    if (!Number.isInteger(window) || window < 1 || window > 64 || !Number.isInteger(concurrency) || concurrency < 1 || concurrency > window || !Number.isFinite(startup) || startup < 0) { setError('请检查高级设置：并发不能超过预取分片数，参数须在标注范围内。'); return; }
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
  return <main><header><div><p className="eyebrow">COMPSCI 512 / LOCAL WORKSPACE</p><h1>Cooperative Video<span className="accent">.</span></h1>
    <p className="muted">启动节点，上传视频，查看每个分片如何抵达播放器。</p></div><span className="badge">本机控制台</span></header>
    <LocalConsole onReset={() => { setMedia(undefined); setVideo(''); setVideoName(''); }} onSelect={(value, name) => {
      setError(''); setVideo(value); setVideoName(name); setMode('video');
      setMedia({ config: { video_id: value, strategy, window, concurrency, startup_buffer_ms: startup }, key: Date.now() });
      document.getElementById('playback')?.scrollIntoView({ behavior: 'smooth' });
    }}/>
    {video && <ReplicaCleanup key={video} video={video}/> }
    <section id="playback"><div className="section-title"><div><span className="step">PLAYER</span><h2>{videoName || '视频播放器'}</h2></div><span className="badge">{video ? '已选择视频' : '等待选择'}</span></div>
      <p className="muted">{video ? '使用下方默认设置即可观看。调整高级设置后，点击“应用设置并重新播放”。' : '在上方视频库点击“播放”，会自动填入视频并开始加载，无需手动输入 ID。'}</p>
      <details className="advanced"><summary>高级设置 · 下载与缓冲（可保持默认）</summary>
        <p className="muted small">这些设置只影响播放时如何下载，不改变已上传的视频和副本分布。修改后需重新播放才会生效。</p>
        <div className="playback-fields"><div><label htmlFor="strategy">从哪个 Peer 下载</label>
          <select id="strategy" value={strategy} onChange={event => setStrategy(event.target.value)}>
            <option value="round_robin">轮换来源（Round Robin，默认）</option><option value="deadline">按预计速度选择（Deadline）</option>
          </select><p className="field-help">轮换来源按固定顺序选副本；Deadline 结合吞吐估计等信息排序，尝试及时取回分片。请求失败都会尝试其他副本。</p></div>
          <div><label htmlFor="window">提前准备多少片</label>
          <input id="window" type="number" min={1} max={64} value={window} onChange={event => { const value = Number(event.target.value); setWindow(value); setConcurrency(current => Math.min(current, Math.max(1, value))); }}/>
          <p className="field-help">默认 3 片。每片约 2 秒，相当于准备约 6 秒内容；越大通常占用越多内存和网络。</p></div>
          <div><label htmlFor="concurrency">同时下载多少片</label>
          <input id="concurrency" type="number" min={1} max={window} value={concurrency} onChange={event => setConcurrency(Number(event.target.value))}/>
          <p className="field-help">默认 2 片，同时发起最多 2 个下载请求。不是 Peer 数量，且不能超过上面的预取片数。</p></div>
          <div><label htmlFor="startup">开始播放前先准备几秒</label>
          <input id="startup" type="number" min={0} max={3600} step={0.1} value={startup / 1000} onChange={event => setStartup(Math.round(Number(event.target.value) * 1000))}/>
          <p className="field-help">默认 0：第一片准备好就播放。填 2 表示先准备至少 2 秒；等待更久，可能减少后续卡顿，且需能放进预取窗口。</p></div>
        </div>
        <details className="experiment-settings"><summary>实验模式与手动视频 ID</summary><div className="form-row"><div><label htmlFor="mode">运行模式</label>
          <select id="mode" value={mode} disabled={running} onChange={event => { setMedia(undefined); setMode(event.target.value); }}><option value="video">真实视频播放</option><option value="simulation">仅传输模拟（不显示画面）</option></select>
          </div><div><label htmlFor="video">已导入的视频 ID</label><input id="video" placeholder="从视频库选择后自动填写" value={video} onChange={event => { setVideo(event.target.value); setVideoName('手动指定视频'); }}/></div></div>
          <p className="field-help">模拟模式用于实验指标；观看视频请保留“真实视频播放”。</p></details>
      </details>
      <div className="actions"><button disabled={!video || busy || running} onClick={start}>{mode === 'video' ? '应用设置并重新播放' : '开始传输模拟'}</button>
      <button className="secondary" disabled={mode === 'video' ? !media : !id || !running} onClick={stop}>停止播放</button></div>
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
