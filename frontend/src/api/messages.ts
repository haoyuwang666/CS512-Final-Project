// TODO(Yuquan): Prefer stable server error codes when all management routes expose them.
export function serverMessage(value: unknown, fallback = 'The server could not complete the request.'): string {
  const text = value instanceof Error ? value.message : String(value ?? '');
  if (!/[\u3400-\u9fff]/.test(text)) return text || fallback;
  const removed = text.match(/已删除 (.*?) \/ (.*?) 份副本/);
  if (removed) return `Removed ${removed[1]} / ${removed[2]} replicas. Offline peers will continue after recovery.`;
  if (/Leader.*变化|已不是 Leader/.test(text)) return 'The leader changed. Refresh status and try again.';
  if (/没有可用 Leader|暂无可读 Leader/.test(text)) return 'No readable leader. Recover a controller quorum and try again.';
  if (/请先启动集群/.test(text)) return 'Start the cluster first.';
  if (/仅允许本机|跨来源/.test(text)) return 'This management request must come from the local console.';
  if (/不属于当前控制台/.test(text)) return 'This node is not managed by the current console.';
  if (/处理|等待当前任务/.test(text)) return 'A processing job is active. Wait for it to finish and try again.';
  if (/最大 512|上传失败/.test(text)) return 'Upload failed. Check the file and the 512 MB limit.';
  if (/无法读取|不可用/.test(text)) return 'The requested service is unavailable. Check node status.';
  return fallback;
}
export function responseError(value: unknown, status: number): Error {
  return new Error(serverMessage(value, `Request failed (HTTP ${status}). Open event details for the original response.`), { cause: value });
}
