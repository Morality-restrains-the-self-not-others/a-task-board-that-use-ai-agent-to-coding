<template>
  <div data-testid="project-tags-input">
    <label v-if="label" :for="inputId" class="block text-sm font-medium text-gray-700 mb-1">
      {{ label }}
    </label>
    <p v-if="hint" class="text-xs text-gray-500 mb-2">{{ hint }}</p>
    <div
      class="flex flex-wrap gap-2 items-center min-h-[42px] w-full border border-gray-300 rounded-md px-3 py-2 focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-transparent"
    >
      <span
        v-for="tag in modelValue"
        :key="tag"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100"
      >
        {{ tag }}
        <button
          type="button"
          class="text-blue-500 hover:text-blue-800 leading-none"
          :aria-label="`移除标签 ${tag}`"
          @click="removeTag(tag)"
        >
          ×
        </button>
      </span>
      <input
        :id="inputId"
        v-model="draft"
        type="text"
        class="flex-1 min-w-[8rem] border-0 p-0 text-sm focus:outline-none focus:ring-0"
        :placeholder="placeholder"
        :maxlength="PROJECT_TAG_MAX_LENGTH"
        @keydown.enter.prevent="commitDraft"
        @keydown="onKeydown"
        @blur="commitDraft"
      />
    </div>
    <p v-if="atMax" class="mt-1 text-xs text-amber-700">最多 {{ PROJECT_TAGS_MAX_COUNT }} 个标签</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import {
  PROJECT_TAG_MAX_LENGTH,
  PROJECT_TAGS_MAX_COUNT,
  normalizeProjectTags,
  parseProjectTagInput,
} from '../utils/projectTagsUtils.js'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  label: { type: String, default: '项目标签' },
  hint: { type: String, default: '按 Enter 或逗号添加标签，用于分类与检索' },
  placeholder: { type: String, default: '输入标签后按 Enter' },
  inputId: { type: String, default: 'project-tags' },
})

const emit = defineEmits(['update:modelValue'])

const draft = ref('')

const atMax = computed(() => (props.modelValue?.length || 0) >= PROJECT_TAGS_MAX_COUNT)

const emitTags = (next) => {
  emit('update:modelValue', normalizeProjectTags(next))
}

const commitDraft = () => {
  const tag = parseProjectTagInput(draft.value)
  draft.value = ''
  if (!tag || atMax.value) return
  const current = Array.isArray(props.modelValue) ? props.modelValue : []
  if (current.some((t) => String(t).toLowerCase() === tag.toLowerCase())) return
  emitTags([...current, tag])
}

const onKeydown = (event) => {
  if (event.key === ',') {
    event.preventDefault()
    commitDraft()
  }
}

const removeTag = (tag) => {
  const current = Array.isArray(props.modelValue) ? props.modelValue : []
  emitTags(current.filter((t) => t !== tag))
}

defineExpose({ commitDraft, normalizeProjectTags })
</script>
