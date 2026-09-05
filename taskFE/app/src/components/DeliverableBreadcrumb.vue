<template>
  <!--
    全部(+) › 类别
               [下拉] › …
    类别名与内容下拉上下排布，压缩水平空间
  -->
  <nav
    class="deliverable-filter-trail flex flex-wrap items-center gap-x-1.5 gap-y-1 px-4 py-1.5 text-sm"
    aria-label="交付物类别与内容过滤"
    data-alias="deliverable-breadcrumb"
  >
    <div class="deliverable-trail-root-row inline-flex items-center gap-1 shrink-0">
      <button
        type="button"
        class="deliverable-trail-root rounded px-1.5 py-0.5 transition-colors leading-6"
        :class="!hasContentFilter
          ? 'font-semibold text-gray-900 bg-gray-100'
          : 'text-indigo-600 hover:bg-indigo-50 hover:text-indigo-800'"
        data-segment-type="root"
        @click="$emit('clear')"
      >
        全部
      </button>
      <button
        type="button"
        class="deliverable-trail-add inline-flex items-center justify-center w-5 h-5 rounded border border-dashed border-gray-300 text-gray-500 hover:border-indigo-400 hover:text-indigo-600 hover:bg-indigo-50"
        title="新建交付物过滤栏"
        aria-label="新建交付物过滤栏"
        data-alias="deliverable-filter-add"
        @click="$emit('add-bar')"
      >
        <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
      </button>
      <button
        v-if="removable"
        type="button"
        class="deliverable-trail-remove text-xs text-gray-400 hover:text-red-500 px-0.5"
        title="删除此过滤栏"
        aria-label="删除此过滤栏"
        data-alias="deliverable-filter-remove"
        @click="$emit('remove-bar')"
      >
        删除
      </button>
    </div>

    <template v-for="level in levels" :key="level.category.id">
      <span class="text-gray-400 select-none shrink-0" aria-hidden="true">›</span>

      <div
        class="deliverable-trail-level inline-flex flex-col items-stretch gap-0.5 min-w-[7.5rem] w-56"
        :data-category-id="level.category.id"
        data-alias="deliverable-trail-level"
      >
        <span
          class="deliverable-trail-title inline-flex items-center gap-1 min-w-0 text-xs text-gray-700 font-medium leading-tight"
          :title="level.category.name"
        >
          <span
            class="inline-block w-2 h-2 rounded-full shrink-0"
            :style="{ backgroundColor: level.category.color }"
          ></span>
          <span class="truncate">{{ level.category.name }}</span>
        </span>
        <select
          class="deliverable-content-select w-full min-w-0 truncate rounded-md border border-gray-200 bg-white px-1.5 py-0.5 text-xs text-gray-700 focus:outline-none focus:ring-1 focus:ring-indigo-400 focus:border-indigo-400"
          :class="level.selectedContentId ? 'border-indigo-400 bg-indigo-50/50 text-indigo-900' : ''"
          data-alias="deliverable-content-select"
          :data-deliverable-column-id="level.category.id"
          :aria-label="`${level.category.name}内容过滤`"
          :disabled="level.contents.length === 0"
          :value="level.selectedContentId || ''"
          @change="onSelectChange(level, $event)"
        >
          <option value="">
            {{ level.contents.length === 0 ? '暂无内容' : '未选择' }}
          </option>
          <option
            v-for="content in level.contents"
            :key="content.id"
            :value="content.id"
          >
            {{ deliverableContentOptionLabel(content) }}
          </option>
        </select>
      </div>
    </template>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import {
  buildDeliverableFilterTrail,
  readFilterFromPath,
  rootDeliverablePath,
} from '../utils/workPanelDeliverableAggregation.js'
import { deliverableContentOptionLabel } from '../utils/workPanelDeliverableContent.js'

const props = defineProps({
  path: {
    type: Array,
    default: () => rootDeliverablePath(),
  },
  categories: {
    type: Array,
    default: () => [],
  },
  todos: {
    type: Array,
    default: () => [],
  },
  removable: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['clear', 'select-content', 'add-bar', 'remove-bar'])

const levels = computed(() =>
  buildDeliverableFilterTrail(props.categories, props.todos, props.path),
)

const hasContentFilter = computed(() => {
  const { taskId } = readFilterFromPath(props.path)
  return taskId != null && taskId !== ''
})

function onSelectChange(level, event) {
  const value = event?.target?.value != null ? String(event.target.value) : ''
  if (!value) {
    if (level.selectedContentId) {
      emit('clear')
    }
    return
  }
  const content = (level.contents || []).find((c) => String(c.id) === value)
  if (!content) return
  emit('select-content', { category: level.category, content })
}
</script>
