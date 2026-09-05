/** 与后端 UserDataTemplate.USERDATA_TEMPLATE_OS_CHOICES 对齐 */
export const USERDATA_OS_GROUPS = [
  {
    label: 'CentOS / Rocky Linux / AlmaLinux',
    options: [
      { value: 'centos_7', label: 'CentOS 7' },
      { value: 'centos_8_stream', label: 'CentOS Stream 8' },
      { value: 'rocky_8', label: 'Rocky Linux 8' },
      { value: 'rocky_9', label: 'Rocky Linux 9' },
      { value: 'almalinux_8', label: 'AlmaLinux 8' },
      { value: 'almalinux_9', label: 'AlmaLinux 9' },
    ],
  },
  {
    label: 'Ubuntu',
    options: [
      { value: 'ubuntu_20_04', label: 'Ubuntu 20.04 LTS' },
      { value: 'ubuntu_22_04', label: 'Ubuntu 22.04 LTS' },
      { value: 'ubuntu_24_04', label: 'Ubuntu 24.04 LTS' },
    ],
  },
  {
    label: 'Debian',
    options: [
      { value: 'debian_11', label: 'Debian 11 (bullseye)' },
      { value: 'debian_12', label: 'Debian 12 (bookworm)' },
    ],
  },
  {
    label: 'Windows Server',
    options: [
      { value: 'windows_server_2019', label: 'Windows Server 2019' },
      { value: 'windows_server_2022', label: 'Windows Server 2022' },
    ],
  },
  {
    label: '其它（仅大类，兼容旧模板）',
    options: [
      { value: 'centos', label: 'CentOS（未指定版本）' },
      { value: 'ubuntu', label: 'Ubuntu（未指定版本）' },
      { value: 'debian', label: 'Debian（未指定版本）' },
      { value: 'windows', label: 'Windows（未指定版本）' },
    ],
  },
]

export function userdataOsOptionsFlat() {
  return USERDATA_OS_GROUPS.flatMap((g) => g.options)
}

export function isWindowsOs(osType) {
  const t = String(osType || '')
  return t === 'windows' || t.startsWith('windows')
}

export function linuxFamilyForScript(osType) {
  const t = String(osType || '')
  if (!t || isWindowsOs(osType)) return null
  if (t === 'rhel' || t.startsWith('centos') || t.startsWith('rocky') || t.startsWith('almalinux')) {
    return 'centos'
  }
  if (t === 'ubuntu' || t.startsWith('ubuntu')) return 'ubuntu'
  if (t === 'debian' || t.startsWith('debian')) return 'debian'
  return 'ubuntu'
}

export function getOsTypeLabel(osType) {
  const opt = userdataOsOptionsFlat().find((o) => o.value === osType)
  if (opt) return opt.label
  const legacy = {
    centos: 'CentOS',
    ubuntu: 'Ubuntu',
    debian: 'Debian',
    windows: 'Windows',
  }
  return legacy[osType] || osType || ''
}

/** 脚本中已由模板导出的环境名；UserData 列表里若重复且值为空，会覆盖正确值（如 CONTAINER_IMAGE 被置空导致 docker run 无镜像） */
export const RESERVED_USERDATA_ENV_NAMES = new Set([
  'ACCESS_TOKEN',
  'TASK_API_ENDPOINT',
  'CONTAINER_NAME',
  'CONTAINER_IMAGE',
  'COMMENT_ID',
  'TRACE_ID',
])

export function userdataVarsExcludingReservedEnv(userdataVars) {
  return (userdataVars || []).filter(
    (v) => v.name && !RESERVED_USERDATA_ENV_NAMES.has(v.name),
  )
}

/** Bash 单引号内嵌单引号的标准转义 */
export function escapeBashSingleQuoted(s) {
  return String(s || '').replace(/'/g, "'\\''")
}

/**
 * Linux UserData：推导 Trae 换票所需的 BUSINESS_API_ENDPOINT（公网 IP + BUSINESS_HOST_PORT + /api）。
 * 容器变量 BUSINESS_API_ENDPOINT 非空时优先使用（手动指定完整 URL）。
 * 须在脚本中已赋值 BUSINESS_HOST_PORT（容器变量 BUSINESS_HOST_PORT，默认 8765）。
 * @param {boolean} useLog - true 时用 log()（完整安装脚本）；false 时用 echo（精简预览脚本）
 */
export function linuxBusinessApiSetupSnippet(explicitBusinessApi, useLog = true) {
  const ex = escapeBashSingleQuoted(explicitBusinessApi || '')
  const say = useLog ? 'log' : 'echo'
  return [
    '# Trae onlineServiceJS：BUSINESS_API_ENDPOINT + TRAE_PUBLIC_IP（UserData 注入；镜像内禁止写死 IP）',
    `BUSINESS_API_ENDPOINT_EXPLICIT='${ex}'`,
    'if [ -n "$BUSINESS_API_ENDPOINT_EXPLICIT" ]; then',
    '  export BUSINESS_API_ENDPOINT="$BUSINESS_API_ENDPOINT_EXPLICIT"',
    '  # 显式 URL 的 host 若为 IPv4，同步注入 TRAE_PUBLIC_IP（容器内不再外网探测）',
    '  if [ -z "${TRAE_PUBLIC_IP:-}" ]; then',
    '    _biz_host=$(printf \'%s\' "$BUSINESS_API_ENDPOINT" | sed -n \'s|^[a-zA-Z][a-zA-Z0-9+.-]*://\\([^/:]*\\).*|\\1|p\')',
    '    case "$_biz_host" in',
    '      [0-9]*.[0-9]*.[0-9]*.[0-9]*) export TRAE_PUBLIC_IP="$_biz_host" ;;',
    '    esac',
    '    unset _biz_host',
    '  fi',
    'else',
    '  resolve_task2app_public_ip() {',
    '    local ip=""',
    '    for meta in "http://100.100.100.200/latest/meta-data/eipv4" "http://100.100.100.200/latest/meta-data/public-ipv4" "http://metadata.tencentyun.com/latest/meta-data/public-ipv4" "http://169.254.169.254/latest/meta-data/public-ipv4"; do',
    '      ip=$(curl -fsS --connect-timeout 2 "$meta" 2>/dev/null | head -1 | tr -d \'\\r\')',
    '      if [ -n "$ip" ] && [ "$ip" != "0.0.0.0" ]; then',
    '        echo "$ip"',
    '        return 0',
    '      fi',
    '    done',
    '    return 1',
    '  }',
    '  PUB_IP=$(resolve_task2app_public_ip || true)',
    '  BUSINESS_HOST_PORT="${BUSINESS_HOST_PORT:-8765}"',
    '  if [ -n "$PUB_IP" ]; then',
    '    export TRAE_PUBLIC_IP="$PUB_IP"',
    '    export BUSINESS_API_ENDPOINT="http://${PUB_IP}:${BUSINESS_HOST_PORT}/api"',
    '    ' + say + ' "推导 TRAE_PUBLIC_IP=$TRAE_PUBLIC_IP BUSINESS_API_ENDPOINT=$BUSINESS_API_ENDPOINT"',
    '  else',
    '    ' + say + ' "警告: 未能解析公网 IP；请在容器变量 BUSINESS_API_ENDPOINT / TRAE_PUBLIC_IP 填写（如 http://EIP:${BUSINESS_HOST_PORT}/api）"',
    '    export BUSINESS_API_ENDPOINT=""',
    '  fi',
    'fi',
  ].join('\n')
}

/**
 * Linux UserData：逐步上报进度到 TASK_API_ENDPOINT → SSE 转发前端。
 * 依赖已 export 的 ACCESS_TOKEN / TASK_API_ENDPOINT；curl 未就绪时仅写本地 log。
 */
export function linuxBootProgressReportSnippet() {
  return [
    '# 逐步进度 → POST .../server-container-token/boot-progress/ → SSE',
    '# 携带 comment_id/container_name/trace_id，供任务详情评论级「启动日志」与 Loki 关联',
    'report_progress() {',
    '    local _p="$1"',
    '    local _m="$2"',
    '    local _s="${3:-processing}"',
    '    log "$_m"',
    '    if [ -z "${TASK_API_ENDPOINT:-}" ] || [ -z "${ACCESS_TOKEN:-}" ]; then',
    '        return 0',
    '    fi',
    '    if ! command -v curl >/dev/null 2>&1; then',
    '        return 0',
    '    fi',
    '    local _extra=""',
    '    if [ -n "${COMMENT_ID:-}" ] && [ "${COMMENT_ID}" != "-" ]; then',
    // 注意：shell 赋值两侧双引号须闭合（末尾 \\"" = 字面 \" + 闭合 "）
    '        _extra="${_extra},\\"comment_id\\":\\"${COMMENT_ID}\\""',
    '    fi',
    '    if [ -n "${CONTAINER_NAME:-}" ] && [ "${CONTAINER_NAME}" != "-" ]; then',
    '        _extra="${_extra},\\"container_name\\":\\"${CONTAINER_NAME}\\""',
    '    fi',
    '    if [ -n "${TRACE_ID:-}" ] && [ "${TRACE_ID}" != "-" ]; then',
    '        _extra="${_extra},\\"trace_id\\":\\"${TRACE_ID}\\""',
    '    fi',
    '    local _hdr=(-H "Content-Type: application/json")',
    '    if [ -n "${TRACE_ID:-}" ] && [ "${TRACE_ID}" != "-" ]; then',
    '        _hdr+=(-H "X-Trace-Id: ${TRACE_ID}")',
    '    fi',
    '    curl -fsS -m 10 -X POST \\',
    '      "${TASK_API_ENDPOINT%/}/server-container-token/boot-progress/" \\',
    '      "${_hdr[@]}" \\',
    '      -d "{\\"access_token\\":\\"${ACCESS_TOKEN}\\",\\"progress\\":${_p},\\"message\\":\\"${_m}\\",\\"status\\":\\"${_s}\\"${_extra}}" \\',
    '      >/dev/null 2>&1 || true',
    '}',
  ].join('\n')
}

/**
 * Windows UserData：逐步上报进度（与 Linux boot-progress 同路径）。
 * 依赖 $env:ACCESS_TOKEN / $env:TASK_API_ENDPOINT。
 */
export function powershellBootProgressReportSnippet() {
  return [
    'function Report-Progress {',
    '    param([int]$Progress, [string]$Message, [string]$Status = \'processing\')',
    '    Write-Output $Message',
    '    if (-not $env:TASK_API_ENDPOINT -or -not $env:ACCESS_TOKEN) { return }',
    '    try {',
    '        $bodyObj = @{ access_token = $env:ACCESS_TOKEN; progress = $Progress; message = $Message; status = $Status }',
    '        if ($env:COMMENT_ID -and $env:COMMENT_ID -ne \'-\') { $bodyObj.comment_id = $env:COMMENT_ID }',
    '        if ($env:CONTAINER_NAME -and $env:CONTAINER_NAME -ne \'-\') { $bodyObj.container_name = $env:CONTAINER_NAME }',
    '        if ($env:TRACE_ID -and $env:TRACE_ID -ne \'-\') { $bodyObj.trace_id = $env:TRACE_ID }',
    '        $body = $bodyObj | ConvertTo-Json -Compress',
    '        $uri = ($env:TASK_API_ENDPOINT.TrimEnd(\'/\')) + \'/server-container-token/boot-progress/\'',
    '        $headers = @{ \'Content-Type\' = \'application/json\' }',
    '        if ($env:TRACE_ID -and $env:TRACE_ID -ne \'-\') { $headers[\'X-Trace-Id\'] = $env:TRACE_ID }',
    '        Invoke-RestMethod -Uri $uri -Method Post -Body $body -Headers $headers -TimeoutSec 10 | Out-Null',
    '    } catch {}',
    '}',
  ].join('\n')
}

/** bash -c 构建的 RUN_CMD 追加：从当前 shell 继承已 export 的 BUSINESS_API_ENDPOINT / TRAE_PUBLIC_IP */
export function linuxDockerEnvBusinessApiAppend() {
  return [
    'if [ -n "$BUSINESS_API_ENDPOINT" ]; then',
    '  RUN_CMD="$RUN_CMD -e BUSINESS_API_ENDPOINT"',
    'fi',
    'if [ -n "$TRAE_PUBLIC_IP" ]; then',
    '  RUN_CMD="$RUN_CMD -e TRAE_PUBLIC_IP"',
    'fi',
  ].join('\n')
}

/** 精简占位符版云上 UserData（无 log 函数）：启动实例后解析公网 IP，供 docker run 继承换票变量 */
export function linuxMinimalBusinessBeforeContainerRun() {
  const bizLines = linuxBusinessApiSetupSnippet('', false).split('\n')
  return [
    'BUSINESS_HOST_PORT="${BUSINESS_HOST_PORT:-8765}"',
    ...bizLines,
    'EXTRA_BIZ=()',
    'if [ -n "$BUSINESS_API_ENDPOINT" ]; then EXTRA_BIZ+=(-e BUSINESS_API_ENDPOINT); fi',
    'if [ -n "$TRAE_PUBLIC_IP" ]; then EXTRA_BIZ+=(-e TRAE_PUBLIC_IP); fi',
  ].join('\n')
}

/** PowerShell 单引号字符串内嵌单引号 */
export function escapePowerShellSingleQuoted(s) {
  return String(s || '').replace(/'/g, "''")
}

/**
 * Windows UserData：设置 $envVars['BUSINESS_API_ENDPOINT']（与 Linux 逻辑一致）
 */
export function powershellBusinessApiEnvBlock(businessHostPort, explicitBusinessApi) {
  const ex = escapePowerShellSingleQuoted(explicitBusinessApi || '')
  const hp = escapePowerShellSingleQuoted(String(businessHostPort || '8765'))
  return [
    '$bizExplicit = \'' + ex + '\'',
    '$bhPort = \'' + hp + '\'',
    'if ($bizExplicit) {',
    '    $envVars[\'BUSINESS_API_ENDPOINT\'] = $bizExplicit',
    '    if (-not $envVars.ContainsKey(\'TRAE_PUBLIC_IP\') -or -not $envVars[\'TRAE_PUBLIC_IP\']) {',
    '        try {',
    '            $bizUri = [Uri]$bizExplicit',
    '            if ($bizUri.Host -match \'^\\d{1,3}(\\.\\d{1,3}){3}$\') { $envVars[\'TRAE_PUBLIC_IP\'] = $bizUri.Host }',
    '        } catch {}',
    '    }',
    '} else {',
    '    function Get-Task2AppPublicIp {',
    '        foreach ($u in @(\'http://100.100.100.200/latest/meta-data/eipv4\',\'http://100.100.100.200/latest/meta-data/public-ipv4\',\'http://metadata.tencentyun.com/latest/meta-data/public-ipv4\',\'http://169.254.169.254/latest/meta-data/public-ipv4\')) {',
    '            try {',
    '                $ip = (Invoke-WebRequest -Uri $u -TimeoutSec 2 -UseBasicParsing).Content.Trim()',
    '                if ($ip -and $ip -ne \'0.0.0.0\') { return $ip }',
    '            } catch {}',
    '        }',
    '        return $null',
    '    }',
    '    $pubIp = Get-Task2AppPublicIp',
    '    if ($pubIp) {',
    '        $envVars[\'TRAE_PUBLIC_IP\'] = $pubIp',
    '        $envVars[\'BUSINESS_API_ENDPOINT\'] = "http://$pubIp:$bhPort/api"',
    '    }',
    '}',
  ].join('\n')
}

export function formatDate(dateString) {
  if (!dateString) return '—'
  // Backend stores UTC as "YYYY-MM-DD HH:mm:ss.ffffff"; normalize for Date parsing.
  const normalized = String(dateString).includes('T')
    ? String(dateString)
    : String(dateString).replace(' ', 'T') + (String(dateString).endsWith('Z') ? '' : 'Z')
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}
