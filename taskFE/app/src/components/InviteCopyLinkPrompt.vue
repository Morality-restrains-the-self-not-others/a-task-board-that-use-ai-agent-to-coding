<template>
  <div
    class="bg-amber-50 border border-amber-200 text-amber-900 rounded-lg p-4 space-y-3"
    data-testid="invite-copy-link-prompt"
  >
    <p class="text-sm">{{ hint }}</p>
    <div v-if="url" class="flex">
      <input
        type="text"
        :value="url"
        readonly
        class="flex-1 px-4 py-2 border border-amber-200 rounded-l-lg bg-white text-sm"
        data-testid="invite-copy-link-input"
      >
      <button
        type="button"
        class="bg-primary text-white px-4 py-2 rounded-r-lg hover:bg-primary/90 text-sm"
        :disabled="!url"
        data-testid="invite-copy-link-btn"
        @click="copy"
      >
        {{ copied ? '已复制' : '复制' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import toastService from '../utils/toastService'

const props = defineProps({
  hint: { type: String, default: '该邮箱已退订邮件邀请，请手动复制邀请链接给对方' },
  url: { type: String, default: '' },
})

const copied = ref(false)

async function copy() {
  // Anti-Replay-OK: clipboard write only; no HTTP
  try {
    await navigator.clipboard.writeText(props.url)
    copied.value = true
    toastService.success('邀请链接已复制到剪贴板！', 2000)
  } catch (error) {
    console.error('复制失败:', error)
    toastService.error('复制失败，请手动复制链接')
  }
}
</script>
