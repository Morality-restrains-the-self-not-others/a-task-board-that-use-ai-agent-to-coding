<template>
  <div v-if="visible" class="mt-3">
    <AutoRunStepsPreview
      :markdown="image?.auto_run_steps_md || ''"
      :extract-status="image?.auto_run_steps_extract_status || ''"
      :default-expanded="defaultExpanded"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AutoRunStepsPreview from './AutoRunStepsPreview.vue'
import { shouldShowImageAutoRunSteps } from '../utils/autoRunStepsPreview.js'

const props = defineProps({
  image: { type: Object, default: null },
  defaultExpanded: { type: Boolean, default: false },
})

const visible = computed(() => shouldShowImageAutoRunSteps(props.image))
</script>
