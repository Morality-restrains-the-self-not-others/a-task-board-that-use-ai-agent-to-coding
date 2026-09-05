<template>
  <div>
    <div
      v-show="!!jobOutputDisplay"
      ref="viewportEl"
      data-testid="layer-job-output-scroll"
      class="max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs bg-white border border-gray-200 rounded p-2 text-gray-800 mb-1"
    >
      <div ref="topPadEl" aria-hidden="true" data-testid="virtual-log-top-pad" />
      <pre
        ref="bodyEl"
        class="m-0 p-0 border-0 bg-transparent whitespace-pre-wrap break-words font-inherit text-inherit"
        data-testid="virtual-log-body"
      ></pre>
      <div ref="bottomPadEl" aria-hidden="true" data-testid="virtual-log-bottom-pad" />
    </div>
    <p v-show="!jobOutputDisplay && liveOutputDisplay" class="text-xs text-gray-400 mb-1">
      控制台输出已在实时输出中展示
    </p>
    <p v-show="!jobOutputDisplay && !liveOutputDisplay" class="text-xs text-gray-400 mb-1">
      暂无控制台输出
    </p>
  </div>
</template>

<script setup>
import { ref, toRef } from 'vue'
import { useVirtualStickyLogScroll } from '../../composables/useVirtualStickyLogScroll.js'

const props = defineProps({
  liveOutputDisplay: { type: String, default: '' },
  jobOutputDisplay: { type: String, default: '' },
})

const viewportEl = ref(null)
const bodyEl = ref(null)
const topPadEl = ref(null)
const bottomPadEl = ref(null)
useVirtualStickyLogScroll(viewportEl, bodyEl, toRef(props, 'jobOutputDisplay'), {
  topPadRef: topPadEl,
  bottomPadRef: bottomPadEl,
})
</script>
