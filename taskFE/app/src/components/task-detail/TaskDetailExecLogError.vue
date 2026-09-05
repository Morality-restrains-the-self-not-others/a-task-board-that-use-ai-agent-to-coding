<template>
  <p
    class="text-xs mb-2"
    :class="pending ? 'text-amber-700' : 'text-red-600'"
    :data-testid="pending ? 'exec-log-waiting' : 'exec-log-error'"
    v-bind="traceId ? { 'data-traceId': traceId } : {}"
  >{{ message }}</p>
</template>

<script setup>
import { computed } from 'vue'
import { isContainerComputeWaitingMessage } from '../../utils/containerComputeWaiting.js'

const props = defineProps({
  message: { type: String, default: '' },
  traceId: { type: String, default: '' },
})

const pending = computed(() => isContainerComputeWaitingMessage(props.message))
</script>
