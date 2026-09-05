/**
 * Normalize Redis / Kafka / HTTP publish payloads into { taskId, statusData }.
 * Billing recharge uses hub key `billing:user:{userId}` (Redis channel sse:billing:user:{userId}).
 * Work panel uses hub key `workspace:{workspaceId}` (Redis channel sse:workspace:{workspaceId}).
 */
export function normalizeInboundMessage(raw) {
  if (!raw || typeof raw !== 'object') return null;

  // Kafka envelope: { event_type, data: { task_id, status_data } }
  if (raw.event_type === 'SSE_MESSAGE' && raw.data && typeof raw.data === 'object') {
    return normalizeInboundMessage(raw.data);
  }
  if (raw.data && typeof raw.data === 'object' && (raw.data.task_id || raw.data.status_data)) {
    return normalizeInboundMessage(raw.data);
  }

  let taskId = String(raw.task_id || raw.taskId || '').trim();
  let statusData = raw.status_data ?? raw.statusData ?? null;
  if (!statusData && raw.message && typeof raw.message === 'object') {
    statusData = raw.message;
  }

  // 充值完成：{ user_id, status_data }（无 task_id 时由 user_id 推导 hub key）
  if (!taskId && statusData && typeof statusData === 'object') {
    const userId = String(raw.user_id || statusData.user_id || '').trim();
    if (userId) {
      taskId = `billing:user:${userId}`;
    }
  }

  // Work panel：无 task_id 时由 workspace_id 推导 hub key
  if (!taskId && statusData && typeof statusData === 'object') {
    const ws = String(raw.workspace_id || statusData.workspace_id || '').trim();
    if (ws) {
      taskId = workspaceHubKey(ws);
    }
  }

  if (!taskId || !statusData || typeof statusData !== 'object') return null;
  if (!statusData.event_name) {
    let defaultName = 'server_status_update';
    if (String(taskId).startsWith('billing:user:')) {
      defaultName = 'recharge_completed';
    } else if (String(taskId).startsWith('workspace:')) {
      defaultName = 'task_status_changed';
    }
    statusData = { ...statusData, event_name: defaultName };
  }
  return { taskId, statusData };
}

/** Hub key for a billing user SSE subscription. */
export function billingUserHubKey(userId) {
  const uid = String(userId || '').trim();
  if (!uid) return '';
  return `billing:user:${uid}`;
}

/** Hub key for a work-panel workspace SSE subscription. */
export function workspaceHubKey(workspaceId) {
  const ws = String(workspaceId || '').trim();
  if (!ws) return '';
  return `workspace:${ws}`;
}

export function formatSseData(statusData) {
  return `data: ${JSON.stringify(statusData)}\n\n`;
}
