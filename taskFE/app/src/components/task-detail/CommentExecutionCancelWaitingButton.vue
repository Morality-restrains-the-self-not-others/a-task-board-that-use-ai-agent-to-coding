<template>
  <button
    v-if="visible"
    type="button"
    class="shrink-0 px-1.5 py-0.5 text-[10px] rounded border border-rose-200 text-rose-800 bg-white hover:bg-rose-50 disabled:opacity-50 disabled:cursor-not-allowed"
    data-testid="comment-execution-cancel-waiting"
    :disabled="busy"
    :title="busy ? '终止中…' : '终止等待前序的执行'"
    @click.stop.prevent="onClick"
  >{{ busy ? '终止中…' : '终止' }}</button>
</template>

<script setup>
import modalService from '../../utils/modalService.js'

defineProps({
  visible: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  commentId: { type: String, default: '' },
})

const emit = defineEmits(['cancel-waiting'])

async function onClick() {
  try {
    await modalService.confirm(
      '确定终止该评论的等待前序执行吗？终止后将不再自动启动容器；依赖本评论的后续串行评论仍会继续等待。',
      '终止等待',
      '终止',
      '取消',
    )
  } catch {
    return
  }
  emit('cancel-waiting')
}
</script>
