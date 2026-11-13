// TODO(Tian): Keep playback controls aligned with reproducible CLI configurations.
import { useState } from 'react';
import { request } from '../api/client';
import { useSession } from '../hooks/useSession';
import MetricsPanel from '../components/MetricsPanel';
import VideoPlayer from '../components/VideoPlayer';
import LocalConsole from '../components/LocalConsole';
import ReplicaCleanup from '../components/ReplicaCleanup';
import type { MediaConfig } from '../types/api';
import EventLog from '../components/EventLog';
import { playbackEntry } from '../logging/events';
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
    if (!video.trim()) { setError('Select a video from the library first.'); return; }
    if (!Number.isInteger(window) || window < 1 || window > 64 || !Number.isInteger(concurrency) || concurrency < 1 || concurrency > window || !Number.isFinite(startup) || startup < 0) { setError('Check advanced settings. Concurrent downloads cannot exceed the prefetch window.'); return; }
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
    <p className="muted">Start the cluster, upload a video, and follow each segment to the player.</p></div><span className="badge">Local console</span></header>
    <LocalConsole onReset={() => { setMedia(undefined); setVideo(''); setVideoName(''); }} onSelect={(value, name) => {
      setError(''); setVideo(value); setVideoName(name); setMode('video');
      setMedia({ config: { video_id: value, strategy, window, concurrency, startup_buffer_ms: startup }, key: Date.now() });
      document.getElementById('playback')?.scrollIntoView({ behavior: 'smooth' });
    }}/>
    {video && <ReplicaCleanup key={video} video={video}/> }
    <section id="playback"><div className="section-title"><div><span className="step">PLAYER</span><h2>{videoName || 'Video player'}</h2></div><span className="badge">{video ? 'Video selected' : 'No video selected'}</span></div>
      <p className="muted">{video ? 'The default settings are ready to use. After changing settings, select Apply and restart.' : 'Select Play in the library to load a video. No manual ID is needed.'}</p>
      <details className="advanced"><summary>Advanced settings · Download and buffer</summary>
        <p className="muted small">These settings affect downloads during playback. Uploaded videos and replica placement stay the same. Restart playback to apply changes.</p>
        <div className="playback-fields"><div><label htmlFor="strategy">Download strategy</label>
          <select id="strategy" value={strategy} onChange={event => setStrategy(event.target.value)}>
            <option value="round_robin">Round Robin (default)</option><option value="deadline">Deadline-aware selection</option>
          </select><p className="field-help">Round Robin rotates through replicas. Deadline ranks sources using throughput estimates. Both try another replica after a failed request.</p></div>
          <div><label htmlFor="window">Prefetch window (segments)</label>
          <input id="window" type="number" min={1} max={64} value={window} onChange={event => { const value = Number(event.target.value); setWindow(value); setConcurrency(current => Math.min(current, Math.max(1, value))); }}/>
          <p className="field-help">Default: 3 segments, roughly 6 seconds for 2-second segments. Larger windows use more memory and bandwidth.</p></div>
          <div><label htmlFor="concurrency">Concurrent downloads</label>
          <input id="concurrency" type="number" min={1} max={window} value={concurrency} onChange={event => setConcurrency(Number(event.target.value))}/>
          <p className="field-help">Default: 2 simultaneous requests. This is not the number of peers and cannot exceed the prefetch window.</p></div>
          <div><label htmlFor="startup">Startup buffer (seconds)</label>
          <input id="startup" type="number" min={0} max={3600} step={0.1} value={startup / 1000} onChange={event => setStartup(Math.round(Number(event.target.value) * 1000))}/>
          <p className="field-help">Default: 0 starts after the first segment. A value of 2 buffers at least 2 seconds first. The startup buffer must fit within the prefetch window.</p></div>
        </div>
        <details className="experiment-settings"><summary>Experiment mode and manual video ID</summary><div className="form-row"><div><label htmlFor="mode">Playback mode</label>
          <select id="mode" value={mode} disabled={running} onChange={event => { setMedia(undefined); setMode(event.target.value); }}><option value="video">Video playback</option><option value="simulation">Headless simulation (no video)</option></select>
          </div><div><label htmlFor="video">Imported video ID</label><input id="video" placeholder="Filled automatically from the library" value={video} onChange={event => { setVideo(event.target.value); setVideoName('Manually selected video'); }}/></div></div>
          <p className="field-help">Simulation measures transport metrics. Use Video playback to watch the video.</p></details>
      </details>
      <div className="actions"><button disabled={!video || busy || running} onClick={start}>{mode === 'video' ? 'Apply and restart' : 'Start simulation'}</button>
      <button className="secondary" disabled={mode === 'video' ? !media : !id || !running} onClick={stop}>Stop playback</button></div>
    </section>
    {(error || connectionError) && <p role="alert">{error || connectionError}</p>}
    {mode === 'video' && media && <VideoPlayer key={media.key} config={media.config}/>}
    {mode === 'simulation' && session && <><MetricsPanel session={session}/><section>
      <EventLog key={session.session_id} entries={session.events.map(playbackEntry)} title="Session events" fileName={`session-${session.session_id}`}
        description="Backend transport and simulated playback events. Timestamps come from the session; raw payloads are available in Details."/>
    </section></>}
  </main>;
}
