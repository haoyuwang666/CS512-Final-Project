// TODO(Tian): Keep single-source, seek and bounded-buffer invariants covered for MSE changes.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { MSEPlayer, segmentAt } from '../src/playback/msePlayer.ts';

const tick = () => new Promise(resolve => setImmediate(resolve));
class FakeBuffer extends EventTarget {
  updating = false; mode = ''; log = []; source; offset = 0;
  constructor(source) { super(); this.source = source; }
  set timestampOffset(value) { this.offset = value; if (this.source.readyState === 'ended') this.source.readyState = 'open'; }
  get timestampOffset() { return this.offset; }
  appendBuffer(data) { this.modify(['append', new Uint8Array(data)[0]]); }
  remove(start, end) { this.modify(['remove', start, end]); }
  modify(operation) {
    assert.equal(this.updating, false, 'SourceBuffer mutations must be serialized');
    this.updating = true; this.log.push(operation);
    queueMicrotask(() => { this.updating = false; this.dispatchEvent(new Event('updateend')); });
  }
  abort() { this.updating = false; this.dispatchEvent(new Event('abort')); }
}
class FakeSource extends EventTarget {
  static isTypeSupported() { return true; }
  readyState = 'closed'; duration = 0; buffer;
  addSourceBuffer() { this.buffer = new FakeBuffer(this); return this.buffer; }
  endOfStream() { assert.equal(this.buffer.updating, false); this.readyState = 'ended'; }
}
class FakeVideo extends EventTarget {
  sources = []; time = 0; loads = 0;
  set src(value) { this.sources.push(value); }
  set currentTime(time) { this.time = time; this.dispatchEvent(new Event('seeking')); }
  get currentTime() { return this.time; }
  load() { this.loads++; }
  pause() {}
  removeAttribute() {}
}
const session = { session_id: 'test', window: 3, concurrency: 2, startup_buffer_ms: 4000,
  manifest: { representation: 'continuous-fmp4-h264-aac-v1', media: { mime_type: 'video/mp4; codecs="avc1.64000c, mp4a.40.2"', duration_ms: 12000 },
    segments: Array.from({ length: 6 }, (_, index) => ({ index, duration_ms: 2000 })) } };

function harness() {
  const old = { source: globalThis.MediaSource, create: URL.createObjectURL, revoke: URL.revokeObjectURL };
  let source; const revoked = []; const errors = []; const requests = []; const released = [];
  globalThis.MediaSource = FakeSource;
  URL.createObjectURL = value => { source = value; return 'blob:one-source'; };
  URL.revokeObjectURL = value => revoked.push(value);
  const video = new FakeVideo();
  const buffer = { setWindow() {}, async get(index) { requests.push(index); return Uint8Array.of(index).buffer; }, release(index) { released.push(index); }, close() {} };
  const player = new MSEPlayer(video, buffer, session, () => {}, error => errors.push(error));
  source.readyState = 'open'; source.dispatchEvent(new Event('sourceopen'));
  const cleanup = () => { player.close(); globalThis.MediaSource = old.source; URL.createObjectURL = old.create; URL.revokeObjectURL = old.revoke; };
  return { player, video, source, errors, requests, released, revoked, cleanup };
}
test('MSE uses one source, serial appends, and no load/pause at segment boundaries', async () => {
  const h = harness();
  try {
    await h.player.ready; await tick();
    assert.deepEqual(h.requests, [0, 1, 2]);
    assert.equal(h.source.buffer.mode, 'segments');
    h.video.currentTime = 2.5; await tick();
    assert.deepEqual(h.requests, [0, 1, 2, 3]);
    assert.equal(h.video.loads, 1);
    assert.deepEqual(h.video.sources, ['blob:one-source']);
    assert.equal(h.errors.length, 0);
    assert.deepEqual(h.released, h.requests);
  } finally { h.cleanup(); }
  assert.deepEqual(h.revoked, ['blob:one-source']);
});

test('unbuffered seeks and replay after end reopen the same source and evict old media', async () => {
  const h = harness();
  try {
    await h.player.ready; await tick();
    h.player.seek(10.1); await tick();
    assert.equal(h.source.readyState, 'ended');
    assert.ok(h.source.buffer.log.some(item => item[0] === 'remove'));
    h.player.seek(0); await tick();
    assert.equal(h.source.readyState, 'open');
    assert.equal(h.video.loads, 1);
    assert.equal(h.video.sources.length, 1);
    assert.equal(h.errors.length, 0);
    assert.equal(h.requests.filter(index => index === 0).length, 2);
  } finally { h.cleanup(); }
});

test('timeline lookup respects exact segment boundaries', () => {
  const offsets = [0, 2.021, 4.021, 6.021];
  assert.equal(segmentAt(offsets, 0), 0);
  assert.equal(segmentAt(offsets, 2.0209), 0);
  assert.equal(segmentAt(offsets, 2.021), 1);
  assert.equal(segmentAt(offsets, 100), 3);
});
