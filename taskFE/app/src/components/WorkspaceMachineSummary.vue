<template>
  <div
    ref="rootEl"
    data-alias="workspace-machine-summary"
    class="inline-flex items-center gap-1.5 text-sm text-gray-600 min-w-0"
  >
    <template v-if="machineSummary">
      <button
        type="button"
        data-alias="machine-filter-started"
        class="inline-flex items-center gap-1 underline-offset-2 hover:text-primary focus:outline-none focus-visible:ring-1 focus-visible:ring-primary rounded-sm shrink-0"
        :class="machineRuntimeFilter === 'started' ? 'text-primary font-medium underline' : 'text-gray-600'"
        :aria-pressed="machineRuntimeFilter === 'started' ? 'true' : 'false'"
        :aria-label="`已启动 ${machineSummary.startedCount}`"
        @click="emit('machine-filter', 'started')"
      >
        <span
          class="inline-block w-3 h-3 rounded-full border-2 border-emerald-500 shrink-0"
          aria-hidden="true"
        />
        <span data-testid="machine-summary-started-count">{{ machineSummary.startedCount }}</span>
      </button>
      <span class="text-gray-400 shrink-0" aria-hidden="true">·</span>
      <button
        type="button"
        data-alias="machine-filter-idle"
        class="inline-flex items-center gap-1 underline-offset-2 hover:text-primary focus:outline-none focus-visible:ring-1 focus-visible:ring-primary rounded-sm shrink-0"
        :class="machineRuntimeFilter === 'idle' ? 'text-primary font-medium underline' : 'text-gray-600'"
        :aria-pressed="machineRuntimeFilter === 'idle' ? 'true' : 'false'"
        :aria-label="`闲置 ${machineSummary.idleCount}`"
        @click="emit('machine-filter', 'idle')"
      >
        <span
          class="inline-block w-3 h-3 rounded-full border-2 border-dashed border-gray-400 shrink-0"
          aria-hidden="true"
        />
        <span data-testid="machine-summary-idle-count">{{ machineSummary.idleCount }}</span>
      </button>
      <template v-if="machineSummary.startingCount > 0">
        <span class="text-gray-400 shrink-0" aria-hidden="true">·</span>
        <span
          data-alias="machine-summary-starting"
          class="inline-flex items-center gap-1 text-amber-600 shrink-0"
          :aria-label="`启动中 ${machineSummary.startingCount}`"
        >
          <span
            class="inline-block w-2 h-2 rounded-full bg-amber-500 shrink-0 animate-pulse"
            aria-hidden="true"
          />
          <span data-testid="machine-summary-starting-count">{{ machineSummary.startingCount }}</span>
        </span>
      </template>
    </template>
    <template v-else>
      <span aria-hidden="true">—</span>
    </template>

    <div class="relative inline-flex shrink-0">
      <button
        type="button"
        data-alias="workspace-machine-summary-info"
        data-testid="workspace-machine-summary-info"
        class="inline-flex h-4 w-4 items-center justify-center rounded-full border border-gray-400 text-[10px] font-bold leading-none text-gray-500 hover:border-primary hover:text-primary focus:outline-none focus-visible:ring-1 focus-visible:ring-primary cursor-help"
        :aria-label="infoAriaLabel"
        :aria-expanded="detailTipOpen ? 'true' : 'false'"
        @pointerdown="onInfoPointerDown"
        @mouseenter="openFromHover"
        @mouseleave="closeFromHover"
        @focus="openFromFocus"
        @blur="closeFromBlur"
        @click="toggleDetailTip"
      >
        !
      </button>
      <span
        v-if="detailTipOpen"
        role="tooltip"
        data-testid="workspace-machine-summary-tooltip"
        class="pointer-events-none absolute left-1/2 top-full z-30 mt-1.5 w-max max-w-xs -translate-x-1/2 whitespace-pre-line rounded bg-gray-900 px-2.5 py-2 text-[11px] leading-relaxed text-white shadow-md"
      >
        {{ detailTooltipText }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  RUNTIME_MACHINE_TIP,
  RUNTIME_CONTAINER_TIP,
} from '../utils/workPanelRuntimeIndicators.js'

const props = defineProps({
  /** @type {import('vue').PropType<import('../utils/workPanelMachineSummary.js').WorkspaceMachineSummary | null>} */
  machineSummary: {
    type: Object,
    default: null,
  },
  machineRuntimeFilter: {
    type: String,
    default: null,
  },
})

const emit = defineEmits(['machine-filter'])

const rootEl = ref(null)
const detailTipOpen = ref(false)
// openMode 记录当前由谁展开（hover / focus / click），使 hover 离开、blur 只关闭
// 自己打开的提示，且 hover/focus 展开后点击可「锁定」为 click 模式不随鼠标离开消失。
const openMode = ref(null)
const OPEN = { HOVER: 'hover', FOCUS: 'focus', CLICK: 'click' }

// 移动端在 tap 前会派发兼容性 mouseenter/focus（先开再被 click 关掉），
// 用 touch pointerdown 置位抑制它们，只让随后的 click 决定开关。
const suppressSyntheticHoverFocus = ref(false)

function onInfoPointerDown(e) {
  suppressSyntheticHoverFocus.value = e.pointerType === 'touch'
}

function openDetail(mode) {
  detailTipOpen.value = true
  openMode.value = mode
}

function closeDetail() {
  detailTipOpen.value = false
  openMode.value = null
}

function openFromHover() {
  if (suppressSyntheticHoverFocus.value) return
  if (openMode.value === OPEN.CLICK) return // click 锁定时 hover 不关闭也不改模式
  openDetail(OPEN.HOVER)
}

function closeFromHover() {
  if (openMode.value === OPEN.HOVER) closeDetail()
}

function openFromFocus() {
  if (suppressSyntheticHoverFocus.value) return
  if (openMode.value === OPEN.CLICK) return
  openDetail(OPEN.FOCUS)
}

function closeFromBlur() {
  if (openMode.value === OPEN.FOCUS) closeDetail()
}

// click/tap 切换：首次点击打开并锁定（触屏用户得以保持可见），
// 再次点击（或点击外部 / Esc）关闭。
function toggleDetailTip() {
  suppressSyntheticHoverFocus.value = false
  if (openMode.value === OPEN.CLICK) {
    closeDetail()
  } else {
    openDetail(OPEN.CLICK)
  }
}

function onDocumentPointerDown(e) {
  if (!detailTipOpen.value) return
  if (rootEl.value && rootEl.value.contains(e.target)) return
  closeDetail()
}

function onDocumentKeyDown(e) {
  if (e.key === 'Escape' && detailTipOpen.value) closeDetail()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  document.addEventListener('keydown', onDocumentKeyDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  document.removeEventListener('keydown', onDocumentKeyDown)
})

const recycleLabel = computed(() => {
  const summary = props.machineSummary
  if (!summary) {
    return ''
  }
  return summary.idleRecycleMinutes <= 0
    ? '闲置回收：已关闭'
    : `闲置回收：${summary.idleRecycleMinutes} 分钟`
})

const detailTooltipText = computed(() => {
  const lines = ['机器节点']
  lines.push(`○ 空心环：已启动机器（${RUNTIME_MACHINE_TIP}）`)
  lines.push(`● 实心点：已启动容器（${RUNTIME_CONTAINER_TIP}）`)
  lines.push('空心=已启动机器 · 实心=已启动容器')

  const summary = props.machineSummary
  if (summary) {
    lines.push(`已启动 ${summary.startedCount} · 闲置 ${summary.idleCount}`)
    if (summary.startingCount > 0) {
      lines.push(`启动中 ${summary.startingCount}`)
    }
    lines.push(recycleLabel.value)
  } else {
    lines.push('当前暂无机器节点摘要数据')
  }
  return lines.join('\n')
})

const infoAriaLabel = computed(() =>
  props.machineSummary ? '查看机器节点说明与回收策略' : '查看机器节点图例说明',
)
</script>
