// TODO(Yuquan): Add server-side pagination if event retention grows beyond bounded snapshots.
import { useMemo, useState } from 'react';
import { eventTime, filterEntries, newestFirst } from '../logging/events';
import type { LogEntry, LogLevel } from '../logging/events';

interface Props { entries: LogEntry[]; title: string; description?: string; fileName?: string }
const levels: LogLevel[] = ['error', 'warning', 'success', 'info'];
export default function EventLog({ entries, title, description, fileName = 'events' }: Props) {
  const [level, setLevel] = useState('all'); const [category, setCategory] = useState('all'); const [query, setQuery] = useState('');
  const [frozen, setFrozen] = useState<LogEntry[] | null>(null); const [limit, setLimit] = useState(50); const [feedback, setFeedback] = useState('');
  const source = frozen ?? entries;
  const filtered = useMemo(() => newestFirst(filterEntries(source, level, category, query)), [source, level, category, query]);
  const categories = [...new Set([...source.map(entry => entry.category), ...(category === 'all' ? [] : [category])])].sort();
  const pending = frozen ? entries.filter(entry => !frozen.some(old => old.id === entry.id)).length : 0;
  const serialized = () => filtered.map(entry => JSON.stringify(entry)).join('\n') + (filtered.length ? '\n' : '');
  function download() {
    const url = URL.createObjectURL(new Blob([serialized()], { type: 'application/x-ndjson' }));
    const link = document.createElement('a'); link.href = url; link.download = `${fileName}-${Date.now()}.jsonl`; link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  async function copy() {
    try { await navigator.clipboard.writeText(serialized()); setFeedback(`Copied ${filtered.length} matching events.`); }
    catch { setFeedback('Clipboard access is unavailable. Use Download JSONL instead.'); }
  }
  return <div className="event-log" role="region" aria-label={title}>
    <div className="section-title"><div><h3>{title}</h3>{description && <p className="muted small">{description}</p>}</div>
      <span className={`log-live ${frozen ? 'paused' : ''}`}><i/>{frozen ? `Paused${pending ? ` · ${pending} new` : ''}` : 'Live'}</span></div>
    <div className="log-counts" aria-label="Event severity filters">
      <button className="secondary" aria-pressed={level === 'all'} onClick={() => setLevel('all')}>All <b>{source.length}</b></button>
      {levels.map(value => <button key={value} className={`log-count ${value}`} aria-pressed={level === value} onClick={() => setLevel(value)}>{value} <b>{source.filter(entry => entry.level === value).length}</b></button>)}
    </div>
    <div className="log-toolbar">
      <label className="log-search">Search events<input type="search" value={query} placeholder="Message, node, or segment…" onChange={event => { setQuery(event.target.value); setLimit(50); }}/></label>
      <label>Source<select value={category} onChange={event => { setCategory(event.target.value); setLimit(50); }}><option value="all">All sources</option>{categories.map(value => <option key={value}>{value}</option>)}</select></label>
      <div className="log-tools"><button className="secondary" onClick={() => setFrozen(frozen ? null : entries.slice())}>{frozen ? 'Resume live' : 'Pause updates'}</button>
        <button className="secondary" disabled={!filtered.length} onClick={() => void copy()}>Copy</button><button className="secondary" disabled={!filtered.length} onClick={download}>Download JSONL</button></div>
    </div>
    {feedback && <p className="muted small" role="status">{feedback}</p>}
    <div className="log-table-wrap" tabIndex={0} aria-label={`${title} entries`}>
      <table className="log-table"><thead><tr><th>Time · local</th><th>Level</th><th>Source / node</th><th>Event</th></tr></thead><tbody>
        {filtered.slice(0, limit).map(entry => <tr key={entry.id} className={`log-row ${entry.level}`}>
          <td className="log-time"><time dateTime={entry.time} title={entry.time}>{eventTime(entry.time)}</time></td>
          <td><span className={`log-level ${entry.level}`}>{entry.level}</span></td>
          <td><span>{entry.category}</span>{entry.node && <code className="log-node">{entry.node}</code>}</td>
          <td><div className="log-message">{entry.message}</div>{entry.segment && <code className="log-segment" title={entry.segment}>{entry.segment}</code>}
            {entry.details !== undefined && <details><summary>Details</summary><pre>{JSON.stringify(entry.details, null, 2)}</pre></details>}</td>
        </tr>)}
      </tbody></table>
      {!filtered.length && <div className="log-empty"><strong>{source.length ? 'No matching events' : 'No events yet'}</strong><p>{source.length ? 'Change the filters or search terms.' : 'Activity will appear here as it happens.'}</p></div>}
    </div>
    <div className="log-footer"><span>{Math.min(limit, filtered.length)} of {filtered.length} matching · newest first{frozen ? ' · display paused; processes continue running' : ''}</span>
      {filtered.length > limit && <button className="secondary" onClick={() => setLimit(value => value + 50)}>Show 50 more</button>}</div>
  </div>;
}
