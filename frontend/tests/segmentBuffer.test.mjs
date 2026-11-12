// TODO(Tian): Extend client buffer regression tests alongside seek and transport changes.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { SegmentBuffer } from '../src/playback/segmentBuffer.ts';

const data = new TextEncoder().encode('verified mp4');
const sha256 = createHash('sha256').update(data).digest('hex');
const manifest = { segments: Array.from({ length: 8 }, (_, index) => ({ index, segment_id: `v--${index}`, size_bytes: data.length, sha256 })) };
const turn = () => new Promise(resolve => setImmediate(resolve));

// Tests replace only transport/object URLs; hashing and buffer scheduling are real.
test('prefetch respects concurrency, seeking cancels obsolete work, and appended data is released', async () => {
  const previous = globalThis.fetch;
  const pending = []; const notices = []; let active = 0; let peak = 0;
  globalThis.fetch = (_url, { signal }) => new Promise((resolve, reject) => {
    active++; peak = Math.max(peak, active);
    const cancel = () => { active--; reject(new DOMException('cancelled', 'AbortError')); };
    signal.addEventListener('abort', cancel, { once: true });
    pending.push(() => {
      if (signal.aborted) return;
      signal.removeEventListener('abort', cancel); active--;
      resolve(new Response(data, { headers: { 'X-Peer-ID': 'peer-2', 'X-Download-Attempts': '2', 'X-Download-Failures': '1' } }));
    });
  });
  const buffer = new SegmentBuffer('media-test', manifest, 3, 2, notice => notices.push(notice));
  try {
    buffer.setWindow(0);
    assert.equal(active, 2);
    const obsolete = buffer.get(0);
    buffer.setWindow(5);
    await assert.rejects(obsolete, { name: 'AbortError' });
    await turn();
    assert.equal(active, 2);
    // Cancelled entries are still in the test transport's list; complete live ones.
    for (const complete of pending.splice(0)) complete();
    await turn(); await turn();
    for (const complete of pending.splice(0)) complete();
    await turn(); await turn();
    assert.deepEqual(new Uint8Array(await buffer.get(5)), data);
    assert.ok(peak <= 2);
    assert.ok(notices.some(notice => notice.index === 5 && notice.state === 'ready' && notice.attempts === 2));
    buffer.release(5);
    await assert.rejects(buffer.get(5), /outside/);
    const before = notices.length;
    buffer.setWindow(5, new Set([5]));
    assert.ok(!notices.slice(before).some(notice => notice.index === 5));
    buffer.close();
  } finally {
    buffer.close(); globalThis.fetch = previous;
  }
});

test('corrupt bytes never reach the media append pipeline', async () => {
  const previous = globalThis.fetch;
  globalThis.fetch = async () => new Response(new TextEncoder().encode('corrupt mp4!'));
  const notices = []; const buffer = new SegmentBuffer('media-test', manifest, 1, 1, notice => notices.push(notice));
  try {
    buffer.setWindow(0);
    await assert.rejects(buffer.get(0), /校验失败/);
    assert.equal(notices.at(-1).state, 'failed');
    assert.ok(!notices.some(notice => notice.state === 'ready'));
  } finally { buffer.close(); globalThis.fetch = previous; }
});
