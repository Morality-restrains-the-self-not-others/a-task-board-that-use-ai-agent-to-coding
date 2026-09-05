<template>
  <ImageAutoRunSteps
    v-if="image"
    class="mt-2 ml-6"
    data-testid="create-task-auto-run-steps"
    :image="image"
    :default-expanded="true"
  />
</template>

<script setup>
import { computed } from 'vue'
import ImageAutoRunSteps from './ImageAutoRunSteps.vue'
import { resolveSelectedInstalledImage } from '../utils/autoRunStepsPreview.js'

const props = defineProps({
  editingTask: { type: Object, default: null },
  installedImages: { type: Array, default: () => [] },
})

const image = computed(() =>
  resolveSelectedInstalledImage(props.editingTask, props.installedImages),
)
</script>
