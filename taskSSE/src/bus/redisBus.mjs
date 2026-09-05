import { createClient } from 'redis';
import { normalizeInboundMessage } from '../messageNormalize.mjs';

/**
 * @param {{ redis: { host: string, port: number, channelPrefix: string }, onMessage: (taskId: string, statusData: object) => void }} opts
 */
export async function startRedisBus({ redis, onMessage, log = console.log }) {
  const prefix = redis.channelPrefix || 'sse:';
  const sub = createClient({ url: `redis://${redis.host}:${redis.port}` });
  sub.on('error', (err) => log('[taskSSE][redis] error', err?.message || err));
  await sub.connect();
  await sub.pSubscribe(`${prefix}*`, (message, channel) => {
    let parsed = null;
    try {
      parsed = JSON.parse(message);
    } catch {
      return;
    }
    const normalized =
      normalizeInboundMessage(parsed) ||
      (() => {
        const taskIdFromChannel = String(channel || '').slice(prefix.length);
        if (!taskIdFromChannel) return null;
        // billing channel may carry full envelope or bare status_data
        if (parsed && typeof parsed === 'object' && parsed.status_data) {
          return normalizeInboundMessage({
            task_id: taskIdFromChannel,
            user_id: parsed.user_id,
            status_data: parsed.status_data,
          });
        }
        return normalizeInboundMessage({ task_id: taskIdFromChannel, status_data: parsed });
      })();
    if (!normalized) return;
    onMessage(normalized.taskId, normalized.statusData);
  });
  log(`[taskSSE][redis] subscribed pattern ${prefix}* @ ${redis.host}:${redis.port}`);
  return {
    async close() {
      try {
        await sub.quit();
      } catch {
        /* ignore */
      }
    },
  };
}
