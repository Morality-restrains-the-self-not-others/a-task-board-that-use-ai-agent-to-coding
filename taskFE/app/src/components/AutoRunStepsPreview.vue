<template>
  <div class="auto-run-steps-preview" data-testid="auto-run-steps-preview">
    <button
      type="button"
      class="text-xs font-medium text-primary hover:underline focus:outline-none"
      data-testid="auto-run-steps-toggle"
      @click="expanded = !expanded"
    >
      {{ expanded ? '收起自动运行说明' : '查看自动运行说明' }}
    </button>
    <div
      v-if="expanded"
      class="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-800 whitespace-pre-wrap"
      data-testid="auto-run-steps-body"
    >
      <p v-if="preview.emptyHint" class="text-gray-500" :class="{ 'text-red-600': preview.failed }">
        {{ preview.emptyHint }}
      </p>
      <pre v-else class="whitespace-pre-wrap font-sans text-sm leading-relaxed">{{ preview.markdown }}</pre>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { resolveAutoRunStepsPreview } from '../utils/autoRunStepsPreview.js'

const props = defineProps({
  markdown: { type: String, default: '' },
  extractStatus: { type: String, default: '' },
  liveMarkdown: { type: [String, null], default: null },
  liveError: { type: String, default: '' },
  defaultExpanded: { type: Boolean, default: false },
})

const expanded = ref(props.defaultExpanded)

const preview = computed(() =>
  resolveAutoRunStepsPreview({
    markdown: props.markdown,
    extractStatus: props.extractStatus,
    liveMarkdown: props.liveMarkdown,
    liveError: props.liveError,
  }),
)
</script>
