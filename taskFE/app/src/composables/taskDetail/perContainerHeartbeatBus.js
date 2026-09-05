/**
 * OPT-20260724-022: 轻量事件总线 — SSE container_heartbeat 事件按 comment_id
 * 路由到 per-binding 状态，无需修改复杂的 deps 回调链。
 *
 * OPT-20260724-021: 扩展总线 — 新增 binding_advanced 事件 + 容器 running 信号，
 * 用于 SSE 驱动的 binding advance 自动重试。
 *
 * establishSSEConnection.js 收到带有 comment_id 的心跳时写入此 ref；
 * useCommentContainerBindings 侦听此 ref 自动更新 per-binding 心跳。
 */
import { ref } from 'vue'

/** 最近一条携带 comment_id 的 container_heartbeat SSE 事件数据 */
export const latestPerContainerHeartbeat = ref(null)

/**
 * OPT-20260724-021: CommentContainerBindingAdvanced 领域事件总线。
 * establishSSEConnection 收到 SSE binding_advanced 事件时写入；
 * useCommentContainerBindings 侦听并本地更新 binding 状态（省去 GET 请求）。
 */
export const latestBindingAdvanced = ref(null)

/**
 * OPT-20260724-021: 容器变为 running 信号（携带 comment_id）。
 * establishSSEConnection 在 container_heartbeat status=ok 且 bidirection_ok=true 时
 * 写入此 ref；useCommentContainerBindings 侦听并触发 advance 抢救卡住的 binding。
 */
export const latestContainerRunning = ref(null)

/**
 * 服务器调度启动进度 → per-binding 启动日志总线。
 * updateServerStatus 在写入任务级 statusLogs 后，若 statusData 含 comment_id，
 * 写入 { comment_id, messages[] }；useCommentContainerBindings 侦听并追加到该评论启动日志。
 */
export const latestServerStartupStatusForBinding = ref(null)

/**
 * OPT-20260822-062: 云实例进入 Released/Terminated 后请求重新拉取绑定列表。
 * useServerConfigRuntime.fetchServerRuntimeStatus 探测到终态时写入
 * { commentId, ts }；useCommentContainerBindings 侦听并 refreshBindings，
 * 使滞后的 running binding 尽快对齐服务端终态。
 */
export const bindingRefreshRequested = ref(null)
