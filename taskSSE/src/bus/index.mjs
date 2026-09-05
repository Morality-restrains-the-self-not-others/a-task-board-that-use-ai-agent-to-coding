import { startRedisBus } from './redisBus.mjs';
import { startKafkaBus } from './kafkaBus.mjs';

export async function startMessageBus(cfg, onMessage, log = console.log) {
  if (cfg.transport === 'kafka') {
    return startKafkaBus({ kafka: cfg.kafka, onMessage, log });
  }
  return startRedisBus({ redis: cfg.redis, onMessage, log });
}
