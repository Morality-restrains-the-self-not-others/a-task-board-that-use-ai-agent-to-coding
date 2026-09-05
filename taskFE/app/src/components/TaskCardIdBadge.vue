<template>
  <span
    v-if="displayId"
    role="button"
    tabindex="0"
    data-testid="task-card-id"
    :class="rootClass"
    :title="copied ? '已复制' : '点击复制 #序号 | 双击复制完整ID'"
    :aria-label="`任务编号 ${displayId}，点击复制序号，双击复制完整ID`"
    @click.stop="copyId(false)"
    @dblclick.stop="copyId(true)"
    @keydown.enter.prevent.stop="copyId(false)"
    @keydown.space.prevent.stop="copyId(false)"
  >{{ displayId }}</span>
</template>

<script setup>
import { computed, ref, onBeforeUnmount } from 'vue'
import toastService from '../utils/toastService'
import { formatTaskDisplayNo } from '../utils/taskIdDisplay.js'

const props = defineProps({
  taskId: {
    type: [String, Number],
    default: ''
  },
  workspaceSeq: {
    type: [Number, String],
    default: 0
  },
  /** 覆盖卡片默认字号/颜色（如任务详情辅助信息里展示大号 #N）；缺省保持看板卡片样式 */
  sizeClass: {
    type: String,
    default: ''
  },
})

const rootClass = computed(() =>
  ['task-card-no-drag font-mono tabular-nums shrink-0 mt-0.5 cursor-pointer hover:text-gray-600 select-none',
   props.sizeClass || 'text-xs text-gray-400'].join(' '),
)
const displayId = computed(() => formatTaskDisplayNo(props.workspaceSeq))
const copied = ref(false)
let copiedResetTimer = null

onBeforeUnmount(() => {
  if (copiedResetTimer) clearTimeout(copiedResetTimer)
})

const copyId = async (fullId) => {
  const tid = String(props.taskId || '')
  const full = tid.startsWith('task_') ? tid : (tid ? `task_${tid}` : '')
  const text = fullId ? full : displayId.value
  if (!text) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.setAttribute('readonly', '')
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      if (!ok) throw new Error('execCommand copy failed')
    }
    copied.value = true
    if (copiedResetTimer) clearTimeout(copiedResetTimer)
    copiedResetTimer = setTimeout(() => {
      copied.value = false
      copiedResetTimer = null
    }, 1500)
    toastService.success(fullId ? `已复制完整编号 ${text}` : `已复制编号 ${text}`, 1500)
  } catch {
    toastService.error('复制失败，请手动选择编号', 2000)
  }
}
</script>
