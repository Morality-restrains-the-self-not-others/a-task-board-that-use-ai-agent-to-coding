<template>
  <div v-if="jobId">
    <p class="text-xs text-gray-500 mb-1">任务执行</p>
    <TaskDetailExecLiveOutput :live-output-display="liveOutputDisplay" />
    <TaskDetailExecJobBody
      :live-output-display="liveOutputDisplay"
      :job-log-fetch-error="jobLogFetchError"
      :job-log-fetch-error-trace-id="jobLogFetchErrorTraceId"
      :job-execution-payload="jobExecutionPayload"
      :job-command-head="jobCommandHead"
      :job-output-display="jobOutputDisplay"
      :agent-step-copy-feedback-key="agentStepCopyFeedbackKey"
      :agent-step-cards="agentStepCards"
      @copy-agent-step-json="onCopyAgentStepJson"
      @rich-interact="onRichInteract"
    />
  </div>
</template>

<script setup>
import TaskDetailExecJobBody from './TaskDetailExecJobBody.vue'
import TaskDetailExecLiveOutput from './TaskDetailExecLiveOutput.vue'

defineProps({
  jobId: { type: String, default: '' },
  liveOutputDisplay: { type: String, default: '' },
  jobLogFetchError: { type: String, default: '' },
  jobLogFetchErrorTraceId: { type: String, default: '' },
  jobExecutionPayload: { type: Object, default: null },
  jobCommandHead: { type: String, default: '' },
  jobOutputDisplay: { type: String, default: '' },
  agentStepCopyFeedbackKey: { type: String, default: '' },
  agentStepCards: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['copy-agent-step-json', 'rich-interact'])

function onCopyAgentStepJson(step, stepIdx) {
  emit('copy-agent-step-json', step, stepIdx)
}

function onRichInteract(payload) {
  emit('rich-interact', payload)
}
</script>
