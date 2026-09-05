<template>
  <details
    class="group/agent-step rounded-lg border border-gray-200 bg-gray-50/90 px-2 pt-2 pb-1 text-xs shadow-sm"
    data-testid="layer-agent-step-card"
    :open="open"
    @toggle="onToggle"
  >
    <TaskDetailAgentStepCardHeader
      :title="card.title"
      :model-badge="card.modelBadge"
      :usage-badge="card.usageBadge"
      :is-copied="isCopied"
      :plain-subtitle="card.plainSubtitle || ''"
      @copy-json="emit('copy-json', card.rawStep, card.stepIdx)"
    />
    <div class="mt-1.5 pl-4" data-testid="layer-agent-step-card-body">
      <TaskDetailAgentStepCardDetails
        :body-mode="card.bodyMode || 'text'"
        :rich-html="card.richHtml || ''"
        :markdown-source="card.markdownSource || ''"
        :tool-result-text="card.toolResultText"
        :timestamp="card.timestamp"
        :error="card.error"
        :json-pretty="card.jsonPretty"
        @rich-interact="emit('rich-interact', $event)"
      />
    </div>
  </details>
</template>

<script setup>
import { computed } from 'vue'
import TaskDetailAgentStepCardHeader from './TaskDetailAgentStepCardHeader.vue'
import TaskDetailAgentStepCardDetails from './TaskDetailAgentStepCardDetails.vue'

const props = defineProps({
  card: {
    type: Object,
    required: true,
  },
  copyFeedbackKey: {
    type: String,
    default: '',
  },
  open: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['copy-json', 'rich-interact', 'toggle'])

const isCopied = computed(() => props.copyFeedbackKey === props.card?.key)

function onToggle(event) {
  const el = event?.target
  if (!el || el.tagName !== 'DETAILS') return
  emit('toggle', { key: props.card.key, open: Boolean(el.open) })
}
</script>
