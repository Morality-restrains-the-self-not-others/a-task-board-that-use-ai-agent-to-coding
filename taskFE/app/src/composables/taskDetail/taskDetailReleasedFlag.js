/**
 * 统一的「容器/服务器已释放」标志读取（兼容 ref 与裸布尔）。
 * 服务器已释放 / 非服务时，容器写路径（指令发送、文件树、job 动作）应短路，
 * 避免打容器得到 409 误导用户；SaaS job 日志（COS step_full）不受此限制。
 */
export function isReleasedFlag(v) {
  if (v && typeof v === 'object' && 'value' in v) return Boolean(v.value)
  return Boolean(v)
}
