// TODO(Tian): Extend codec negotiation and eviction policy with real-browser regressions.
import type { MediaSession } from '../types/api.ts';
import { SegmentBuffer } from './segmentBuffer.ts';

export function segmentAt(offsets: number[], time: number): number {
  let low = 0; let high = offsets.length;
  while (low + 1 < high) { const middle = (low + high) >>> 1; if (offsets[middle] <= time) low = middle; else high = middle; }
  return low;
}

// Exactly one MediaSource URL and one SourceBuffer live for the entire playback.
// SourceBuffer mutations are serialized; seeking changes demand, never video.src.
export class MSEPlayer {
  readonly ready: Promise<void>;
  private resolveReady!: () => void;
  private rejectReady!: (error: unknown) => void;
  private initialized = false;
  private startupComplete = false;
  private closed = false;
  private running = false;
  private again = false;
  private appended = new Set<number>();
  private offsets: number[];
  private source: MediaSource;
  private sourceBuffer?: SourceBuffer;
  private url: string;
  private abort = new AbortController();
  private video: HTMLVideoElement;
  private buffer: SegmentBuffer;
  private session: MediaSession;
  private changed: (index: number, state: string) => void;
  private failed: (error: unknown) => void;

  constructor(video: HTMLVideoElement, buffer: SegmentBuffer, session: MediaSession,
    changed: (index: number, state: string) => void, failed: (error: unknown) => void) {
    const media = session.manifest.media;
    if (!media || !session.manifest.representation.startsWith('continuous-fmp4-')) throw new Error('Regenerate continuous fMP4 segments from the source video and import them with a new ID.');
    if (typeof MediaSource === 'undefined' || !MediaSource.isTypeSupported(media.mime_type)) throw new Error('This browser does not support the video’s MSE/H.264/AAC format. Try a compatible browser such as Edge.');
    this.video = video; this.buffer = buffer; this.session = session; this.changed = changed; this.failed = failed;
    let time = 0;
    this.offsets = session.manifest.segments.map(segment => { const at = time; time += segment.duration_ms / 1000; return at; });
    this.ready = new Promise((resolve, reject) => { this.resolveReady = resolve; this.rejectReady = reject; });
    void this.ready.catch(() => {});
    this.source = new MediaSource();
    this.url = URL.createObjectURL(this.source);
    this.source.addEventListener('sourceopen', () => {
      if (this.closed || this.initialized) return;
      try {
        this.sourceBuffer = this.source.addSourceBuffer(media.mime_type);
        this.sourceBuffer.mode = 'segments';
        this.source.duration = media.duration_ms / 1000;
        this.initialized = true; this.pump();
      } catch (error) { this.error(error); }
    }, { signal: this.abort.signal });
    for (const event of ['timeupdate', 'seeking', 'seeked', 'play']) {
      video.addEventListener(event, () => this.pump(), { signal: this.abort.signal });
    }
    video.src = this.url;
    video.load();
  }
  seek(seconds: number) {
    if (this.closed) return;
    const duration = this.session.manifest.media!.duration_ms / 1000;
    this.video.currentTime = Math.max(0, Math.min(seconds, duration - 0.001));
    this.pump();
  }
  close() {
    if (this.closed) return;
    this.closed = true;
    this.rejectReady(new DOMException('Player closed', 'AbortError'));
    this.abort.abort(); this.buffer.close();
    try { if (this.sourceBuffer?.updating && this.source.readyState === 'open') this.sourceBuffer.abort(); } catch { /* Already detached. */ }
    this.video.pause(); this.video.removeAttribute('src'); this.video.load();
    URL.revokeObjectURL(this.url);
    this.appended.clear();
  }
  private error(error: unknown) {
    if (this.closed) return;
    this.rejectReady(error); this.failed(error); this.close();
  }
  private pump() {
    if (this.closed || !this.initialized) return;
    // Cancel obsolete downloads immediately, even while the append loop awaits one.
    const base = segmentAt(this.offsets, this.video.currentTime);
    this.buffer.setWindow(base, this.appended);
    if (this.running) { this.again = true; return; }
    this.running = true;
    void this.fill().catch(error => this.error(error)).finally(() => {
      this.running = false;
      if (this.again) { this.again = false; this.pump(); }
    });
  }
  private async fill() {
    const count = this.session.manifest.segments.length;
    while (!this.closed) {
      const base = segmentAt(this.offsets, this.video.currentTime);
      await this.trim(base);
      this.buffer.setWindow(base, this.appended);
      const upper = Math.min(count, base + this.session.window);
      let index = base;
      while (index < upper && this.appended.has(index)) index++;
      if (index === upper) {
        if (upper === count && this.source.readyState === 'open') this.source.endOfStream();
        return;
      }
      let data: ArrayBuffer;
      try { data = await this.buffer.get(index); }
      catch (error) {
        if (this.closed) return;
        // A seek may discard the awaited prefetch entry without failing playback.
        const now = segmentAt(this.offsets, this.video.currentTime);
        if (index < now || index >= now + this.session.window || (error instanceof DOMException && error.name === 'AbortError')) continue;
        throw error;
      }
      if (this.closed) return;
      const now = segmentAt(this.offsets, this.video.currentTime);
      if (index < now || index >= now + this.session.window) continue;
      try { await this.mutate(() => this.sourceBuffer!.appendBuffer(data)); }
      catch (error) {
        if (!(error instanceof DOMException && error.name === 'QuotaExceededError')) throw error;
        // Under memory pressure sacrifice back-buffer, never rewrite timestamps.
        await this.trim(now, true);
        await this.mutate(() => this.sourceBuffer!.appendBuffer(data));
      }
      if (this.closed) return;
      this.appended.add(index); this.buffer.release(index); this.changed(index, 'appended');
      if (!this.startupComplete) {
        let duration = 0; let enough = false;
        for (let i = now; i < Math.min(count, now + this.session.window); i++) {
          if (!this.appended.has(i)) break;
          duration += this.session.manifest.segments[i].duration_ms;
          if (duration >= this.session.startup_buffer_ms || i === count - 1) { enough = true; break; }
        }
        if (enough) { this.startupComplete = true; this.resolveReady(); }
      }
    }
  }
  private reopen() {
    // SourceBuffer timestampOffset reopens an ended MediaSource. Zero is the
    // original offset: tfdt values still define the shared encoded timeline.
    if (this.source.readyState === 'ended') this.sourceBuffer!.timestampOffset = 0;
  }
  private async trim(base: number, aggressive = false) {
    const lower = Math.max(0, base - (aggressive ? 0 : 2));
    const upper = Math.min(this.offsets.length, base + this.session.window);
    if ([...this.appended].some(i => i < lower)) {
      await this.mutate(() => this.sourceBuffer!.remove(0, this.offsets[lower]));
      for (const index of this.appended) if (index < lower) this.appended.delete(index);
    }
    if (upper < this.offsets.length && [...this.appended].some(i => i >= upper)) {
      await this.mutate(() => this.sourceBuffer!.remove(this.offsets[upper], Infinity));
      for (const index of this.appended) if (index >= upper) this.appended.delete(index);
    }
  }
  private mutate(operation: () => void): Promise<void> {
    if (this.closed) return Promise.reject(new DOMException('Player closed', 'AbortError'));
    this.reopen();
    const sourceBuffer = this.sourceBuffer!;
    return new Promise((resolve, reject) => {
      const done = () => { cleanup(); resolve(); };
      const fail = () => { cleanup(); reject(new Error('Media buffer update failed. Check segment format and timestamps.')); };
      const cancel = () => { cleanup(); reject(new DOMException('Player closed', 'AbortError')); };
      const cleanup = () => {
        sourceBuffer.removeEventListener('updateend', done); sourceBuffer.removeEventListener('error', fail);
        sourceBuffer.removeEventListener('abort', fail); this.abort.signal.removeEventListener('abort', cancel);
      };
      sourceBuffer.addEventListener('updateend', done, { once: true });
      sourceBuffer.addEventListener('error', fail, { once: true });
      sourceBuffer.addEventListener('abort', fail, { once: true });
      this.abort.signal.addEventListener('abort', cancel, { once: true });
      try { operation(); } catch (error) { cleanup(); reject(error); }
    });
  }
}
