// TODO(Tian): Keep snapshot recovery compatible with future event schema versions.
import { useEffect, useState } from 'react';
import { request } from '../api/client';
import type { Session } from '../types/api';
export function useSession(id: string) {
  const [session, setSession] = useState<Session>();
  const [error, setError] = useState('');
  useEffect(() => {
    setSession(undefined); setError('');
    if (!id) return;
    let active = true;
    let fetching = false;
    let refreshTimer: number | undefined;
    const stream = new EventSource(`/api/v1/sessions/${id}/events`);
    // EventSource carries Last-Event-ID on reconnect. REST remains authoritative
    // for buffer/metrics and also recovers from expired or unavailable event history.
    const update = async () => {
      if (fetching || !active) return;
      fetching = true;
      try {
        const value = await request<Session>(`/sessions/${id}`);
        if (!active) return;
        setSession(value); setError('');
        if (value.summary.status !== 'running') stream.close();
      } catch (err) { if (active) setError(String(err)); }
      finally { fetching = false; }
    };
    const refreshSoon = () => {
      if (refreshTimer !== undefined) return;
      refreshTimer = window.setTimeout(() => { refreshTimer = undefined; void update(); }, 100);
    };
    stream.onmessage = refreshSoon;
    stream.onerror = refreshSoon;
    stream.addEventListener('reset', () => { stream.close(); void update(); });
    void update();
    const timer = window.setInterval(() => void update(), 1000);
    return () => {
      active = false; stream.close(); window.clearInterval(timer);
      if (refreshTimer !== undefined) window.clearTimeout(refreshTimer);
    };
  }, [id]);
  return { session, error };
}
