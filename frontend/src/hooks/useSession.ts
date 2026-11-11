// TODO(Tian): Add SSE replay with Last-Event-ID; snapshots currently recover state by polling once per second.
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
    const update = () => request<Session>(`/sessions/${id}`).then(value => { if (active) { setSession(value); setError(''); } }).catch(err => { if (active) setError(String(err)); });
    void update();
    const timer = window.setInterval(update, 1000);
    return () => { active = false; window.clearInterval(timer); };
  }, [id]);
  return { session, error };
}
