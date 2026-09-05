<template>
  <!-- 任务详情等桌面场景固定左右分栏；勿再用 flex-col + md:flex-row（窄主栏/中等视口会退化成上下堆叠） -->
  <div
    ref="rootRef"
    class="flex flex-row items-stretch gap-0"
    data-testid="resizable-split-pane"
  >
    <div
      class="min-w-0 w-[var(--split-left-width)] shrink-0 max-w-[70%]"
      :style="leftPaneStyle"
      data-testid="resizable-split-pane-left"
    >
      <slot name="left" />
    </div>
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="拖动调整目录栏宽度"
      :aria-valuenow="Math.round(leftWidthPx)"
      :aria-valuemin="minWidth"
      :aria-valuemax="maxWidth"
      tabindex="0"
      class="flex w-1.5 shrink-0 cursor-col-resize items-stretch justify-center group select-none touch-none"
      :class="dragging ? 'bg-blue-200' : 'bg-transparent hover:bg-gray-200'"
      data-testid="resizable-split-pane-gutter"
      @pointerdown="onPointerDown"
      @dblclick="onDoubleClick"
      @keydown="onKeyDown"
    >
      <span
        class="w-px self-stretch my-1"
        :class="dragging ? 'bg-blue-500' : 'bg-gray-300 group-hover:bg-gray-500'"
        aria-hidden="true"
      />
    </div>
    <div class="min-w-0 flex-1" data-testid="resizable-split-pane-right">
      <slot name="right" />
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  DEFAULT_LEFT_WIDTH_PX,
  KEYBOARD_STEP_PX,
  MAX_LEFT_WIDTH_PX,
  MIN_LEFT_WIDTH_PX,
  clampLeftWidth,
  leftWidthFromPointer,
  readStoredLeftWidth,
  writeStoredLeftWidth,
} from '../composables/useResizableSplitPane.js'

const props = defineProps({
  /** localStorage 键；为空则不持久化 */
  storageKey: { type: String, default: '' },
  defaultWidth: { type: Number, default: DEFAULT_LEFT_WIDTH_PX },
  minWidth: { type: Number, default: MIN_LEFT_WIDTH_PX },
  maxWidth: { type: Number, default: MAX_LEFT_WIDTH_PX },
})

const rootRef = ref(null)
const leftWidthPx = ref(props.defaultWidth)
const dragging = ref(false)

const leftPaneStyle = computed(() => {
  const px = Math.round(leftWidthPx.value)
  return {
    '--split-left-width': `${px}px`,
    // 内联 width 兜底：避免仅依赖任意值 utility 被 purge 后左栏回到 w-full、把右栏挤成上下堆叠观感
    width: `${px}px`,
    maxWidth: '70%',
  }
})

function containerWidth() {
  const el = rootRef.value
  if (!el) return 0
  return el.getBoundingClientRect().width || 0
}

function applyWidth(next) {
  leftWidthPx.value = clampLeftWidth(next, containerWidth(), {
    min: props.minWidth,
    max: props.maxWidth,
  })
}

function persist() {
  writeStoredLeftWidth(props.storageKey, leftWidthPx.value, getStorage())
}

function getStorage() {
  try {
    return typeof localStorage !== 'undefined' ? localStorage : null
  } catch {
    return null
  }
}

function onPointerDown(e) {
  if (e.button != null && e.button !== 0) return
  const el = rootRef.value
  if (!el) return
  dragging.value = true
  e.preventDefault()
  const target = e.currentTarget
  if (target && typeof target.setPointerCapture === 'function' && e.pointerId != null) {
    try {
      target.setPointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
  }
  const prevUserSelect = document.body.style.userSelect
  document.body.style.userSelect = 'none'

  const onMove = (ev) => {
    const rect = el.getBoundingClientRect()
    applyWidth(leftWidthFromPointer(ev.clientX, rect.left, rect.width))
  }
  const onUp = () => {
    dragging.value = false
    document.body.style.userSelect = prevUserSelect
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    persist()
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
  onMove(e)
}

function onDoubleClick() {
  applyWidth(props.defaultWidth)
  persist()
}

function onKeyDown(e) {
  if (e.key === 'ArrowLeft') {
    e.preventDefault()
    applyWidth(leftWidthPx.value - KEYBOARD_STEP_PX)
    persist()
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    applyWidth(leftWidthPx.value + KEYBOARD_STEP_PX)
    persist()
  } else if (e.key === 'Home') {
    e.preventDefault()
    applyWidth(props.minWidth)
    persist()
  } else if (e.key === 'End') {
    e.preventDefault()
    applyWidth(props.maxWidth)
    persist()
  }
}

onMounted(() => {
  const stored = readStoredLeftWidth(props.storageKey, getStorage())
  applyWidth(stored != null ? stored : props.defaultWidth)
})

onBeforeUnmount(() => {
  document.body.style.userSelect = ''
})
</script>
