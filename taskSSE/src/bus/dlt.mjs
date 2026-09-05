/**
 * Dead Letter Topic (DLT) publisher for taskSSE Kafka consumer.
 *
 * When a message cannot be processed after exhausting retries, it is published
 * to a DLT topic for offline inspection.  The DLT topic is named {topic}-dlt.
 *
 * All operations are best-effort: errors are logged, never thrown, because DLT
 * publish must not block the consumer from committing the original offset.
 */

import { Kafka } from 'kafkajs';

/** @type {import('kafkajs').Kafka | null} */
let _client = null;

/** @type {import('kafkajs').Producer | null} */
let _producer = null;

/**
 * Initialize (or return) a shared Kafka producer for DLT publishing.
 * Call once at consumer startup.
 *
 * @param {string} bootstrapServers - comma-separated broker list
 * @returns {Promise<import('kafkajs').Producer>}
 */
export async function getDLTProducer(bootstrapServers) {
  if (_producer) return _producer;
  const brokers = bootstrapServers.split(',').map((s) => s.trim()).filter(Boolean);
  _client = new Kafka({ clientId: 'task-sse-dlt', brokers });
  _producer = _client.producer({ allowAutoTopicCreation: true });
  await _producer.connect();
  return _producer;
}

/**
 * Publish an unprocessable message to the dead-letter topic.
 *
 * @param {object} opts
 * @param {string} opts.bootstrapServers - Kafka broker list
 * @param {string} opts.originalTopic - the topic the message was consumed from
 * @param {object} opts.originalMessage - the raw kafkajs message value (parsed JSON)
 * @param {string} opts.error - error message
 * @param {'permanent'|'retry_exhausted'} opts.reason - why it was dead-lettered
 * @param {number} opts.retryCount - number of retries attempted
 * @param {object} [opts.log] - logger (defaults to console)
 */
export async function publishToDLT({ bootstrapServers, originalTopic, originalMessage, error, reason, retryCount, log }) {
  const logger = log || console;
  const dltTopic = `${originalTopic}-dlt`;

  const dltMessage = {
    original_event_type: 'SSE_MESSAGE',
    original_topic: originalTopic,
    original_data: originalMessage,
    original_key: '',
    error: String(error || ''),
    dead_lettered_at: new Date().toISOString(),
    failure_reason: reason,
    retry_count: retryCount,
  };

  try {
    const producer = await getDLTProducer(bootstrapServers);
    await producer.send({
      topic: dltTopic,
      messages: [{ value: JSON.stringify(dltMessage) }],
    });
    logger.log(`[taskSSE][dlt] dead-lettered → ${dltTopic} reason=${reason} retries=${retryCount}`);
  } catch (err) {
    logger.error(`[taskSSE][dlt] publish failed → ${dltTopic}: ${err.message}`);
  }
}

/**
 * Shut down the DLT producer (call on graceful shutdown).
 */
export async function closeDLTProducer() {
  if (_producer) {
    try { await _producer.disconnect(); } catch { /* ignore */ }
    _producer = null;
    _client = null;
  }
}
