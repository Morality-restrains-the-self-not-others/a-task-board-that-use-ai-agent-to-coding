<template>
  <div
    v-if="serverStatus || isServerRunning || isServerStarting || runtimeStatus || statusLogs.length > 0"
    class="p-3 bg-white border border-gray-200 rounded-lg"
    data-testid="server-start-status-panel"
    v-bind="panelStartTraceId ? { 'data-traceId': panelStartTraceId } : {}"
  >
    <div class="space-y-1.5 mb-2">
      <div
        v-if="showServerLifecycleRow"
        class="flex items-center gap-2 flex-wrap"
        data-testid="server-lifecycle-row"
      >
        <h3 class="text-sm font-medium text-gray-500 shrink-0">服务器启动状态</h3>
        <span
          class="inline-block w-2.5 h-2.5 rounded-full shrink-0"
          :class="serverLifecycleDotClass"
          aria-hidden="true"
        />
        <span
          class="text-xs font-medium"
          :class="serverLifecycleTextClass"
          data-testid="server-lifecycle-status"
          role="status"
          :aria-label="`服务器启动状态 ${serverLifecycleLabel}`"
        >{{ serverLifecycleLabel }}</span>
      </div>
      <div class="flex items-center gap-2 flex-wrap" data-testid="sse-connection-row">
        <h3 class="text-sm font-medium text-gray-500 shrink-0">SSE 连接</h3>
        <button
          type="button"
          class="inline-block w-3 h-3 rounded-full shrink-0 ring-2 ring-gray-100 cursor-pointer hover:ring-primary/50 focus:outline-none focus:ring-2 focus:ring-primary transition-shadow disabled:cursor-wait"
          :class="sseLive ? 'bg-green-500' : (sseReconnecting ? 'bg-yellow-500 animate-pulse' : 'bg-gray-400')"
          :title="sseLive ? '点击断开 SSE 连接' : (sseReconnecting ? '点击立即重连' : '点击重新连接 SSE')"
          :aria-label="sseLive ? 'SSE 已连接，点击断开' : (sseReconnecting ? '正在重连，点击立即重连' : 'SSE 未连接，点击重连')"
          :disabled="sseReconnecting"
          @click="emit('sse-manual-reconnect')"
        />
        <span
          class="text-xs"
          :class="sseLive ? 'text-green-600' : (sseReconnecting ? 'text-yellow-600' : 'text-gray-400')"
          data-testid="sse-connection-status"
          role="status"
          :aria-label="sseAriaStatusName"
        >{{ sseDisplayText }}</span>
        <span
          v-if="ssePlatformRestartHint && !sseLive"
          class="text-xs text-amber-700"
          data-testid="sse-platform-restart-hint"
          title="与云服务器启停无关：任务状态推送通道（SSE/网关）暂不可达"
        >状态推送服务暂不可用</span>
      </div>
      <!-- OPT-20260724-025: per-container heartbeat 通信状态（仅 per-binding 面板渲染） -->
      <div
        v-if="heartbeatStatus"
        class="flex items-center gap-2 flex-wrap"
        data-testid="container-heartbeat-row"
      >
        <h3 class="text-sm font-medium text-gray-500 shrink-0">容器通信</h3>
        <span
          class="inline-block w-2.5 h-2.5 rounded-full shrink-0"
          :class="heartbeatDotClass"
          aria-hidden="true"
        />
        <span
          class="text-xs"
          :class="heartbeatTextClass"
          data-testid="container-heartbeat-status"
        >{{ heartbeatDisplayText }}</span>
        <span
          v-if="heartbeatSeqDisplay"
          class="text-xs text-gray-400 font-mono"
          data-testid="container-heartbeat-seq"
        >{{ heartbeatSeqDisplay }}</span>
        <span
          v-if="heartbeatError"
          class="text-xs text-red-500 truncate max-w-[200px]"
          :title="heartbeatError"
          data-testid="container-heartbeat-error"
        >{{ heartbeatError }}</span>
      </div>
    </div>
    <div v-if="serverStatus || statusLogs.length > 0" class="space-y-3">
      <div
        v-if="startupErrorBanner"
        class="rounded-md border px-3 py-2 text-sm"
        :class="startupErrorBanner.severity === 'warning'
          ? 'border-amber-200 bg-amber-50 text-amber-900'
          : 'border-red-200 bg-red-50 text-red-900'"
        data-testid="server-startup-error-banner"
        role="alert"
      >
        <p class="m-0 font-semibold">{{ startupErrorBanner.title }}</p>
        <p class="m-0 mt-1 text-xs opacity-90">{{ startupErrorBanner.hint }}</p>
        <a
          v-if="startupErrorBanner.showRecharge && tenantRechargePath"
          :href="tenantRechargePath"
          class="inline-block mt-2 text-xs font-medium underline hover:no-underline"
        >
          去购买
        </a>
      </div>
      <div v-if="statusProgress > 0 && statusProgress < 100" class="w-full bg-gray-200 rounded-full h-2">
        <div class="bg-primary h-2 rounded-full" :style="{ width: statusProgress + '%' }"></div>
      </div>
      <p
        v-if="statusMessage"
        class="text-gray-700"
        data-testid="server-start-status-message"
      >{{ statusMessage }}</p>
      <div v-if="statusLogs.length > 0" class="mt-2">
        <h4 class="text-xs font-medium text-gray-500 mb-1">启动日志</h4>
        <div class="bg-gray-50 p-2 rounded-md max-h-32 overflow-y-auto text-sm">
          <p v-for="(log, index) in statusLogs" :key="index" class="text-gray-600 mb-1">{{ log }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { classifyServerStartupError } from '../../utils/serverStartupErrorDisplay.js'
import {
  resolveServerLifecycleLabel,
  serverLifecycleDotClass as lifecycleDotClassFor,
  serverLifecycleTextClass as lifecycleTextClassFor,
} from '../../utils/serverLifecycleStatus.js'
import { resolveLifecycleFlagsFromRuntimeStatus } from '../../utils/serverLifecycleFromRuntime.js'

const route = useRoute()
const props = defineProps({
  serverStatus: { type: String, default: '' },
  isServerRunning: { type: Boolean, default: false },
  isServerStarting: { type: Boolean, default: false },
  /** 云 Describe runtime_status；冷打开标志位未回填时用于对齐「已启动」 */
  runtimeStatus: { type: String, default: '' },
  sseLive: { type: Boolean, required: true },
  sseReconnecting: { type: Boolean, required: true },
  sseReconnectAttempts: { type: Number, required: true },
  ssePlatformRestartHint: { type: Boolean, default: false },
  statusProgress: { type: Number, required: true },
  statusMessage: { type: String, required: true },
  statusLogs: { type: Array, default: () => [] },
  /** 本次启动链路 TraceId（HTTP/SSE）；挂 data-traceId 供排障与 Chrome 插件读取 */
  startTraceId: { type: String, default: '' },
  /** 兼容旧调用方 :status-trace-id */
  statusTraceId: { type: String, default: '' },
  /** OPT-20260724-025: per-container heartbeat 健康信息（可选，仅 per-binding 面板传入） */
  heartbeatStatus: { type: String, default: '' },
  heartbeatSeqInfo: { type: Object, default: null },
  heartbeatError: { type: String, default: '' },
})

const emit = defineEmits(['sse-manual-reconnect'])

const panelStartTraceId = computed(() =>
  String(props.startTraceId || props.statusTraceId || '').trim(),
)

const showServerLifecycleRow = computed(
  () =>
    Boolean(props.serverStatus) ||
    props.isServerRunning ||
    props.isServerStarting ||
    Boolean(resolveLifecycleFlagsFromRuntimeStatus(props.runtimeStatus)) ||
    (Array.isArray(props.statusLogs) && props.statusLogs.length > 0),
)

/** 与 SSE 无关：仅反映云服务器/VM 生命周期 */
const serverLifecycleLabel = computed(() =>
  resolveServerLifecycleLabel({
    isServerRunning: props.isServerRunning,
    isServerStarting: props.isServerStarting,
    serverStatus: props.serverStatus,
    runtimeStatus: props.runtimeStatus,
  }),
)

const serverLifecycleDotClass = computed(() => lifecycleDotClassFor(serverLifecycleLabel.value))

const serverLifecycleTextClass = computed(() => lifecycleTextClassFor(serverLifecycleLabel.value))

const sseDisplayText = computed(() => {
  if (props.sseLive) return 'SSE 已连接'
  if (props.sseReconnecting) return `正在重连（第 ${props.sseReconnectAttempts} 次）...`
  return 'SSE 未连接（点击圆点重连）'
})

/** 保留 Playwright 既有 accessible name：启动状态 SSE 已连接 */
const sseAriaStatusName = computed(() => {
  if (props.sseLive) return '启动状态 SSE 已连接'
  if (props.sseReconnecting) return `启动状态 SSE 正在重连（第 ${props.sseReconnectAttempts} 次）`
  return '启动状态 SSE 未连接'
})

const startupErrorBanner = computed(() => {
  if (props.serverStatus !== 'error') {
    return null
  }
  return classifyServerStartupError(props.statusMessage)
})

const tenantRechargePath = computed(() => {
  const tid = String(route.params.tenant || '').trim()
  return tid ? `/tenant/${tid}/billing/orders/create/` : ''
})

// =========================================================================
// OPT-20260724-025: per-container heartbeat 状态展示
// =========================================================================

const heartbeatDotClass = computed(() => {
  const s = props.heartbeatStatus
  if (s === 'connected') return 'bg-green-500'
  if (s === 'connecting') return 'bg-yellow-500 animate-pulse'
  return 'bg-gray-400'
})

const heartbeatTextClass = computed(() => {
  const s = props.heartbeatStatus
  if (s === 'connected') return 'text-green-600'
  if (s === 'connecting') return 'text-yellow-600'
  return 'text-gray-400'
})

const heartbeatDisplayText = computed(() => {
  const s = props.heartbeatStatus
  if (s === 'connected') return '双向通信正常'
  if (s === 'connecting') return '正在建立通信…'
  return s || ''
})

const heartbeatSeqDisplay = computed(() => {
  const si = props.heartbeatSeqInfo
  if (!si || typeof si !== 'object') return ''
  const parts = []
  if (si.bidirectionalOk != null) {
    parts.push(`双向:${si.bidirectionalOk ? '✓' : '✗'}`)
  }
  if (si.uplinkOk != null) parts.push(`上行:${si.uplinkOk ? '✓' : '✗'}`)
  if (si.downlinkOk != null) parts.push(`下行:${si.downlinkOk ? '✓' : '✗'}`)
  if (si.probeOk != null) parts.push(`探活:${si.probeOk ? '✓' : '✗'}`)
  if (si.containerSeq != null) parts.push(`seq:${si.containerSeq}`)
  if (si.containerAck != null) parts.push(`ack:${si.containerAck}`)
  return parts.join(' ')
})
</script>
