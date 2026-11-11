// TODO(Tian): Add buffer/Peer views after the backend exposes snapshots; never manufacture missing live values.
import type { Summary } from '../types/api';
export default function MetricsPanel({ summary }: { summary: Summary }) {
  return <section><h2>Session · {summary.status}</h2><dl>
    <dt>已播放片段</dt><dd>{summary.segments_played}</dd><dt>模拟启动延迟</dt><dd>{summary.startup_delay_ms.toFixed(1)} ms</dd>
    <dt>累计卡顿</dt><dd>{summary.stall_time_ms.toFixed(1)} ms</dd><dt>卡顿比例</dt><dd>{(summary.rebuffering_ratio * 100).toFixed(2)}%</dd>
  </dl>{summary.status === 'running' && <p>最终汇总将在 Session 结束后更新。</p>}{summary.error && <p role="alert">{summary.error}</p>}</section>;
}
