<template>
  <div v-if="totalPages > 1" class="mt-6 flex justify-center">
    <nav class="flex items-center space-x-1" aria-label="分页">
      <button
        type="button"
        :disabled="currentPage === 1"
        class="px-3 py-1 border rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        @click="$emit('change-page', currentPage - 1)"
      >
        上一页
      </button>
      <template v-for="(item, idx) in pageItems" :key="item === 'ellipsis' ? `e-${idx}` : `p-${item}`">
        <span
          v-if="item === 'ellipsis'"
          class="px-2 text-sm text-gray-400 select-none"
          aria-hidden="true"
        >…</span>
        <button
          v-else
          type="button"
          :aria-current="currentPage === item ? 'page' : undefined"
          :class="[
            'px-3 py-1 border rounded-md text-sm transition-colors',
            currentPage === item
              ? 'bg-primary text-white border-primary'
              : 'hover:bg-gray-50'
          ]"
          @click="$emit('change-page', item)"
        >
          {{ item }}
        </button>
      </template>
      <button
        type="button"
        :disabled="currentPage === totalPages"
        class="px-3 py-1 border rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        @click="$emit('change-page', currentPage + 1)"
      >
        下一页
      </button>
    </nav>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { buildPaginationItems } from '../utils/paginationPages.js'

const props = defineProps({
  currentPage: { type: Number, required: true },
  totalPages: { type: Number, required: true }
})

defineEmits(['change-page'])

const pageItems = computed(() =>
  buildPaginationItems(props.currentPage, props.totalPages)
)
</script>
