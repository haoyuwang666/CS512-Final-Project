// TODO(Tian): Add versioned API capabilities; display server errors without generating local experimental metrics.
import { responseError } from './messages';
export async function request<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { method, headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
  const data = await response.json();
  if (!response.ok) throw responseError(data.error?.message ?? `HTTP ${response.status}`, response.status);
  return data as T;
}
