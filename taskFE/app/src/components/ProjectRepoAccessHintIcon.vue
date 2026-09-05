<template>
  <div ref="rootEl" class="relative inline-flex items-center shrink-0" data-alias="project-repo-access-hint">
    <button
      type="button"
      class="inline-flex h-7 w-7 items-center justify-center rounded-full text-amber-600 hover:bg-amber-50 focus:outline-none focus:ring-2 focus:ring-amber-400 focus:ring-offset-1"
      :aria-expanded="open ? 'true' : 'false'"
      aria-label="项目仓库「不可访问」说明"
      @click.stop="toggle"
    >
      <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
    </button>
    <div
      v-show="open"
      role="tooltip"
      class="absolute left-0 top-full z-[10050] mt-1 w-64 rounded-md border border-gray-200 bg-white p-3 text-left text-sm leading-relaxed text-gray-700 shadow-lg"
      @click.stop
    >
      「不可访问」表示需要去对应的项目仓库详情页中获取授权。
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted, nextTick } from 'vue'

const props = defineProps({
  /** 为 true 时强制关闭（例如父级模态框点击遮罩关闭其它浮层） */
  dismissSignal: {
    type: Number,
    default: 0
  }
})

const open = ref(false)
const rootEl = ref(null)

const toggle = () => {
  open.value = !open.value
}

const onDocumentClick = (event) => {
  if (!open.value) return
  const el = rootEl.value
  if (el && !el.contains(event.target)) {
    open.value = false
  }
}

watch(open, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      document.addEventListener('click', onDocumentClick, true)
    })
  } else {
    document.removeEventListener('click', onDocumentClick, true)
  }
})

watch(
  () => props.dismissSignal,
  () => {
    open.value = false
  }
)

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick, true)
})

defineExpose({
  close: () => {
    open.value = false
  }
})
</script>
