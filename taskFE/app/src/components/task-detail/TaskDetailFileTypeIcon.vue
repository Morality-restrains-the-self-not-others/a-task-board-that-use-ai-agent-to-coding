<template>
  <span
    class="inline-flex shrink-0 items-center justify-center text-gray-500"
    :title="label"
    data-testid="file-type-icon"
    :data-file-type="category"
    aria-hidden="true"
  >
    <!-- 简易线条图标：按类型换 path，避免引入图标库 -->
    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path
        v-if="category === 'folder'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
      />
      <path
        v-else-if="category === 'image'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
      />
      <path
        v-else-if="category === 'archive'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
      />
      <path
        v-else-if="category === 'code' || category === 'markup'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
      />
      <path
        v-else-if="category === 'binary'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
      />
      <path
        v-else
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M7 3h7l5 5v11a2 2 0 01-2 2H7a2 2 0 01-2-2V5a2 2 0 012-2z"
      />
    </svg>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { fileTypeFromPath, fileTypeLabel } from '../../utils/fileTypeFromPath.js'

const props = defineProps({
  /** 文件相对路径或文件名 */
  path: { type: String, default: '' },
  /** 覆盖分类（如目录传 folder） */
  type: { type: String, default: '' },
})

const category = computed(() => {
  const t = String(props.type || '').trim()
  if (t) return t
  return fileTypeFromPath(props.path)
})

const label = computed(() => {
  if (category.value === 'folder') return '目录'
  return fileTypeLabel(props.path)
})
</script>
