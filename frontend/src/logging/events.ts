// TODO(Yuquan): Extend event labels alongside backend schema additions.
import type { PlaybackEvent } from '../types/api.ts';
export type LogLevel = 'info' | 'success' | 'warning' | 'error';
export interface LogEntry {
  id: string; time?: string; level: LogLevel; category: string; message: string;
  node?: string; segment?: string; details?: unknown;
}
export interface FaultRecord { sequence: number; node_id: string; action: string; at: string }
const labels: Record<string, string> = {
  session_started: 'Session started', session_completed: 'Session completed', session_failed: 'Session failed', session_stopped: 'Session stopped',
  download_started: 'Download started', download_completed: 'Download verified', download_failed: 'Download failed',
  request_started: 'Request started', request_completed: 'Request completed', request_failed: 'Request failed',
  segment_ready: 'Segment verified', segment_played: 'Segment consumed', stall_started: 'Buffering started', stall_ended: 'Buffering ended',
  deadline_miss: 'Deadline missed', metadata_cache_fallback: 'Using cached metadata',
};
export function playbackEntry(event: PlaybackEvent): LogEntry {
  const level: LogLevel = /session_failed|error/.test(event.type) ? 'error'
    : /fail|retry|stall|miss|fallback/.test(event.type) ? 'warning'
    : /completed|verified|ready/.test(event.type) ? 'success' : 'info';
  const title = labels[event.type] ?? event.type.replaceAll('_', ' ').replace(/^./, letter => letter.toUpperCase());
  return { id: `session-${event.sequence}`, time: event.time, level, category: 'Session', node: event.peer_id,
    segment: event.segment_id, message: event.message ? `${title}: ${event.message}` : title, details: event };
}
export function faultEntry(event: FaultRecord): LogEntry {
  const action = { stop: 'Process stopped', start: 'Process recovered', restart: 'Process restarted' }[event.action] ?? event.action;
  return { id: `fault-${event.sequence}`, time: event.at, level: event.action === 'stop' ? 'warning' : 'success',
    category: 'Fault control', node: event.node_id, message: action, details: event };
}
export function filterEntries(entries: LogEntry[], level: string, category: string, query: string): LogEntry[] {
  const needle = query.trim().toLowerCase();
  return entries.filter(entry => (level === 'all' || entry.level === level) && (category === 'all' || entry.category === category)
    && (!needle || [entry.message, entry.node, entry.segment, entry.category].filter(Boolean).join(' ').toLowerCase().includes(needle)));
}
export function newestFirst(entries: LogEntry[]): LogEntry[] {
  return entries.slice().reverse().sort((a, b) => {
    const aTime = Date.parse(a.time ?? ''); const bTime = Date.parse(b.time ?? '');
    // Unknown timestamps stay at the bottom; never invent event times.
    const first = Number.isFinite(aTime) ? aTime : -Infinity;
    const second = Number.isFinite(bTime) ? bTime : -Infinity;
    return first === second ? 0 : second - first;
  });
}
export function eventTime(value?: string): string {
  if (!value || !Number.isFinite(Date.parse(value))) return '—';
  return new Intl.DateTimeFormat('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3, hour12: false }).format(new Date(value));
}
