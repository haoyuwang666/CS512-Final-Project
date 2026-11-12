// TODO(Tian): Validate dashboard terminology alongside future metric schema changes.
import type { Session } from '../types/api';
export default function MetricsPanel({ session }: { session: Session }) {
  const { summary, buffer, peers } = session;
  return <>
    <section><h2>Session · {summary.status}</h2><dl>
      <dt>已播放片段</dt><dd>{summary.segments_played}</dd>
      <dt>模拟启动延迟</dt><dd>{summary.startup_delay_ms.toFixed(1)} ms</dd>
      <dt>已消费媒体时长</dt><dd>{summary.media_playback_elapsed_ms.toFixed(1)} ms</dd>
      <dt>累计卡顿</dt><dd>{summary.stall_time_ms.toFixed(1)} ms</dd>
      <dt>卡顿比例</dt><dd>{(summary.rebuffering_ratio * 100).toFixed(2)}%</dd>
      <dt>原始时间线 Deadline Miss</dt><dd>{summary.deadline_misses}</dd>
      <dt>请求 / 失败</dt><dd>{summary.requests} / {summary.failed_requests}</dd>
      <dt>下载响应体 / 已验证字节</dt><dd>{summary.downloaded_bytes} / {summary.verified_bytes}</dd>
    </dl>{summary.error && <p role="alert">{summary.error}</p>}
      {session.persistence_error && <p role="alert">日志写入失败：{session.persistence_error}</p>}
    </section>
    <section><h2>逻辑 Buffer</h2><p>仅表示下载与模拟消费状态，不保留视频画面。</p>
      {buffer.length ? <ul>{buffer.map(segment => <li key={segment.index}>
        #{segment.index} · {segment.state} · {segment.duration_ms} ms · {segment.peer_id ?? '等待来源'}
      </li>)}</ul> : <p>当前没有缓冲分片。</p>}
      <h2>来源估计</h2><ul>{peers.map(peer => <li key={peer.peer_id}>
        {peer.peer_id} · {(peer.bytes_per_second / 1024).toFixed(1)} KiB/s · 并发 {peer.in_flight} · 连续失败 {peer.failures}
      </li>)}</ul>
    </section>
  </>;
}
