<template>
  <div>
    <TaskDetailExecLogError
      v-if="jobLogFetchError"
      class="mb-1"
      :message="jobLogFetchError"
      :trace-id="jobLogFetchErrorTraceId"
    />
    <template v-else-if="jobExecutionPayload?.job">
      <TaskDetailExecJobHeader
        :job="jobExecutionPayload.job"
        :job-command-head="jobCommandHead"
        :steps-note="jobExecutionPayload.steps?.note || ''"
      />
      <TaskDetailExecJobOutput
        :live-output-display="liveOutputDisplay"
        :job-output-display="jobOutputDisplay"
        :job-execution-payload="jobExecutionPayload"
      />
      <TaskDetailAgentStepsSection
        :copy-feedback-key="agentStepCopyFeedbackKey"
        :job-status="jobExecutionPayload?.job?.status || ''"
        :cards="agentStepCards"
        @copy-agent-step-json="onCopyAgentStepJson"
        @rich-interact="onRichInteract"
      />
    </template>
    <p v-else class="text-xs text-gray-400">暂无任务日志</p>
  </div>
</template>

<script setup>
import TaskDetailAgentStepsSection from './TaskDetailAgentStepsSection.vue'
import TaskDetailExecJobHeader from './TaskDetailExecJobHeader.vue'
import TaskDetailExecJobOutput from './TaskDetailExecJobOutput.vue'
import TaskDetailExecLogError from './TaskDetailExecLogError.vue'

defineProps({
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
