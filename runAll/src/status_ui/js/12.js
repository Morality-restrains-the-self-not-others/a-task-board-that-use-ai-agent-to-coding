// status_ui 拆分块（OPT-20260810-040）：clearAllObservability 独立功能块，经 manifest.json 拼接后与主文件同 scope 执行。

async function clearAllObservability() {
  const confirmed = await showModalConfirm(
    '将清空：① 所有服务内存/tee 日志 ② Loki 日志 ③ Tempo Trace ④ Prometheus 指标（含 span-metrics）。Grafana 仪表盘配置保留。数秒内查询将不再显示历史。是否继续？',
    {opLabel: '清空可观测数据'}
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/observability/clear-all', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('清空可观测数据失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    const summary = [
      `status: ${result.status || 'unknown'}`,
      `memory cleared: ${result.memory_services_cleared ?? 0}`,
      `files truncated: ${result.files_truncated ?? 0}`,
      `loki: ${result.loki_reset || '-'}`,
      `promtail: ${result.promtail_reset || '-'}`,
      `tempo: ${result.tempo_reset || '-'}`,
      `prometheus: ${result.prometheus_reset || '-'}`,
    ].join('\n');
    await showModalAlert(`可观测数据已清空:\n${summary}`, {type: 'success', opLabel: '清空可观测数据'});
    if (logsState.open) {
      fetchLogsOnce();
    }
    refresh();
  } catch (err) {
    showRequestError('清空可观测数据失败: ' + err.message, err);
  }
}

async function loadObservabilityBar() {
  const bar = document.getElementById('observability-bar');
  if (!bar) {
    return;
  }
  try {
    const resp = await fetch('/api/observability');
    const data = await parseJsonSafe(resp);
    if (!resp.ok) {
      bar.innerHTML = '<span class="obs-label">集中日志:</span><span>未配置（启动 ai-monitor 服务）</span>';
      return;
    }
    observabilityState.grafanaUrl = data.grafana_url || '';
    observabilityState.lokiUrl = data.loki_url || '';
    observabilityState.lokiPushUrl = data.loki_push_url || '';
    observabilityState.logShipping = data.log_shipping || '';
    observabilityState.grafanaLokiExplore = data.grafana_loki_explore || '';
    observabilityState.grafanaTempoExplore = data.grafana_tempo_explore || '';
    observabilityState.logFileRoot = data.log_file_root || '';
    const logShippingText = observabilityState.logFileRoot
      ? (observabilityState.logShipping === 'local_promtail_remote_loki'
        ? `tee → ${observabilityState.logFileRoot} → scripts/runall-local-promtail.sh → Loki ${observabilityState.lokiPushUrl || observabilityState.lokiUrl}`
        : `tee → ${observabilityState.logFileRoot} → Promtail`)
      : 'tee 未启用（配置 logging.file_root）';
    const parts = [
      '<span class="obs-label">可观测:</span>',
      observabilityState.grafanaTempoExplore
        ? `<a href="${observabilityState.grafanaTempoExplore}" target="_blank" rel="noopener noreferrer">Tempo Explore</a>`
        : '',
      observabilityState.grafanaLokiExplore
        ? `<a href="${observabilityState.grafanaLokiExplore}" target="_blank" rel="noopener noreferrer">Loki Explore</a>`
        : '',
      observabilityState.grafanaUrl
        ? `<a href="${observabilityState.grafanaUrl}" target="_blank" rel="noopener noreferrer">Grafana</a>`
        : '',
      observabilityState.lokiUrl
        ? `<span>Loki API <a href="${observabilityState.lokiUrl}/ready" target="_blank" rel="noopener noreferrer">${observabilityState.lokiUrl}</a></span>`
        : '',
      `<span class="obs-path" title="${logShippingText.replace(/"/g, '&quot;')}">${logShippingText}</span>`,
      '<button id="obs-clear-all" class="logs-panel-btn obs-clear-all-btn" type="button">清空 Grafana 可观测数据</button>',
    ].filter(Boolean);
    bar.innerHTML = parts.join(' · ');
    const clearBtn = document.getElementById('obs-clear-all');
    if (clearBtn) {
      clearBtn.addEventListener('click', (event) => {
        pulseClickFeedback(event.currentTarget);
        clearAllObservability();
      });
    }
  } catch (err) {
    bar.innerHTML = `<span class="obs-label">集中日志:</span><span>加载失败: ${err.message}</span>`;
  }
}


async function openLokiExplore() {
  const url = observabilityState.grafanaLokiExplore || observabilityState.lokiUrl;
  if (!url) {
    await showModalAlert('Loki 未配置。请确认 ai-monitor 服务已启动。', {type: 'warning', opLabel: '打开 Loki 日志'});
    return;
  }
  window.open(url, '_blank', 'noopener,noreferrer');
}

async function openGrafanaTraceLogs() {
  if (!logsState.service) {
    await showModalAlert('Select a service first.', {type: 'warning', opLabel: '打开 Grafana Trace 日志'});
    return;
  }
  const params = new URLSearchParams({ service: logsState.service });
  const fromRows = extractTraceIdFromLogRows(logsState.lastRows);
  if (fromRows) {
    params.set('trace_id', fromRows);
  }
  try {
    const resp = await fetch(`/api/observability/grafana-trace?${params.toString()}`);
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      const manual = await showModalPrompt('未从日志中识别 trace_id，请手动输入：', fromRows || '');
      if (!manual) {
        return;
      }
      const retry = await fetch(`/api/observability/grafana-trace?${new URLSearchParams({ trace_id: manual.trim() }).toString()}`);
      const retryResult = await parseJsonSafe(retry);
      if (!retry.ok || !retryResult.url) {
        showRequestError(`打开 Grafana 失败: ${retryResult.error || retry.status}`, resolveTraceIdFromResponse(retry, retry.requestTraceId, retryResult));
        return;
      }
      window.open(retryResult.url, '_blank', 'noopener,noreferrer');
      return;
    }
    if (!result.url) {
      await showModalAlert('Grafana 链接为空', {type: 'warning', opLabel: '打开 Grafana Trace 日志'});
      return;
    }
    window.open(result.url, '_blank', 'noopener,noreferrer');
  } catch (err) {
    showRequestError(`打开 Grafana 失败: ${err.message}`, err);
  }
}
