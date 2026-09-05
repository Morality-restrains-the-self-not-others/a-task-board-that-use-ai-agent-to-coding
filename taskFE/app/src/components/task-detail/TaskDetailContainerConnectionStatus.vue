<template>
  <div
    v-if="visible"
    class="mt-3 pt-3 border-t border-gray-100 space-y-2"
    data-testid="container-connection-status"
  >
    <div class="flex items-center justify-between gap-2">
      <h4 class="text-xs font-medium text-gray-500">容器连接状态</h4>
      <button
        type="button"
        class="px-2 py-1 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50 shrink-0"
        data-testid="server-status-copy"
        @click="copyConnectionStatus"
      >
        {{ copyState === 'copied' ? '已复制' : '复制连接状态' }}
      </button>
    </div>
    <div class="flex items-center gap-1.5 text-xs flex-wrap">
      <span
        class="inline-block w-2 h-2 rounded-full shrink-0"
        :class="{
          'bg-gray-300': containerHeartbeatStatus === 'idle',
          'bg-yellow-400 animate-pulse': containerHeartbeatStatus === 'connecting',
          'bg-green-500': containerHeartbeatStatus === 'connected',
          'bg-red-500': containerHeartbeatStatus === 'disconnected'
        }"
      />
      <span :class="{
        'text-gray-400': containerHeartbeatStatus === 'idle',
        'text-yellow-600': containerHeartbeatStatus === 'connecting',
        'text-green-600': containerHeartbeatStatus === 'connected',
        'text-red-500': containerHeartbeatStatus === 'disconnected'
      }">
        {{ containerStatusLabel }}
      </span>
      <template v-if="containerHeartbeatStatus === 'connected' && containerHeartbeatLastSuccess">
        <span class="text-gray-300">·</span>
        <span class="text-gray-400">{{ containerHeartbeatLastSuccess.toLocaleTimeString('zh-CN') }}</span>
      </template>
      <template v-if="containerHeartbeatStatus !== 'idle'">
        <span class="text-gray-300">·</span>
        <span class="text-gray-400">检测{{ containerHeartbeatAttempts }}次</span>
      </template>
      <template v-if="containerHeartbeatStatus === 'disconnected' && containerHeartbeatError">
        <span class="text-gray-300">·</span>
        <span class="text-red-400 truncate max-w-48" :title="containerHeartbeatError">{{ containerHeartbeatError }}</span>
      </template>
      <template v-else-if="containerHeartbeatStatus === 'connecting' && containerHeartbeatError">
        <span class="text-gray-300">·</span>
        <span class="text-amber-600/90 truncate max-w-[18rem]" :title="containerHeartbeatError">{{ containerHeartbeatError }}</span>
      </template>
    </div>

    <div
      v-if="showSeqAckPanel"
      class="rounded-md bg-gray-50 border border-gray-100 px-2 py-1.5 text-[11px] leading-relaxed font-mono text-gray-600 space-y-1"
    >
      <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
        <span class="text-gray-400 shrink-0">上行</span>
        <span>容器→SaaS</span>
        <span>seq<span class="text-gray-400">=</span>{{ fmtSeq(containerHeartbeatSeqInfo.containerSeq) }}</span>
        <span class="text-gray-300">|</span>
        <span>SaaS ack<span class="text-gray-400">=</span>{{ fmtSeq(containerHeartbeatSeqInfo.saasAck) }}</span>
        <span
          class="shrink-0 font-sans font-medium"
          :class="okClass(containerHeartbeatSeqInfo.uplinkOk)"
          :title="okTitle(containerHeartbeatSeqInfo.uplinkOk, '上行')"
        >{{ okLabel(containerHeartbeatSeqInfo.uplinkOk) }}</span>
      </div>
      <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
        <span class="text-gray-400 shrink-0">下行</span>
        <span>SaaS→容器</span>
        <span>seq<span class="text-gray-400">=</span>{{ fmtSeq(containerHeartbeatSeqInfo.saasSeq) }}</span>
        <span class="text-gray-300">|</span>
        <span>容器 ack<span class="text-gray-400">=</span>{{ fmtSeq(containerHeartbeatSeqInfo.containerAck) }}</span>
        <span
          v-if="containerHeartbeatSeqInfo.probeOk !== null"
          class="text-gray-400 font-sans"
          :title="containerHeartbeatSeqInfo.probeOk ? '本轮 HTTP 探测成功' : '本轮 HTTP 探测失败'"
        >probe{{ containerHeartbeatSeqInfo.probeOk ? '✓' : '✗' }}</span>
        <span
          class="shrink-0 font-sans font-medium"
          :class="okClass(containerHeartbeatSeqInfo.downlinkOk)"
          :title="okTitle(containerHeartbeatSeqInfo.downlinkOk, '下行')"
        >{{ okLabel(containerHeartbeatSeqInfo.downlinkOk) }}</span>
      </div>
      <div
        v-if="containerHeartbeatSeqInfo.bidirectionalOk !== null"
        class="text-[10px] font-sans text-gray-400 pt-0.5 border-t border-gray-100/80"
      >
        双向确认
        <span :class="containerHeartbeatSeqInfo.bidirectionalOk ? 'text-green-600' : 'text-amber-600'">
          {{ containerHeartbeatSeqInfo.bidirectionalOk ? '已达成' : '未达成' }}
        </span>
      </div>
    </div>

    <div
      v-if="heartbeatLogSummary.count > 0"
      class="mt-1"
      data-testid="container-heartbeat-logs"
    >
      <button
        type="button"
        class="w-full flex items-start gap-1.5 text-left rounded-md border border-gray-100 bg-gray-50/80 px-2 py-1 hover:bg-gray-50 focus:outline-none focus:ring-1 focus:ring-primary/40"
        data-testid="container-heartbeat-logs-toggle"
        :aria-expanded="heartbeatLogsExpanded"
        @click="heartbeatLogsExpanded = !heartbeatLogsExpanded"
      >
        <span class="text-[10px] text-gray-400 shrink-0 mt-px" aria-hidden="true">
          {{ heartbeatLogsExpanded ? '▾' : '▸' }}
        </span>
        <span class="min-w-0 flex-1">
          <span class="text-[10px] font-medium text-gray-500">
            心跳探测日志
            <span class="font-normal text-gray-400">（{{ heartbeatLogSummary.count }} 条）</span>
          </span>
          <span
            v-if="!heartbeatLogsExpanded && heartbeatLogSummary.latestPreview"
            class="block mt-0.5 text-[10px] font-mono text-gray-400 truncate"
            :title="heartbeatLogSummary.latestPreview"
          >最新 {{ heartbeatLogSummary.latestPreview }}</span>
        </span>
      </button>
      <div
        v-if="heartbeatLogsExpanded"
        class="mt-1 bg-gray-50 border border-gray-100 rounded-md max-h-24 overflow-y-auto px-2 py-1 text-[10px] font-mono text-gray-500 leading-relaxed"
        data-testid="container-heartbeat-logs-body"
      >
        <p
          v-for="(line, index) in containerHeartbeatLogLines"
          :key="index"
          class="mb-0.5 break-all"
        >{{ formatHeartbeatLogLine(line) }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import {
  formatHeartbeatLogLine,
  summarizeHeartbeatLogRing,
} from '../../utils/containerHeartbeatLogRing.js'

const props = defineProps({
  containerHeartbeatStatus: { type: String, required: true },
  containerHeartbeatLastSuccess: { type: Date, default: null },
  containerHeartbeatAttempts: { type: Number, required: true },
  containerHeartbeatError: { type: String, default: '' },
  containerHeartbeatSeqInfo: {
    type: Object,
    default: () => ({
      containerSeq: null,
      containerAck: null,
      saasSeq: null,
      saasAck: null,
      uplinkOk: null,
      downlinkOk: null,
      probeOk: null,
      bidirectionalOk: null,
    }),
  },
  containerHeartbeatLogLines: {
    type: Array,
    default: () => [],
  },
  /** 为 false 时即使非 idle 也不渲染（供父级条件挂载） */
  forceVisible: { type: Boolean, default: false },
})

const heartbeatLogsExpanded = ref(false)
const heartbeatLogSummary = computed(() => summarizeHeartbeatLogRing(props.containerHeartbeatLogLines))

const visible = computed(() => {
  if (props.forceVisible) return true
  return (
    props.containerHeartbeatStatus !== 'idle' ||
    (props.containerHeartbeatLogLines?.length || 0) > 0
  )
})

const containerStatusLabel = computed(() => {
  const s = props.containerHeartbeatStatus
  if (s === 'idle') return '等待连接'
  if (s === 'connecting') {
    const err = String(props.containerHeartbeatError || '')
    if (err.includes('登记') || err.includes('初始化')) return '等待容器登记'
    return err ? '单向连接中' : '连接中'
  }
  if (s === 'connected') return '双向已连接'
  return '连接断开'
})

const showSeqAckPanel = computed(() => {
  if (props.containerHeartbeatStatus === 'idle') return false
  const s = props.containerHeartbeatSeqInfo || {}
  return (
    s.containerSeq != null ||
    s.containerAck != null ||
    s.saasSeq != null ||
    s.saasAck != null
  )
})

watch(
  () => props.containerHeartbeatLogLines?.length ?? 0,
  (n, prev) => {
    if (n === 0 && prev > 0) heartbeatLogsExpanded.value = false
  },
)

function fmtSeq(v) {
  return v == null ? '—' : String(v)
}

function okClass(ok) {
  if (ok === true) return 'text-green-600'
  if (ok === false) return 'text-amber-600'
  return 'text-gray-400'
}

function okLabel(ok) {
  if (ok === true) return '✓'
  if (ok === false) return '✗'
  return '·'
}

function okTitle(ok, dir) {
  if (ok === true) return `${dir}已确认`
  if (ok === false) return `${dir}未确认`
  return `${dir}待确认`
}

function okText(ok) {
  if (ok === true) return '已确认'
  if (ok === false) return '未确认'
  return '待确认'
}

const copyState = ref('idle')
let copyStateTimer = null

const connectionStatusText = computed(() => {
  const lines = []
  lines.push(`快照时间: ${new Date().toLocaleString('zh-CN')}`)
  lines.push('')
  lines.push('=== 容器连接状态 ===')
  lines.push(`状态: ${containerStatusLabel.value} (${props.containerHeartbeatStatus})`)
  if (props.containerHeartbeatStatus === 'connected' && props.containerHeartbeatLastSuccess) {
    lines.push(`最后成功: ${props.containerHeartbeatLastSuccess.toLocaleString('zh-CN')}`)
  }
  if (props.containerHeartbeatStatus !== 'idle') {
    lines.push(`检测次数: ${props.containerHeartbeatAttempts}`)
  }
  if (props.containerHeartbeatError) {
    lines.push(`错误: ${props.containerHeartbeatError}`)
  }
  if (showSeqAckPanel.value) {
    const s = props.containerHeartbeatSeqInfo || {}
    lines.push('')
    lines.push('=== 双向确认 ===')
    lines.push(
      `上行 容器→SaaS: seq=${fmtSeq(s.containerSeq)}, SaaS ack=${fmtSeq(s.saasAck)}, ${okText(s.uplinkOk)}`,
    )
    const probe = s.probeOk === null ? '' : `, probe=${s.probeOk ? '成功' : '失败'}`
    lines.push(
      `下行 SaaS→容器: seq=${fmtSeq(s.saasSeq)}, 容器 ack=${fmtSeq(s.containerAck)}, ${okText(s.downlinkOk)}${probe}`,
    )
    if (s.bidirectionalOk !== null) {
      lines.push(`双向确认: ${s.bidirectionalOk ? '已达成' : '未达成'}`)
    }
  }
  if (heartbeatLogSummary.value.count > 0) {
    lines.push('')
    lines.push(`=== 心跳探测日志（最近 ${heartbeatLogSummary.value.count} 条）===`)
    for (const log of props.containerHeartbeatLogLines) {
      lines.push(`  - ${formatHeartbeatLogLine(log)}`)
    }
  }
  return lines.join('\n')
})

const copyConnectionStatus = async () => {
  const text = connectionStatusText.value
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    copyState.value = 'copied'
    if (copyStateTimer) clearTimeout(copyStateTimer)
    copyStateTimer = window.setTimeout(() => {
      copyState.value = 'idle'
      copyStateTimer = null
    }, 1500)
  } catch (error) {
    console.error('复制连接状态失败:', error)
  }
}
</script>
