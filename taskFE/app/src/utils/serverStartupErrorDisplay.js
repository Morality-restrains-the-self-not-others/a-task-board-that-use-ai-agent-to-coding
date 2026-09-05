/**
 * 将服务器启动/停止异步错误分类为可展示的标题与操作建议。
 */

/** @typedef {{ title: string, hint: string, code: string, severity: 'error' | 'warning', showRecharge?: boolean }} ServerStartupErrorDisplay */

/**
 * @param {string} message
 * @param {{ code?: string }} [extras]
 * @returns {ServerStartupErrorDisplay | null}
 */
export function classifyServerStartupError(message, extras = {}) {
  const msg = String(message || '').trim()
  if (!msg) {
    return null
  }
  const code = String(extras.code || '').trim()

  if (
    code === 'INSUFFICIENT_BALANCE' ||
    /余额不足|余额不足|INSUFFICIENT_BALANCE/i.test(msg)
  ) {
    return {
      severity: 'error',
      code: 'INSUFFICIENT_BALANCE',
      title: '资源配额不足',
      hint: '启动服务器需要足够资源配额，请先购买资源后再试。',
      showRecharge: true,
    }
  }

  if (
    /Zone\.NotOnSale|可用区.*停售|已停售|无可售资源/i.test(msg)
  ) {
    return {
      severity: 'error',
      code: 'ZONE_NOT_ON_SALE',
      title: '可用区已停售或无可售资源',
      hint: '请在硬件配置中更换可用区或地域，或联系管理员更新项目运行模版。',
    }
  }

  if (/NoStock|暂无库存|OperationDenied\.NoStock/i.test(msg)) {
    return {
      severity: 'error',
      code: 'INSTANCE_NO_STOCK',
      title: '实例规格暂无库存',
      hint: '当前可用区所选实例规格已无库存，请更换实例规格或地域后重试。',
    }
  }

  if (/InvalidAccountStatus\.NotEnoughBalance|账户.*余额|Account.*balance/i.test(msg)) {
    return {
      severity: 'error',
      code: 'CLOUD_ACCOUNT_BALANCE',
      title: '云平台账户余额不足',
      hint: '阿里云账户余额不足，请登录云厂商控制台充值后重试。',
    }
  }

  if (/架构不匹配|architecture/i.test(msg)) {
    return {
      severity: 'error',
      code: 'ARCH_MISMATCH',
      title: '实例规格与镜像架构不匹配',
      hint: '请更换与镜像 CPU 架构一致的实例规格，或选择支持该架构的镜像。',
    }
  }

  if (/未找到已安装镜像|未找到可用的云服务器镜像/i.test(msg)) {
    return {
      severity: 'error',
      code: 'IMAGE_NOT_FOUND',
      title: '镜像不可用',
      hint: '请确认任务已选择已安装的容器镜像，且镜像在目标地域有可用的云主机镜像。',
    }
  }

  if (/未配置云平台授权|authorization/i.test(msg)) {
    return {
      severity: 'warning',
      code: 'MISSING_CLOUD_AUTH',
      title: '缺少云平台授权',
      hint: '请在工作区或租户设置中配置并激活云平台授权。',
    }
  }

  if (
    code === 'CONTAINER_REACHABILITY_TIMEOUT' ||
    /容器服务未在时限内登记可达地址|register-reachability/i.test(msg)
  ) {
    return {
      severity: 'error',
      code: 'CONTAINER_REACHABILITY_TIMEOUT',
      title: '云主机已运行，容器未就绪',
      hint: '不是阿里云 API 连不上。实例已 Running，但容器未登记 HTTP 地址。请检查镜像 UserData、安全组出站与容器进程后重试。',
    }
  }

  return {
    severity: 'error',
    code: code || 'STARTUP_ERROR',
    title: '服务器启动失败',
    hint: msg,
  }
}

/**
 * 为 status 更新附加 error_code / error_hint，供 UI 醒目展示。
 * @param {Record<string, unknown>} statusUpdate
 * @returns {Record<string, unknown>}
 */
export function enrichStartupStatusUpdate(statusUpdate) {
  if (!statusUpdate || typeof statusUpdate !== 'object') {
    return statusUpdate
  }
  if (statusUpdate.status !== 'error') {
    return statusUpdate
  }
  const message = typeof statusUpdate.message === 'string' ? statusUpdate.message : ''
  const code =
    typeof statusUpdate.error_code === 'string' && statusUpdate.error_code
      ? statusUpdate.error_code
      : typeof statusUpdate.code === 'string'
        ? statusUpdate.code
        : ''
  const display = classifyServerStartupError(message, { code })
  if (!display) {
    return statusUpdate
  }
  return {
    ...statusUpdate,
    error_code: display.code,
    error_hint: display.hint,
    error_title: display.title,
    show_recharge: Boolean(display.showRecharge),
  }
}
