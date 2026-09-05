<template>
  <div>
    <TaskDetailExecLogLoading v-if="loading" />
    <TaskDetailExecLogError v-if="!loading && topError" :message="topError" :trace-id="topErrorTraceId" />
    <template v-if="!topError || loading">
      <TaskDetailExecCloneLogSection
        :layer-id="targets.layerId"
        :clone-log-fetch-error="cloneLogFetchError"
        :clone-log-fetch-error-trace-id="cloneLogFetchErrorTraceId"
        :clone-log-text="cloneLogText"
      />
      <TaskDetailExecJobSection
        :job-id="targets.jobId"
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
      <TaskDetailExecLogEmptyTarget v-if="!loading && !targets.layerId && !targets.jobId" />
    </template>
  </div>
</template>

<script setup>
import TaskDetailExecCloneLogSection from './TaskDetailExecCloneLogSection.vue'
import TaskDetailExecJobSection from './TaskDetailExecJobSection.vue'
import TaskDetailExecLogLoading from './TaskDetailExecLogLoading.vue'
import TaskDetailExecLogError from './TaskDetailExecLogError.vue'
import TaskDetailExecLogEmptyTarget from './TaskDetailExecLogEmptyTarget.vue'

defineProps({
  loading: { type: Boolean, default: false },
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
})

const emit = defineEmits(['copy-agent-step-json', 'rich-interact'])

function onCopyAgentStepJson(step, stepIdx) {
  emit('copy-agent-step-json', step, stepIdx)
}

function onRichInteract(payload) {
  emit('rich-interact', payload)
}
</script>
