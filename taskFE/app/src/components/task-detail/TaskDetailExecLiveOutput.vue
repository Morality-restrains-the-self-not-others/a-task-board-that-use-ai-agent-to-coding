<template>
  <div
    v-show="!!liveOutputDisplay"
    ref="viewportEl"
    data-testid="layer-live-output-scroll"
    class="max-h-32 overflow-auto whitespace-pre-wrap break-words text-xs bg-violet-50 border border-violet-200 rounded p-2 text-violet-900 mb-1"
  >
    <div ref="topPadEl" aria-hidden="true" data-testid="virtual-log-top-pad" />
    <pre
      ref="bodyEl"
      class="m-0 p-0 border-0 bg-transparent whitespace-pre-wrap break-words font-inherit text-inherit"
      data-testid="virtual-log-body"
    ></pre>
    <div ref="bottomPadEl" aria-hidden="true" data-testid="virtual-log-bottom-pad" />
  </div>
</template>

<script setup>
import { ref, toRef } from 'vue'
import { useVirtualStickyLogScroll } from '../../composables/useVirtualStickyLogScroll.js'

const props = defineProps({
  liveOutputDisplay: { type: String, default: '' },
})

const viewportEl = ref(null)
const bodyEl = ref(null)
const topPadEl = ref(null)
const bottomPadEl = ref(null)
useVirtualStickyLogScroll(viewportEl, bodyEl, toRef(props, 'liveOutputDisplay'), {
  topPadRef: topPadEl,
  bottomPadRef: bottomPadEl,
})
</script>
