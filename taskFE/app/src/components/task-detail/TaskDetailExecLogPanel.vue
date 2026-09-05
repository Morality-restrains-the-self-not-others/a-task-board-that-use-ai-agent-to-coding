<template>
  <div>
    <TaskDetailExecLogHeader
      :loading="loading"
      :copyable="copyable"
      :copy-feedback="copyFeedback"
      :clearable="clearable"
      @copy-log="emit('copy-log')"
      @clear-log="emit('clear-log')"
    />
    <TaskDetailExecLogContent
      :loading="loading"
      :top-error="topError"
      :top-error-trace-id="topErrorTraceId"
      :targets="targets"
      :clone-log-fetch-error="cloneLogFetchError"
      :clone-log-fetch-error-trace-id="cloneLogFetchErrorTraceId"
      :clone-log-text="cloneLogText"
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
import TaskDetailExecLogHeader from './TaskDetailExecLogHeader.vue'
import TaskDetailExecLogContent from './TaskDetailExecLogContent.vue'

defineProps({
  loading: { type: Boolean, default: false },
  copyable: { type: Boolean, default: false },
  copyFeedback: { type: Boolean, default: false },
  topError: { type: String, default: '' },
  topErrorTraceId: { type: String, default: '' },
  targets: {
    type: Object,
    default: () => ({ layerId: '', jobId: '' }),
  },
  cloneLogFetchError: { type: String, default: '' },
  cloneLogFetchErrorTraceId: { type: String, default: '' },
  cloneLogText: { type: String, default: '' },
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
  clearable: { type: Boolean, default: false },
})

const emit = defineEmits(['copy-log', 'clear-log', 'copy-agent-step-json', 'rich-interact'])

function onCopyAgentStepJson(step, stepIdx) {
  emit('copy-agent-step-json', step, stepIdx)
}

function onRichInteract(payload) {
  emit('rich-interact', payload)
}
</script>
