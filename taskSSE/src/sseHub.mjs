import { formatSseData } from './messageNormalize.mjs';

export class SseHub {
  /** @type {Map<string, Set<import('node:http').ServerResponse>>} */
  #byTask = new Map();

  subscribe(taskId, res) {
    const key = String(taskId || '').trim();
    if (!key) return () => {};
    let set = this.#byTask.get(key);
    if (!set) {
      set = new Set();
      this.#byTask.set(key, set);
    }
    set.add(res);
    return () => {
      set.delete(res);
      if (set.size === 0) this.#byTask.delete(key);
    };
  }

  connectionCount(taskId) {
    const set = this.#byTask.get(String(taskId || '').trim());
    return set ? set.size : 0;
  }

  totalConnections() {
    let n = 0;
    for (const set of this.#byTask.values()) n += set.size;
    return n;
  }

  publish(taskId, statusData) {
    const key = String(taskId || '').trim();
    const set = this.#byTask.get(key);
    if (!set || set.size === 0) return 0;
    const frame = formatSseData(statusData);
    let sent = 0;
    for (const res of [...set]) {
      try {
        if (res.writableEnded || res.destroyed) {
          set.delete(res);
          continue;
        }
        res.write(frame);
        sent += 1;
      } catch {
        set.delete(res);
      }
    }
    if (set.size === 0) this.#byTask.delete(key);
    return sent;
  }
}
