import { queryClientPublicIpForAutoSg } from './publicClientIp.js'

/**
 * 创建/更新任务且 auto_run=true 时，强制查询用户公网 IP 写入 payload，
 * 供 taskTaskService 异步 start-vm-auto 透传到自动安全组入网白名单。
 * 查询失败不阻断创建（后端仍可从创建请求 XFF 解析）。
 *
 * @param {Record<string, any>} payload
 * @returns {Promise<Record<string, any>>}
 */
export async function attachClientPublicIpForAutoRun(payload) {
  const next = payload && typeof payload === 'object' ? { ...payload } : {}
  if (next.auto_run !== true) {
    return next
  }
  const ip = await queryClientPublicIpForAutoSg()
  if (ip) {
    next.client_public_ip = ip
  }
  return next
}
