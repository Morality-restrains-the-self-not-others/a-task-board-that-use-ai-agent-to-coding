<template>
  <div v-if="layerId && (cloneLogFetchError || cloneLogText)" class="mb-3">
    <TaskDetailExecLogError
      v-if="cloneLogFetchError"
      class="mb-1"
      :message="cloneLogFetchError"
      :trace-id="cloneLogFetchErrorTraceId"
    />
    <div
      v-else-if="cloneLogText"
      ref="viewportEl"
      data-testid="layer-clone-log-scroll"
      class="max-h-40 overflow-auto whitespace-pre-wrap break-words text-xs bg-white border border-gray-200 rounded p-2 text-gray-800"
    >
      <div ref="topPadEl" aria-hidden="true" data-testid="virtual-log-top-pad" />
      <pre
        ref="bodyEl"
        class="m-0 p-0 border-0 bg-transparent whitespace-pre-wrap break-words font-inherit text-inherit"
        data-testid="virtual-log-body"
      ></pre>
      <div ref="bottomPadEl" aria-hidden="true" data-testid="virtual-log-bottom-pad" />
    </div>
  </div>
</template>

<script setup>
import { ref, toRef } from 'vue'
import { useVirtualStickyLogScroll } from '../../composables/useVirtualStickyLogScroll.js'
import TaskDetailExecLogError from './TaskDetailExecLogError.vue'

const props = defineProps({
  layerId: { type: String, default: '' },
  cloneLogFetchError: { type: String, default: '' },
  cloneLogFetchErrorTraceId: { type: String, default: '' },
  cloneLogText: { type: String, default: '' },
})

const viewportEl = ref(null)
const bodyEl = ref(null)
const topPadEl = ref(null)
const bottomPadEl = ref(null)
useVirtualStickyLogScroll(viewportEl, bodyEl, toRef(props, 'cloneLogText'), {
  topPadRef: topPadEl,
  bottomPadRef: bottomPadEl,
})
</script>
