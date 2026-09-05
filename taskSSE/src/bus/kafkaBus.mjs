import { Kafka } from 'kafkajs';
import { normalizeInboundMessage } from '../messageNormalize.mjs';
import { publishToDLT, closeDLTProducer } from './dlt.mjs';

/**
 * @param {{ kafka: { bootstrapServers: string, topic: string, groupId: string }, onMessage: (taskId: string, statusData: object) => void }} opts
 */
export async function startKafkaBus({ kafka, onMessage, log = console.log }) {
  const bootstrapServers = kafka.bootstrapServers;
  const brokers = bootstrapServers.split(',').map((s) => s.trim()).filter(Boolean);
  const client = new Kafka({ clientId: 'task-sse', brokers });
  const consumer = client.consumer({ groupId: kafka.groupId });
  await consumer.connect();
  await consumer.subscribe({ topic: kafka.topic, fromBeginning: false });

  const maxRetries = parseInt(process.env.DLT_MAX_RETRIES || '10', 10);
  /** @type {Map<string, {count: number, lastSeen: number}>} */
  const retryState = new Map();

  await consumer.run({
    eachMessage: async ({ message }) => {
      if (!message.value) return;
      let parsed;
      try {
        parsed = JSON.parse(message.value.toString('utf8'));
      } catch {
        return;
      }
      const normalized = normalizeInboundMessage(parsed);
      if (!normalized) return;

      try {
        onMessage(normalized.taskId, normalized.statusData);
        // Success — clear retry state
        if (normalized.taskId) retryState.delete(normalized.taskId);
      } catch (err) {
        // Track retry attempts per taskId
        const key = normalized.taskId || 'unknown';
        const state = retryState.get(key) || { count: 0, lastSeen: Date.now() };
        state.count++;
        state.lastSeen = Date.now();
        retryState.set(key, state);

        if (state.count > maxRetries) {
          // Retry exhausted → DLT
          log(`[taskSSE][kafka] retry exhausted for ${key} after ${state.count} attempts`);
          await publishToDLT({
            bootstrapServers,
            originalTopic: kafka.topic,
            originalMessage: parsed,
            error: err.message,
            reason: 'retry_exhausted',
            retryCount: state.count,
            log,
          });
          retryState.delete(key);
          // kafkajs auto-commits after eachMessage returns without throwing
          return;
        }

        // Within retry budget — throw to trigger kafkajs retry (default: 5 retries,
        // then consumer restart).  This is a secondary retry mechanism; the DLT
        // path above is the primary circuit-breaker.
        log(`[taskSSE][kafka] handler error for ${key} (attempt ${state.count}/${maxRetries}): ${err.message}`);
        throw err;
      }
    },
  });

  // Periodic cleanup of stale retry state (clear entries older than 5 min)
  const cleanupTimer = setInterval(() => {
    const cutoff = Date.now() - 5 * 60 * 1000;
    for (const [key, state] of retryState) {
      if (state.lastSeen < cutoff) retryState.delete(key);
    }
  }, 60000);

  log(`[taskSSE][kafka] consuming topic ${kafka.topic} group ${kafka.groupId} dltMaxRetries=${maxRetries}`);
  return {
    async close() {
      clearInterval(cleanupTimer);
      try {
        await consumer.disconnect();
      } catch {
        /* ignore */
      }
      await closeDLTProducer();
    },
  };
}
