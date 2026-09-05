<template>
  <div>
    <div class="flex justify-between items-start gap-3 mb-3">
      <h2 class="text-lg font-bold text-text leading-snug line-clamp-2">{{ project.name }}</h2>
      <span
        class="shrink-0 px-2.5 py-0.5 rounded-full text-xs font-medium"
        :class="statusBadgeClass"
      >
        {{ project.status || '进行中' }}
      </span>
    </div>

    <div
      v-if="displayTags.length"
      class="flex flex-wrap gap-1.5 mb-3"
      data-testid="project-card-tags"
    >
      <span
        v-for="tag in displayTags"
        :key="tag"
        class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100"
      >
        {{ tag }}
      </span>
      <span
        v-if="overflowCount > 0"
        class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600"
      >
        +{{ overflowCount }}
      </span>
    </div>

    <p class="text-sm text-text-light mb-4 line-clamp-3 flex-1 min-h-[3.75rem]">
      {{ project.description || '暂无描述' }}
    </p>

    <div v-if="project.container_image" class="bg-gray-50 p-3 rounded-lg mb-4 border border-gray-100">
      <div class="flex items-start text-sm text-text-light gap-2">
        <svg class="w-4 h-4 mt-0.5 shrink-0 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"></path>
        </svg>
        <span class="break-all leading-relaxed">镜像：{{ formatImageLabel(project.container_image) }}</span>
      </div>
    </div>

    <div class="flex justify-between items-center gap-3 mt-auto pt-2 border-t border-gray-50">
      <time class="text-xs text-text-light tabular-nums" :datetime="project.created_at || undefined">
        {{ formatDate(project.created_at) }}
      </time>
      <a
        v-if="showDetailLink"
        :href="tenantPath + '/projects/' + project.id + '/'"
        class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:gap-2 transition-all"
        @click.stop
      >
        详情
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </a>
      <span
        v-else
        class="inline-flex items-center gap-1 text-sm font-medium text-primary group-hover:gap-2 transition-all"
      >
        详情
        <svg class="w-4 h-4 transition-transform group-hover:translate-x-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { normalizeProjectTags } from '../utils/projectTagsUtils.js'

const TAGS_VISIBLE_MAX = 3

const props = defineProps({
  project: { type: Object, required: true },
  tenantPath: { type: String, default: '' },
  statusBadgeClass: { type: String, default: '' },
  formatDate: { type: Function, required: true },
  formatImageLabel: { type: Function, required: true },
  showDetailLink: { type: Boolean, default: false },
})

const normalizedTags = computed(() =>
  normalizeProjectTags(Array.isArray(props.project?.tags) ? props.project.tags : []),
)

const displayTags = computed(() => normalizedTags.value.slice(0, TAGS_VISIBLE_MAX))

const overflowCount = computed(() =>
  Math.max(0, normalizedTags.value.length - TAGS_VISIBLE_MAX),
)
</script>
