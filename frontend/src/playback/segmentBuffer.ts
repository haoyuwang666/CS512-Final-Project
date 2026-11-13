// TODO(Tian): Preserve verified, bounded prefetch as new transports are introduced.
import type { MediaManifest } from '../types/api.ts';
import { responseError } from '../api/messages.ts';

export interface SegmentNotice {
  index: number; state: string; peer?: string; bytes?: number; attempts?: number; failures?: number;
}
interface Entry {
  controller: AbortController;
  promise: Promise<ArrayBuffer>;
  resolve: (data: ArrayBuffer) => void;
  reject: (error: unknown) => void;
}

// Seeking or stopping must also cancel time spent waiting for a new leader.
function retryDelay(signal: AbortSignal, milliseconds: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const abort = () => { clearTimeout(timer); reject(new DOMException('Obsolete segment', 'AbortError')); };
    const timer = setTimeout(() => { signal.removeEventListener('abort', abort); resolve(); }, milliseconds);
    if (signal.aborted) abort();
    else signal.addEventListener('abort', abort, { once: true });
  });
}

// The browser holds at most `window` verified fMP4 buffers. Seeking discards
// obsolete work; even cancelled requests occupy a worker until fetch settles.
export class SegmentBuffer {
  private entries = new Map<number, Entry>();
  private queue: number[] = [];
  private active = 0;
  private closed = false;
  private sessionID: string;
  private manifest: MediaManifest;
  private windowSize: number;
  private concurrency: number;
  private notify: (notice: SegmentNotice) => void;

  constructor(sessionID: string, manifest: MediaManifest, windowSize: number, concurrency: number,
    notify: (notice: SegmentNotice) => void) {
    this.sessionID = sessionID; this.manifest = manifest;
    this.windowSize = windowSize; this.concurrency = concurrency; this.notify = notify;
  }
  setWindow(index: number, appended: ReadonlySet<number> = new Set()) {
    if (this.closed) return;
    for (const [i, entry] of this.entries) {
      if (i < index || i >= index + this.windowSize || appended.has(i)) {
        entry.controller.abort(); entry.reject(new DOMException('Obsolete segment', 'AbortError'));
        this.entries.delete(i);
      }
    }
    this.queue = this.queue.filter(i => this.entries.has(i));
    for (let i = index; i < Math.min(this.manifest.segments.length, index + this.windowSize); i++) {
      if (this.entries.has(i) || appended.has(i)) continue;
      let resolve!: Entry['resolve']; let reject!: Entry['reject'];
      const promise = new Promise<ArrayBuffer>((yes, no) => { resolve = yes; reject = no; });
      // Prefetch errors are surfaced when the consumer reaches that segment.
      void promise.catch(() => {});
      this.entries.set(i, { controller: new AbortController(), promise, resolve, reject });
      this.queue.push(i); this.notify({ index: i, state: 'queued' });
    }
    this.pump();
  }
  get(index: number): Promise<ArrayBuffer> {
    const entry = this.entries.get(index);
    return entry ? entry.promise : Promise.reject(new Error('Segment is outside the buffer window'));
  }
  // SourceBuffer owns appended bytes; discard their JS copies immediately.
  release(index: number) {
    const entry = this.entries.get(index);
    if (!entry) return;
    entry.controller.abort();
    this.entries.delete(index);
    this.queue = this.queue.filter(i => i !== index);
  }
  close() {
    this.closed = true;
    for (const entry of this.entries.values()) {
      entry.controller.abort(); entry.reject(new DOMException('Player closed', 'AbortError'));
    }
    this.entries.clear(); this.queue = [];
  }
  private pump() {
    while (!this.closed && this.active < this.concurrency && this.queue.length) {
      const index = this.queue.shift()!;
      const entry = this.entries.get(index);
      if (!entry) continue;
      this.active++;
      void this.download(index, entry).finally(() => { this.active--; this.pump(); });
    }
  }
  private async download(index: number, entry: Entry) {
    const segment = this.manifest.segments[index];
    const signal = entry.controller.signal;
    try {
      this.notify({ index, state: 'downloading' });
      let response: Response;
      for (let attempt = 0; ; attempt++) {
        response = await fetch(`/api/v1/media/sessions/${this.sessionID}/segments/${index}`, { signal });
        if (response.ok) break;
        const body = await response.json().catch(() => null);
        // Keep the promise pending so MSE retains its source and buffered frames.
        // Only explicit temporary metadata failures qualify; corrupt bytes and
        // expired sessions must not be retried or silently accepted.
        if (response.status !== 503 || body?.error?.retryable !== true || attempt >= 2) {
          throw responseError(body?.error?.message ?? `Media HTTP ${response.status}`, response.status);
        }
        await retryDelay(signal, 500 * (attempt + 1));
      }
      const data = await response.arrayBuffer();
      if (data.byteLength !== segment.size_bytes) throw new Error(`Segment ${index} size verification failed`);
      if (!globalThis.crypto?.subtle) throw new Error('Verification requires localhost or a secure HTTPS connection');
      const digest = await crypto.subtle.digest('SHA-256', data);
      const hash = Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2, '0')).join('');
      if (hash !== segment.sha256) throw new Error(`Segment ${index} SHA-256 verification failed`);
      if (signal.aborted || this.closed) throw new DOMException('Obsolete segment', 'AbortError');
      this.notify({ index, state: 'ready', peer: response.headers.get('X-Peer-ID') ?? undefined,
        bytes: data.byteLength, attempts: Number(response.headers.get('X-Download-Attempts') ?? 1),
        failures: Number(response.headers.get('X-Download-Failures') ?? 0) });
      entry.resolve(data);
    } catch (error) {
      if (!signal.aborted && !this.closed) this.notify({ index, state: 'failed' });
      entry.reject(error);
    }
  }
}
