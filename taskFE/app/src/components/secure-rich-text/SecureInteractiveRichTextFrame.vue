<template>
  <div
    class="secure-rich-frame rounded border border-indigo-100 bg-white overflow-hidden max-h-48"
    data-testid="secure-rich-frame-host"
  >
    <iframe
      ref="frameRef"
      class="w-full border-0 block min-h-[4rem] h-48 max-h-48"
      :title="title"
      sandbox="allow-same-origin allow-scripts"
      data-testid="agent-step-rich-iframe"
      @load="onFrameLoad"
    />
  </div>
</template>

<script setup>
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { purifyStepInteractiveHtml, buildStepIframeSrcDoc } from '../../utils/secureStepRichText.js'
import { setIframeSrcdocSticky } from '../../utils/stickyIframeSrcdoc.js'

const props = defineProps({
  html: { type: String, default: '' },
  title: { type: String, default: '安全富文本步骤' },
})

const emit = defineEmits(['interactive'])

const frameRef = ref(null)
let onWindowMessage = null

const srcDoc = computed(() => {
  const clean = purifyStepInteractiveHtml(props.html)
  return buildStepIframeSrcDoc(clean)
})

function onMessage(ev) {
  if (ev.origin !== window.location.origin) return
  const d = ev.data
  if (!d || d.__secureStepRich !== true) return
  const win = frameRef.value?.contentWindow
  if (!win || ev.source !== win) return
  emit('interactive', {
    kind: d.kind,
    value: d.value,
    group: d.group,
    label: d.label,
    selected: d.selected,
    action: d.action,
    param: d.param,
  })
}

function onFrameLoad() {
  /* sticky restore 已在 setIframeSrcdocSticky 的 load 回调中处理 */
}

function writeSrcDocSticky() {
  const fr = frameRef.value
  if (!fr) return
  setIframeSrcdocSticky(fr, srcDoc.value)
}

watch(
  srcDoc,
  () => {
    nextTick(() => writeSrcDocSticky())
  },
  { flush: 'post' },
)

onMounted(() => {
  window.addEventListener(
    'message',
    (onWindowMessage = (ev) => {
      if (!ev || ev.origin !== window.location.origin) return
      onMessage(ev)
    }),
  )
  writeSrcDocSticky()
})

onBeforeUnmount(() => {
  if (onWindowMessage) window.removeEventListener('message', onWindowMessage)
})

defineExpose({ writeSrcDocSticky })
</script>
