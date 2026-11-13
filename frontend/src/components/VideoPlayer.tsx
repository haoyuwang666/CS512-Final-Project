// TODO(Tian): Keep real decoder metrics separate from headless experiment summaries.
import { useEffect, useRef, useState } from 'react';
import type { MediaConfig, MediaSession } from '../types/api';
import { SegmentBuffer } from '../playback/segmentBuffer';
import { MSEPlayer, segmentAt } from '../playback/msePlayer';
import EventLog from './EventLog';
import type { LogEntry, LogLevel } from '../logging/events';
import { responseError } from '../api/messages';

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
  const [events, setEvents] = useState<LogEntry[]>([]);
  useEffect(() => {
    const video = videoRef.current!;
    const life = new AbortController(); const startedAt = performance.now();
    let alive = true; let sessionID = ''; let buffer: SegmentBuffer | undefined; let player: MSEPlayer | undefined;
    let offsets: number[] = []; let intent = true; let firstPlayed = false; let seeking = false;
    let stallStart: number | null = null; let totalStall = 0;
    let state: View = { ...initial, states: {}, peers: {} };
    let sequence = 0;
    setEvents([]);
    const record = (message: string, level: LogLevel = 'info', segment?: string, node?: string, details?: unknown) => {
      if (!alive) return;
      const entry: LogEntry = { id: `browser-${++sequence}`, time: new Date().toISOString(), level, category: 'Browser playback', message, segment, node, details };
      setEvents(previous => [...previous, entry].slice(-300));
    };
    const publish = (patch: Partial<View> = {}) => {
      state = { ...state, ...patch };
      if (alive) setView({ ...state, states: { ...state.states }, peers: { ...state.peers },
        stall: totalStall + (stallStart === null ? 0 : performance.now() - stallStart) });
    };
    const stopStall = () => { if (stallStart !== null) { totalStall += performance.now() - stallStart; stallStart = null; } };
    const fail = (error: unknown) => {
      if (!alive) return;
      record(error instanceof Error ? error.message : String(error), 'error', undefined, undefined, { error: String(error) });
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
      if (state.status !== 'playing') record(firstPlayed ? 'Playback resumed.' : 'Playback started.', 'success');
      intent = true; seeking = false; stopStall();
      if (!firstPlayed) { firstPlayed = true; state.startup = performance.now() - startedAt; }
      publish({ status: 'playing', hint: '' }); progress();
    };
    const pause = () => { if (!video.ended && alive && state.status !== 'failed') { if (state.status !== 'paused') record('Playback paused.'); intent = false; stopStall(); publish({ status: 'paused' }); } };
    const waiting = () => {
      if (!intent || !alive) return;
      if (state.status !== 'buffering') record('Waiting for playable media.', 'warning');
      if (firstPlayed && !seeking && stallStart === null) stallStart = performance.now();
      publish({ status: 'buffering' });
    };
    const metadata = () => {
      if (video.videoWidth && video.videoHeight && stageRef.current) stageRef.current.style.aspectRatio = `${video.videoWidth} / ${video.videoHeight}`;
    };
    const listeners: Record<string, () => void> = {
      playing, pause, waiting, timeupdate: progress, loadedmetadata: metadata,
      play: () => { intent = true; publish({ hint: '' }); },
      seeking: () => { record(`Seeking to ${video.currentTime.toFixed(2)} seconds.`); seeking = true; stopStall(); if (intent) publish({ status: 'buffering' }); },
      seeked: () => { seeking = false; progress(); if (video.paused) publish({ status: 'paused' }); },
      ended: () => { record('Playback completed.', 'success'); intent = false; stopStall(); publish({ status: 'completed', position: state.duration }); },
      error: () => { if (video.error) fail(new Error(`Browser decoding failed (code ${video.error.code}). Check the continuous fMP4 segments.`)); },
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
        if (!response.ok) throw responseError(body.error?.message ?? `Media HTTP ${response.status}`,response.status);
        const session = body as MediaSession;
        if (!alive) { stopSession(session.session_id); return; }
        sessionID = session.session_id;
        record(`Media session created with ${session.manifest.segments.length} segments.`, 'info', undefined, undefined, { session_id: sessionID, window: session.window, concurrency: session.concurrency });
        let duration = 0;
        offsets = session.manifest.segments.map(segment => { const offset = duration; duration += segment.duration_ms / 1000; return offset; });
        publish({ duration, count: offsets.length, status: 'buffering' });
        buffer = new SegmentBuffer(sessionID, session.manifest, session.window, session.concurrency, notice => {
          if (notice.state !== 'queued') {
            const segment = session.manifest.segments[notice.index]?.segment_id;
            record(`Segment ${notice.index}: ${notice.state}${notice.failures ? ` after ${notice.failures} failed-source attempts` : ''}.`, notice.state === 'failed' ? 'error' : notice.failures ? 'warning' : notice.state === 'ready' ? 'success' : 'info', segment, notice.peer, notice);
          }
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
            intent = false; stopStall(); publish({ status: 'paused', hint: 'Select the player’s Play button to start with audio.' });
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
  return <section className="video-player"><h2>Continuous playback · {view.status}</h2>
    <div ref={stageRef} className="video-stage">
      <video ref={videoRef} controls playsInline preload="auto" aria-label="Video player" />
    </div>
    <button disabled={!view.count} onClick={() => {
      const stage = stageRef.current;
      if (!stage) return;
      const operation = document.fullscreenElement === stage ? document.exitFullscreen() : stage.requestFullscreen?.();
      void operation?.catch(() => setView(current => ({ ...current, hint: 'Fullscreen failed. Try the browser player’s fullscreen control.' })));
    }}>Fullscreen</button>
    <label htmlFor="video-timeline">Video timeline: {format(view.position)} / {format(view.duration)}</label>
    <input id="video-timeline" className="timeline" type="range" min={0} max={view.duration || 1} step={0.1}
      value={view.position} disabled={!view.count || view.status === 'failed'} onChange={event => seekRef.current(Number(event.target.value))}/>
    <p>Segment {view.count ? view.index + 1 : 0} / {view.count} · Continuous MSE timeline · Pause, volume, fullscreen, and seeking supported.</p>
    <button disabled={!view.count || view.status === 'failed'} onClick={() => seekRef.current(0)}>Back to start</button>
    {view.hint && <p role="status">{view.hint}</p>}{view.error && <p role="alert">{view.error}</p>}
    <dl><dt>Browser startup delay</dt><dd>{view.startup === null ? 'Waiting for first playback' : `${view.startup.toFixed(1)} ms`}</dd>
      <dt>Playback stalls (excluding pause / seek)</dt><dd>{view.stall.toFixed(1)} ms</dd>
      <dt>Browser-verified bytes</dt><dd>{view.bytes}</dd><dt>Peer requests / failed-source attempts</dt><dd>{view.attempts} / {view.failures}</dd></dl>
    <ul>{Object.entries(view.states).map(([index, status]) => <li key={index}>#{index} · {status} · {view.peers[+index] ?? 'Awaiting source'}</li>)}</ul>
    <p>Segments share one audio/video timeline without reloading the media source. Insufficient bandwidth can still cause buffering.</p>
    <EventLog entries={events} title="Playback events" fileName="browser-playback" description="Browser-observed activity and verified segment delivery. Latest 300 events are retained; this is separate from headless simulation metrics."/>
  </section>;
}
