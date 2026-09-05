<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999 p-4"
    data-testid="batch-delete-projects-modal"
    @click="emit('close')"
  >
    <div
      class="bg-white rounded-xl shadow-xl w-full max-w-md max-h-[85vh] flex flex-col z-10000"
      @click.stop
      @keydown.enter="emit('confirm')"
    >
      <div class="p-6 border-b border-gray-100">
        <h3 class="text-xl font-bold text-gray-900">确认批量删除</h3>
        <p class="mt-2 text-sm text-text-light">
          确定删除以下 <strong class="text-text">{{ projectNames.length }}</strong> 个项目吗？此操作不可恢复。
        </p>
      </div>

      <div class="flex-1 overflow-y-auto p-6 space-y-3">
        <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert" :data-traceId="errorTraceId || undefined">
          {{ error }}
        </div>

        <ul class="list-disc list-inside text-sm text-text space-y-1">
          <li v-for="name in previewNames" :key="name">{{ name }}</li>
        </ul>
        <p v-if="projectNames.length > previewNames.length" class="text-xs text-text-light">
          等 {{ projectNames.length }} 个项目
        </p>

        <div
          v-if="result?.errors?.length"
          class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900"
        >
          {{ result.errors.length }} 个项目删除失败
          <ul class="mt-2 list-disc list-inside text-xs">
            <li v-for="item in result.errors" :key="item.id">{{ item.id }}：{{ item.error }}</li>
          </ul>
        </div>
      </div>

      <div class="flex justify-end gap-3 p-6 border-t border-gray-100">
        <button
          type="button"
          class="btn-secondary px-4 py-2 text-sm"
          :disabled="deleting"
          @click="emit('close')"
        >
          取消
        </button>
        <button
          type="button"
          class="px-4 py-2 text-sm font-medium rounded-lg border border-red-200 text-red-600 bg-white hover:bg-red-50 disabled:opacity-50"
          :disabled="deleting"
          data-testid="batch-delete-confirm-btn"
          @click="emit('confirm')"
        >
          <span v-if="deleting">删除中…</span>
          <span v-else>确认删除</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  projectNames: { type: Array, default: () => [] },
  deleting: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
  result: { type: Object, default: null },
})

const emit = defineEmits(['close', 'confirm'])

const previewNames = computed(() => props.projectNames.slice(0, 5))
</script>
