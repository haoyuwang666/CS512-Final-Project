// TODO(Tian): Validate dashboard terminology alongside future metric schema changes.
import type { Session } from '../types/api';
export default function MetricsPanel({ session }: { session: Session }) {
  const { summary, buffer, peers } = session;
  return <>
    <section><h2>Session · {summary.status}</h2><dl>
      <dt>Segments played</dt><dd>{summary.segments_played}</dd>
      <dt>Simulated startup delay</dt><dd>{summary.startup_delay_ms.toFixed(1)} ms</dd>
      <dt>Media time consumed</dt><dd>{summary.media_playback_elapsed_ms.toFixed(1)} ms</dd>
      <dt>Total stall time</dt><dd>{summary.stall_time_ms.toFixed(1)} ms</dd>
      <dt>Rebuffering ratio</dt><dd>{(summary.rebuffering_ratio * 100).toFixed(2)}%</dd>
      <dt>Original-timeline deadline misses</dt><dd>{summary.deadline_misses}</dd>
      <dt>Requests / failures</dt><dd>{summary.requests} / {summary.failed_requests}</dd>
      <dt>Downloaded / verified bytes</dt><dd>{summary.downloaded_bytes} / {summary.verified_bytes}</dd>
    </dl>{summary.error && <p role="alert">{summary.error}</p>}
      {session.persistence_error && <p role="alert">Event persistence failed: {session.persistence_error}</p>}
    </section>
    <section><h2>Logical buffer</h2><p>Shows download and simulated consumption state, not decoded video.</p>
      {buffer.length ? <ul>{buffer.map(segment => <li key={segment.index}>
        #{segment.index} · {segment.state} · {segment.duration_ms} ms · {segment.peer_id ?? 'Awaiting source'}
      </li>)}</ul> : <p>No buffered segments.</p>}
      <h2>Source estimates</h2><ul>{peers.map(peer => <li key={peer.peer_id}>
        {peer.peer_id} · {(peer.bytes_per_second / 1024).toFixed(1)} KiB/s · In flight {peer.in_flight} · Consecutive failures {peer.failures}
      </li>)}</ul>
    </section>
  </>;
}
