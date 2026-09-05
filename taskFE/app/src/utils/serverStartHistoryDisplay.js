import { formatUptimeSinceMs } from './serverRuntimeStatusDetails.js'

const START_REASON_LABELS = Object.freeze({
  cloud_vm: '云虚拟机启动',
  cloud_vm_manual: '任务详情手动选配启动',
  cloud_vm_template: '任务详情项目模版启动',
  cloud_vm_auto_run: '创建任务/工作台自动运行启动',
  cloud_vm_comment_mention: '评论区$镜像启动',
  cloud_vm_terminal_migrate: '终态迁移再供给启动',
  relay_local: '本地中继启动',
})

const STOP_REASON_LABELS = Object.freeze({
  relay_stop: '本地中继停止',
  stop_vm: '用户停止虚拟机',
  user_stop: '用户点击停止服务器',
  instruction_idle: '容器指令空闲超时回收',
  schedule_window_end: '排期窗口结束自动关闭',
  task_status_cancelled: '任务进度变为已取消',
  task_status_completed: '任务完成后释放',
  inbound_task_terminal: '任务已终态（容器入站补偿释放）',
  reconcile_task_terminal: '任务终态对账释放',
  cloud_server_stopped: '云服务器已停止',
  superseded_by_new_start: '被新启动会话取代',
  idle_recycle: '空闲回收',
  orphan_reconcile: '孤儿实例对账清理',
  runtime_released: '运行时已释放',
  stale_reachability_purge: '陈旧可达性清理',
})

function resolveHistoryStartMs(record) {
  if (!record || typeof record !== 'object') {
    return NaN
  }
  const raw = record.started_at || record.created_at
  if (!raw) {
    return NaN
  }
  return Date.parse(String(raw))
}

function resolveHistoryEndMs(record, nowMs) {
  if (record?.stopped_at) {
    return Date.parse(String(record.stopped_at))
  }
  return typeof nowMs === 'number' ? nowMs : Date.now()
}

/** 历史会话运行持续时间文案；无起点时返回 '-' */
export function formatServerStartHistoryDuration(record, nowMs = Date.now()) {
  const startedMs = resolveHistoryStartMs(record)
  if (Number.isNaN(startedMs)) {
    return '-'
  }
  const endMs = resolveHistoryEndMs(record, nowMs)
  if (Number.isNaN(endMs)) {
    return '-'
  }
  return formatUptimeSinceMs(startedMs, endMs)
}

/** 启动原因：映射 runtime_source；空为 '-'，未知码原样展示 */
export function formatServerStartHistoryStartReason(runtimeSource) {
  const raw = String(runtimeSource ?? '').trim()
  if (!raw) {
    return '-'
  }
  return START_REASON_LABELS[raw] || raw
}

/** 关闭原因：映射 stop_reason；空为 '-'，未知码原样展示 */
export function formatServerStartHistoryStopReason(stopReason) {
  const raw = String(stopReason ?? '').trim()
  if (!raw) {
    return '-'
  }
  return STOP_REASON_LABELS[raw] || raw
}
