import { lastValueFrom } from 'rxjs';
import { getBackendSrv } from '@grafana/runtime';

import type { AppPluginSettings } from '../types/settings';
import { dispatchCapturedLog } from './captureEngine';

export type ErrorReportPayload = {
  pluginId: string;
  kind: 'window.error' | 'unhandledrejection' | 'resource.error' | 'loki.error';
  message: string;
  stack?: string;
  source?: string;
  lineno?: number;
  colno?: number;
  href: string;
  userAgent: string;
  timestamp: string;
  service?: string;
  traceId?: string;
};

type PluginSettingsResponse = {
  jsonData?: AppPluginSettings;
};

let settingsCache: { data?: AppPluginSettings; fetchedAt: number } = { fetchedAt: 0 };
const SETTINGS_TTL_MS = 60_000;

async function getPluginJsonData(pluginId: string): Promise<AppPluginSettings | undefined> {
  const now = Date.now();
  if (settingsCache.data !== undefined && now - settingsCache.fetchedAt < SETTINGS_TTL_MS) {
    return settingsCache.data;
  }
  try {
    const res = await lastValueFrom(
      getBackendSrv().fetch<PluginSettingsResponse>({
        url: `/api/plugins/${pluginId}/settings`,
        method: 'GET',
      })
    );
    const jsonData = res.data.jsonData;
    settingsCache = { data: jsonData, fetchedAt: now };
    return jsonData;
  } catch {
    return undefined;
  }
}

const lastSent = new Map<string, number>();
const DEDUP_MS = 4000;

function shouldSkipDuplicate(key: string): boolean {
  const t = Date.now();
  const prev = lastSent.get(key) ?? 0;
  if (t - prev < DEDUP_MS) {
    return true;
  }
  lastSent.set(key, t);
  return false;
}

async function postReport(pluginId: string, payload: ErrorReportPayload): Promise<void> {
  const cfg = await getPluginJsonData(pluginId);
  if (!cfg) {
    return;
  }

  const dedupeKey = `${payload.kind}:${payload.message}:${payload.source ?? ''}`;
  if (shouldSkipDuplicate(dedupeKey)) {
    return;
  }

  try {
    await dispatchCapturedLog(cfg, {
      message: payload.message,
      service: payload.service || 'browser',
      timestamp: payload.timestamp,
      traceId: payload.traceId,
      rawLine: [payload.message, payload.stack, payload.source].filter(Boolean).join('\n'),
    });
  } catch {
    // Intentionally swallow: avoid recursive error reporting.
  }
}

export function installGlobalErrorReporter(pluginId: string): void {
  if (typeof window === 'undefined') {
    return;
  }

  window.addEventListener(
    'error',
    (event) => {
      const target = event.target;
      if (target && target !== window && target instanceof HTMLElement) {
        const src =
          (target as HTMLImageElement).src ||
          (target as HTMLScriptElement).src ||
          (target as HTMLLinkElement).href ||
          '';
        void postReport(pluginId, {
          pluginId,
          kind: 'resource.error',
          message: `Resource failed to load: ${target.tagName}${src ? ` ${src}` : ''}`,
          href: window.location.href,
          userAgent: navigator.userAgent,
          timestamp: new Date().toISOString(),
        });
        return;
      }

      void postReport(pluginId, {
        pluginId,
        kind: 'window.error',
        message: event.message || 'Unknown error',
        stack: event.error instanceof Error ? event.error.stack : undefined,
        source: event.filename,
        lineno: event.lineno,
        colno: event.colno,
        href: window.location.href,
        userAgent: navigator.userAgent,
        timestamp: new Date().toISOString(),
      });
    },
    true
  );

  window.addEventListener('unhandledrejection', (event) => {
    const reason = event.reason;
    const message =
      reason instanceof Error ? reason.message : typeof reason === 'string' ? reason : JSON.stringify(reason);
    const stack = reason instanceof Error ? reason.stack : undefined;
    void postReport(pluginId, {
      pluginId,
      kind: 'unhandledrejection',
      message,
      stack,
      href: window.location.href,
      userAgent: navigator.userAgent,
      timestamp: new Date().toISOString(),
    });
  });
}
