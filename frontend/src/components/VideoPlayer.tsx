// TODO(Tian): Keep real decoder metrics separate from headless experiment summaries.
import { useEffect, useRef, useState } from 'react';
import type { MediaConfig, MediaSession } from '../types/api';
import { SegmentBuffer } from '../playback/segmentBuffer';
import { MSEPlayer, segmentAt } from '../playback/msePlayer';

interface View {
  status: string; position: number; duration: number; index: number; count: number;
  startup: number | null; stall: number; bytes: number; attempts: number; failures: number;
  states: Record<number, string>; peers: Record<number, string>; error: string; hint: string;
}
const initial: View = { status: 'loading', position: 0, duration: 0, index: 0, count: 0,
  startup: null, stall: 0, bytes: 0, attempts: 0, failures: 0, states: {}, peers: {}, error: '', hint: '' };

export default function VideoPlayer({ config }: { config: MediaConfig }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const seekRef = useRef<(seconds: number) => void>(() => {});
  const [view, setView] = useState<View>(initial);
  useEffect(() => {
    const video = videoRef.current!;
    const life = new AbortController(); const startedAt = performance.now();
    let alive = true; let sessionID = ''; let buffer: SegmentBuffer | undefined; let player: MSEPlayer | undefined;
    let offsets: number[] = []; let intent = true; let firstPlayed = false; let seeking = false;
    let stallStart: number | null = null; let totalStall = 0;
    let state: View = { ...initial, states: {}, peers: {} };
    const publish = (patch: Partial<View> = {}) => {
      state = { ...state, ...patch };
      if (alive) setView({ ...state, states: { ...state.states }, peers: { ...state.peers },
        stall: totalStall + (stallStart === null ? 0 : performance.now() - stallStart) });
    };
    const stopStall = () => { if (stallStart !== null) { totalStall += performance.now() - stallStart; stallStart = null; } };
    const fail = (error: unknown) => {
      if (!alive) return;
      intent = false; stopStall(); buffer?.close(); player?.close();
      publish({ status: 'failed', error: error instanceof Error ? error.message : String(error) });
    };
    const progress = () => {
      if (!offsets.length) return;
      const index = segmentAt(offsets, video.currentTime);
      const states = Object.fromEntries(Object.entries(state.states).filter(([i]) => +i >= index - 2 && +i < index + config.window));
      const peers = Object.fromEntries(Object.entries(state.peers).filter(([i]) => +i in states));
      if (states[index]) states[index] = 'playing';
      publish({ index, position: Math.min(video.currentTime, state.duration), states, peers });
    };
    const playing = () => {
      intent = true; seeking = false; stopStall();
      if (!firstPlayed) { firstPlayed = true; state.startup = performance.now() - startedAt; }
      publish({ status: 'playing', hint: '' }); progress();
    };
    const pause = () => { if (!video.ended && alive && state.status !== 'failed') { intent = false; stopStall(); publish({ status: 'paused' }); } };
    const waiting = () => {
      if (!intent || !alive) return;
      if (firstPlayed && !seeking && stallStart === null) stallStart = performance.now();
      publish({ status: 'buffering' });
    };
    const metadata = () => {
      if (video.videoWidth && video.videoHeight && stageRef.current) stageRef.current.style.aspectRatio = `${video.videoWidth} / ${video.videoHeight}`;
    };
    const listeners: Record<string, () => void> = {
      playing, pause, waiting, timeupdate: progress, loadedmetadata: metadata,
      play: () => { intent = true; publish({ hint: '' }); },
      seeking: () => { seeking = true; stopStall(); if (intent) publish({ status: 'buffering' }); },
      seeked: () => { seeking = false; progress(); if (video.paused) publish({ status: 'paused' }); },
      ended: () => { intent = false; stopStall(); publish({ status: 'completed', position: state.duration }); },
      error: () => { if (video.error) fail(new Error(`浏览器连续流解码失败（错误 ${video.error.code}），请检查连续 fMP4 分片。`)); },
    };
    for (const [name, handler] of Object.entries(listeners)) video.addEventListener(name, handler, { signal: life.signal });
    seekRef.current = seconds => player?.seek(seconds);
    const stopSession = (id: string) => { void fetch(`/api/v1/media/sessions/${id}/stop`, { method: 'POST', keepalive: true }).catch(() => {}); };
    const unload = () => { if (sessionID) stopSession(sessionID); };
    window.addEventListener('pagehide', unload);
    void (async () => {
      try {
        const response = await fetch('/api/v1/media/sessions', { method: 'POST', signal: life.signal,
          headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(config) });
        const body = await response.json();
        if (!response.ok) throw new Error(body.error?.message ?? `Media HTTP ${response.status}`);
        const session = body as MediaSession;
        if (!alive) { stopSession(session.session_id); return; }
        sessionID = session.session_id;
        let duration = 0;
        offsets = session.manifest.segments.map(segment => { const offset = duration; duration += segment.duration_ms / 1000; return offset; });
        publish({ duration, count: offsets.length, status: 'buffering' });
        buffer = new SegmentBuffer(sessionID, session.manifest, session.window, session.concurrency, notice => {
          publish({ states: { ...state.states, [notice.index]: notice.state },
            peers: notice.peer ? { ...state.peers, [notice.index]: notice.peer } : state.peers,
            bytes: state.bytes + (notice.bytes ?? 0), attempts: state.attempts + (notice.attempts ?? 0), failures: state.failures + (notice.failures ?? 0) });
        });
        player = new MSEPlayer(video, buffer, session, (index, status) => {
          publish({ states: { ...state.states, [index]: status } });
        }, fail);
        await player.ready;
        if (!alive) return;
        try { await video.play(); }
        catch (error) {
          if (!alive) return;
          if (error instanceof DOMException && error.name === 'NotAllowedError') {
            intent = false; stopStall(); publish({ status: 'paused', hint: '点击播放器中的播放按钮开始有声播放。' });
          } else if (!(error instanceof DOMException && error.name === 'AbortError')) throw error;
        }
      } catch (error) { if (alive && !life.signal.aborted) fail(error); }
    })();
    const timer = window.setInterval(() => { if (stallStart !== null) publish(); }, 250);
    return () => {
      alive = false; life.abort(); player?.close(); buffer?.close();
      window.clearInterval(timer); window.removeEventListener('pagehide', unload);
      if (sessionID) stopSession(sessionID);
      seekRef.current = () => {};
    };
  }, [config]);
  const format = (seconds: number) => `${Math.floor(seconds / 60)}:${Math.floor(seconds % 60).toString().padStart(2, '0')}`;
  return <section className="video-player"><h2>连续视频播放 · {view.status}</h2>
    <div ref={stageRef} className="video-stage">
      <video ref={videoRef} controls playsInline preload="auto" aria-label="视频播放器" />
    </div>
    <button disabled={!view.count} onClick={() => {
      const stage = stageRef.current;
      if (!stage) return;
      const operation = document.fullscreenElement === stage ? document.exitFullscreen() : stage.requestFullscreen?.();
      void operation?.catch(() => setView(current => ({ ...current, hint: '全屏切换失败，请使用浏览器播放器的全屏控件。' })));
    }}>全屏播放</button>
    <label htmlFor="video-timeline">整段视频进度：{format(view.position)} / {format(view.duration)}</label>
    <input id="video-timeline" className="timeline" type="range" min={0} max={view.duration || 1} step={0.1}
      value={view.position} disabled={!view.count || view.status === 'failed'} onChange={event => seekRef.current(Number(event.target.value))}/>
    <p>分片 {view.count ? view.index + 1 : 0} / {view.count} · MSE 连续时间线 · 支持暂停、音量、全屏和整段跳转。</p>
    <button disabled={!view.count || view.status === 'failed'} onClick={() => seekRef.current(0)}>回到开头</button>
    {view.hint && <p role="status">{view.hint}</p>}{view.error && <p role="alert">{view.error}</p>}
    <dl><dt>浏览器实际启动延迟</dt><dd>{view.startup === null ? '等待首次播放' : `${view.startup.toFixed(1)} ms`}</dd>
      <dt>播放等待（不含手动暂停/跳转）</dt><dd>{view.stall.toFixed(1)} ms</dd>
      <dt>浏览器已验证字节</dt><dd>{view.bytes}</dd><dt>已交付分片的 Peer 请求 / 失败切源</dt><dd>{view.attempts} / {view.failures}</dd></dl>
    <ul>{Object.entries(view.states).map(([index, status]) => <li key={index}>#{index} · {status} · {view.peers[+index] ?? '等待来源'}</li>)}</ul>
    <p>分片沿同一音视频时间线追加，换段不会更换视频源或重建解码器；网络供给不足时仍可能缓冲。</p>
  </section>;
}
