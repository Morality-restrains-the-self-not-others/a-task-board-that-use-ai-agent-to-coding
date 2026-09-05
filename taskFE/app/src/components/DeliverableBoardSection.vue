<template>
  <section
    class="deliverable-board-section"
    :class="{ 'deliverable-board-section--fill': fillRemaining && expanded }"
    :data-alias="dataAlias"
    :data-bar-id="barId || undefined"
  >
    <button
      type="button"
      class="deliverable-section-toggle w-full flex items-center gap-2 text-left px-1 py-1.5 rounded hover:bg-gray-50 transition-colors shrink-0"
      :aria-expanded="expanded ? 'true' : 'false'"
      data-alias="deliverable-section-toggle"
      @click="$emit('toggle')"
    >
      <svg
        class="w-4 h-4 text-gray-400 shrink-0 block transition-transform duration-200"
        :class="expanded ? 'rotate-90' : 'rotate-0'"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
      </svg>
      <h3
        class="text-sm font-semibold leading-none m-0 flex-1 min-w-0 truncate"
        :class="titleClass"
      >
        {{ title }}
      </h3>
      <span
        v-if="normalizedColumnCounts.length"
        class="flex items-center gap-2 shrink-0"
        data-alias="deliverable-section-count"
        :aria-label="columnCountsAriaLabel"
      >
        <span
          v-for="col in normalizedColumnCounts"
          :key="String(col.id)"
          class="inline-flex items-center gap-1 text-xs text-gray-400 tabular-nums"
          data-alias="deliverable-section-column-count"
          :data-progress-column-id="String(col.id)"
          :title="col.name"
        >
          <span
            class="inline-block w-1.5 h-1.5 rounded-full shrink-0"
            :style="{ backgroundColor: col.color }"
            aria-hidden="true"
          />
          {{ col.count }}
        </span>
      </span>
    </button>
    <div
      v-show="expanded"
      class="deliverable-section-body mt-1"
      :class="{ 'flex-1 min-h-0 flex flex-col': fillRemaining }"
      data-alias="deliverable-section-body"
    >
      <slot />
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  titleClass: { type: String, default: 'text-gray-800' },
  expanded: { type: Boolean, default: true },
  /** 为 true 时撑满父级剩余高度（工作面板「其他」看板贴底） */
  fillRemaining: { type: Boolean, default: false },
  /** @type {{ id: *, name?: string, color?: string, count: number }[]} */
  columnCounts: { type: Array, default: () => [] },
  dataAlias: { type: String, default: 'deliverable-section' },
  barId: { type: String, default: '' },
})

defineEmits(['toggle'])

const normalizedColumnCounts = computed(() =>
  (Array.isArray(props.columnCounts) ? props.columnCounts : []).map((col, index) => ({
    id: col?.id ?? index,
    name: col?.name || '',
    color: col?.color || '#94a3b8',
    count: Number(col?.count) || 0,
  })),
)

const columnCountsAriaLabel = computed(() =>
  normalizedColumnCounts.value
    .map((col) => (col.name ? `${col.name} ${col.count}` : String(col.count)))
    .join('，'),
)
</script>

<style scoped>
.deliverable-board-section--fill {
  flex: 1 0 0%;
  min-height: 280px;
  display: flex;
  flex-direction: column;
}
</style>
