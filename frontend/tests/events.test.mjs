// TODO(Yuquan): Extend event normalization tests as backend event types evolve.
import test from 'node:test';
import assert from 'node:assert/strict';
import { faultEntry, playbackEntry, filterEntries, newestFirst, eventTime } from '../src/logging/events.ts';
test('fault actions keep server timestamps and intentional stops are warnings', () => {
  const entry = faultEntry({ sequence: 3, node_id: 'peer-2', action: 'stop', at: '2026-11-13T10:00:00Z' });
  assert.equal(entry.level, 'warning'); assert.equal(entry.time, '2026-11-13T10:00:00Z'); assert.equal(entry.node, 'peer-2');
});
test('session failures remain errors and retryable failures are warnings', () => {
  assert.equal(playbackEntry({ sequence: 1, type: 'session_failed', time: '' }).level, 'error');
  assert.equal(playbackEntry({ sequence: 2, type: 'request_failed', time: '' }).level, 'warning');
});
test('filters combine severity, source and node or segment search without mutating entries', () => {
  const entries = [faultEntry({ sequence: 1, node_id: 'peer-2', action: 'stop', at: '2026-11-13T10:00:00Z' }), playbackEntry({ sequence: 2, type: 'session_completed', segment_id: 'movie--0003', time: '2026-11-13T11:00:00Z' })];
  assert.equal(filterEntries(entries, 'warning', 'Fault control', 'PEER-2').length, 1);
  assert.equal(filterEntries(entries, 'error', 'all', '').length, 0);
  assert.equal(filterEntries(entries, 'all', 'all', 'movie--0003').length, 1);
  assert.equal(newestFirst(entries)[0].id, 'session-2'); assert.equal(entries[0].id, 'fault-1');
  assert.equal(eventTime(undefined), '—'); assert.equal(eventTime('invalid'), '—');
});
