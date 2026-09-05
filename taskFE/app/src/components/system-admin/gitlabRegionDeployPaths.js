/**
 * Format runAll service name + optional Docker container for admin display.
 * @param {string} serviceProcess
 * @param {string} [containerName]
 * @returns {string}
 */
export function formatGitlabRegionServiceProcess(serviceProcess, containerName) {
  const svc = String(serviceProcess || '').trim()
  const ctn = String(containerName || '').trim()
  if (!svc && !ctn) return '（未解析）'
  if (svc && ctn) return `${svc}（容器: ${ctn}）`
  return svc || `容器: ${ctn}`
}

const PRIMARY_GIT_SERVICE_START = 'bash gitService/run.sh start'

/**
 * Derive the copy-paste start command from runAll/conf service name.
 * Keep in sync with taskBill gitlabRegionServiceStart.
 * @param {string} serviceProcess
 * @returns {string}
 */
export function deriveGitlabRegionServiceStart(serviceProcess) {
  const proc = String(serviceProcess || '').trim()
  if (!proc) return ''
  if (proc === 'git-service') return PRIMARY_GIT_SERVICE_START
  if (proc === 'git-service-tencent-sh-1') {
    return 'bash gitService/scripts/deploy_tencent_sh_1.sh'
  }
  return `GITSERVICE_CONF_APP=${proc} bash gitService/run.sh start`
}

/**
 * Prefer API start command, but replace a stale primary command when the
 * process is a different git-service instance.
 * @param {string} serviceProcess
 * @param {string} [serviceStart]
 * @returns {string}
 */
export function formatGitlabRegionServiceStart(serviceProcess, serviceStart) {
  const derived = deriveGitlabRegionServiceStart(serviceProcess)
  const api = String(serviceStart || '').trim()
  if (!api) return derived
  if (api === PRIMARY_GIT_SERVICE_START && derived && derived !== PRIMARY_GIT_SERVICE_START) {
    return derived
  }
  return api
}
