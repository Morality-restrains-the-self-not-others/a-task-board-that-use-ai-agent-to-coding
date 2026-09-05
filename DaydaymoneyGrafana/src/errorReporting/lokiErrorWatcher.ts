import { dateTime } from '@grafana/data';
import { lastValueFrom } from 'rxjs';
import { getBackendSrv } from '@grafana/runtime';

import type { AppPluginSettings, CaptureRule } from '../types/settings';
import { lokiQueryForRule } from '../types/settings';
import { dispatchCapturedLog } from './captureEngine';
import { matchesRuleLine } from './ruleMatcher';

type PluginSettingsResponse = {
  jsonData?: AppPluginSettings;
};

type LokiStreamValue = [string, string];

type LokiQueryResponse = {
  data?: {
    result?: Array<{
      stream?: Record<string, string>;
      values?: LokiStreamValue[];
    }>;
  };
};

const DEFAULT_LOKI_UID = 'loki';
const DEFAULT_POLL_SEC = 30;
const seenFingerprints = new Map<string, number>();
const SEEN_TTL_MS = 5 * 60_000;

function fingerprint(service: string, message: string, ts: string): string {
  return `${service}|${message}|${ts}`;
}

function pruneSeen(now: number): void {
  for (const [key, ts] of seenFingerprints.entries()) {
    if (now - ts > SEEN_TTL_MS) {
      seenFingerprints.delete(key);
    }
  }
}

async function getPluginJsonData(pluginId: string): Promise<AppPluginSettings | undefined> {
  try {
    const res = await lastValueFrom(
      getBackendSrv().fetch<PluginSettingsResponse>({
        url: `/api/plugins/${pluginId}/settings`,
        method: 'GET',
      })
    );
    return res.data.jsonData;
  } catch {
    return undefined;
  }
}

function enabledRules(cfg: AppPluginSettings): CaptureRule[] {
  return (cfg.captureRules ?? []).filter((rule) => rule.enabled);
}

async function queryLoki(uid: string, query: string): Promise<LokiQueryResponse | undefined> {
  const end = dateTime();
  const start = end.subtract(2, 'minute');

  try {
    const fetched = await lastValueFrom(
      getBackendSrv().fetch<LokiQueryResponse>({
        url: `/api/datasources/proxy/uid/${uid}/loki/api/v1/query_range`,
        method: 'GET',
        params: {
          query,
          limit: 40,
          direction: 'forward',
          start: String(start.valueOf() * 1_000_000),
          end: String(end.valueOf() * 1_000_000),
        },
        showErrorAlert: false,
        hideFromInspector: true,
      })
    );
    return fetched.data;
  } catch {
    return undefined;
  }
}

async function processStreams(
  cfg: AppPluginSettings,
  streams: NonNullable<LokiQueryResponse['data']>['result'],
  rules: CaptureRule[]
): Promise<void> {
  const now = Date.now();
  pruneSeen(now);

  for (const stream of streams ?? []) {
    const labels = stream.stream ?? {};
    const service = labels.service || labels.job || 'unknown';
    for (const [ts, line] of stream.values ?? []) {
      const message = line.trim();
      if (!message) {
        continue;
      }
      const fp = fingerprint(service, message, ts);
      if (seenFingerprints.has(fp)) {
        continue;
      }

      const applicable = rules.filter((rule) => matchesRuleLine(rule, line));
      if (applicable.length === 0 && (cfg.captureRules?.length ?? 0) > 0) {
        continue;
      }
      seenFingerprints.set(fp, now);

      const traceMatch =
        message.match(/"trace_id"\s*:\s*"([^"]+)"/) || message.match(/trace_id=([A-Za-z0-9._:-]+)/);

      await dispatchCapturedLog(cfg, {
        message: message.slice(0, 2000),
        service,
        traceId: traceMatch?.[1],
        timestamp: new Date(Number(ts.slice(0, 13)) || Date.now()).toISOString(),
        rawLine: line,
      });
    }
  }
}

async function pollLokiOnce(pluginId: string): Promise<void> {
  const cfg = await getPluginJsonData(pluginId);
  if (!cfg?.lokiPollEnabled) {
    return;
  }

  const hasRules = (cfg.captureRules?.length ?? 0) > 0;
  const hasTaskApi = Boolean(cfg.task2appApiBaseUrl?.trim() && cfg.task2appSessionToken?.trim());
  if (!hasRules || !hasTaskApi) {
    return;
  }

  const uid = cfg.lokiDatasourceUid?.trim() || DEFAULT_LOKI_UID;
  const rules = enabledRules(cfg);
  const queries = rules.length
    ? [...new Set(rules.map((rule) => lokiQueryForRule(rule)))]
    : ['{job="runall"} | json | level=~"(?i)error|fatal"'];

  for (const query of queries) {
    const response = await queryLoki(uid, query);
    await processStreams(cfg, response?.data?.result, rules);
  }
}

export function installLokiErrorWatcher(pluginId: string): void {
  if (typeof window === 'undefined') {
    return;
  }

  const tick = () => {
    void pollLokiOnce(pluginId);
  };

  void (async () => {
    const cfg = await getPluginJsonData(pluginId);
    const pollSec = Math.max(10, cfg?.lokiPollIntervalSec ?? DEFAULT_POLL_SEC);
    tick();
    window.setInterval(tick, pollSec * 1000);
  })();
}
