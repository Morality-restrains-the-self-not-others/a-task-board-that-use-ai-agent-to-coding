<template>
  <div>
    <TaskDetailExecConsoleOutput
      :live-output-display="liveOutputDisplay"
      :job-output-display="jobOutputDisplay"
    />
  </div>
</template>

<script setup>
import TaskDetailExecConsoleOutput from './TaskDetailExecConsoleOutput.vue'

defineProps({
  liveOutputDisplay: { type: String, default: '' },
  jobOutputDisplay: { type: String, default: '' },
  jobExecutionPayload: { type: Object, default: null },
})
</script>
