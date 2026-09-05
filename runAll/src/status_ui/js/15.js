function renderStatus(data) {
  const container = document.getElementById('services');
  let html = '';
  const byName = indexServicesByName(data);
  const byGroup = new Map();
  for (const svc of data) {
    const groupName = typeof svc.group === 'string' ? svc.group.trim() : '';
    const groupKey = groupName || 'ungrouped';
    if (!byGroup.has(groupKey)) byGroup.set(groupKey, []);
    byGroup.get(groupKey).push(svc);
  }

  const sortedGroups = [...byGroup.entries()].sort((a, b) => {
    if (a[0] === 'ungrouped') return 1;
    if (b[0] === 'ungrouped') return -1;
    return a[0].localeCompare(b[0]);
  });

  for (const [groupKey, services] of sortedGroups) {
    const collapsed = isGroupCollapsed(groupKey);
    const groupDotClass = groupHealthDotClass(services);
    const groupSummary = groupHealthSummary(services);
    const hasBuildable = services.some(svc => svc.buildable === true);
    const groupActions = groupKey !== 'ungrouped'
      ? `<span class="group-actions">`
        + `<button class="action-btn" type="button" data-action="start-group" data-group="${esc(groupKey)}">启动本组</button>`
        + `<button class="action-btn" type="button" data-action="stop-group" data-group="${esc(groupKey)}">关闭本组</button>`
        + (hasBuildable
            ? `<button class="action-btn build-btn rebuild-all-btn" type="button" data-action="build-group" data-group="${esc(groupKey)}">全部重新编译</button>`
            : `<button class="action-btn build-btn is-disabled" type="button" disabled aria-disabled="true" title="本组无可编译服务">全部重新编译</button>`)
        + `</span>`
      : '';
    const title = groupKey === 'ungrouped' ? 'Ungrouped' : `Group ${groupKey}`;
    html += `<section class="service-group${collapsed ? ' collapsed' : ''}" data-group="${esc(groupKey)}">`;
    html += `<div class="group-header">`;
    html += `<button class="group-toggle" type="button" data-action="toggle-group" data-group="${esc(groupKey)}" aria-expanded="${collapsed ? 'false' : 'true'}" title="${esc(groupSummary)}">`;
    html += `<span class="group-chevron" aria-hidden="true">▾</span>`;
    html += `<span class="dot ${groupDotClass}"></span>`;
    html += `<span class="group-title">${esc(title)}</span>`;
    html += `<span class="group-count">${esc(groupSummary)}</span>`;
    html += `</button>`;
    html += groupActions;
    html += `</div>`;
    html += `<div class="group-body">`;
    const sortedServices = [...services].sort((a, b) => {
      const depthA = a.depends_on ? a.depends_on.length : 0;
      const depthB = b.depends_on ? b.depends_on.length : 0;
      if (depthA !== depthB) return depthA - depthB;
      return String(a.name || '').localeCompare(String(b.name || ''));
    });
    for (const svc of sortedServices) {
      const cls = serviceDotClass(svc);
      const startable = isServiceStartable(svc.status) && !svc.listen_port_active;
      const toggleAction = startable ? 'start' : 'stop';
      const toggleLabel = startable ? '启动' : '关闭';
      const sessionID = typeof svc.session_id === 'string' ? svc.session_id.trim() : '';
      const healthPortCellHtml = healthPortCell(svc);
      const commandPortCellHtml = commandPortCell(svc);
      const buildable = svc.buildable === true;
      const language = typeof svc.language === 'string' && svc.language.trim()
        ? svc.language.trim()
        : '—';
      html += `<div class="service" data-name="${escAttr(svc.name)}">`;
      html += `<span class="dot ${cls}"></span>`;
      html += `<span class="ports" title="health / command">${healthPortCellHtml} / ${commandPortCellHtml}</span>`;
      html += `<span class="name" title="${esc(svc.name)}">${esc(svc.name)}</span>`;
      html += `<span class="status">${formatServiceStatus(svc)}</span>`;
      html += `<span class="language">${esc(language)}</span>`;
      html += `<span class="actions">`;
      html += `<button class="action-btn" type="button" data-action="${toggleAction}" data-name="${esc(svc.name)}" data-session-id="${esc(sessionID)}">${toggleLabel}</button>`;
      if (buildable) {
        html += `<button class="action-btn build-btn" type="button" data-action="build" data-name="${esc(svc.name)}">编译</button>`;
      } else {
        html += `<button class="action-btn build-btn is-disabled" type="button" disabled aria-disabled="true" title="未配置 build_command，不可编译">编译</button>`;
      }
      html += `<button class="action-btn restart-btn" type="button" data-action="restart" data-name="${esc(svc.name)}" data-session-id="${esc(sessionID)}">重启</button>`;
      html += `<button class="action-btn logs-btn" type="button" data-action="logs" data-name="${esc(svc.name)}">日志</button>`;
      html += `<button class="action-btn clear-logs-btn" type="button" data-action="clear-logs" data-name="${esc(svc.name)}">清空日志</button>`;
      html += `</span>`;
      html += `<div class="service-extra">`;
      if (svc.depends_on && svc.depends_on.length > 0) {
        html += `<span class="deps">`;
        for (const dep of svc.depends_on) {
          const dcls = resolveDepDotClass(dep, byName.get(dep.name), serviceDotClass);
          html += `<span class="dep" data-dep-name="${escAttr(dep.name)}" title="点击跳转到 ${escAttr(dep.name)}"><span class="dot ${dcls}"></span>${esc(dep.name)}</span>`;
        }
        html += `</span>`;
      }
      if (svc.listen_port_active && isServiceStartable(svc.status)) {
        html += `<span class="error-msg" title="配置端口仍有 LISTEN 进程">端口仍占用</span>`;
      }
      if (svc.failure_code || svc.error) {
        const code = svc.failure_code ? `[${esc(svc.failure_code)}] ` : '';
        const msg = svc.error ? esc(svc.error) : '';
        html += `<span class="error-msg" title="${esc(svc.failure_code || '')}">${code}${msg}</span>`;
      }
      if (svc.readiness_error) {
        html += `<span class="error-msg" title="readiness probe">${esc(svc.readiness_error)}</span>`;
      }
      if (svc.last_checked) {
        html += `<span class="last-checked">checked: ${formatTime(svc.last_checked)}</span>`;
      }
      html += `</div>`;
      html += `</div>`;
    }
    html += `</div>`;
    html += `</section>`;
  }
  container.innerHTML = html;

  const elapsed = Math.floor((Date.now() - startTime) / 1000);
  document.getElementById('uptime').textContent = `Uptime: ${elapsed}s`;
}
