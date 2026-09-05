<template>
  <div
    v-if="target"
    class="app-modal-overlay z-50 flex items-center justify-center bg-black/30"
    data-testid="gitlab-region-delete-modal"
    @click.self="emit('cancel')"
  >
    <div class="bg-white rounded-xl shadow-lg p-6 max-w-md w-full space-y-4">
      <h4 class="text-lg font-semibold text-text">确认删除区域</h4>
      <p class="text-sm text-text-light">
        请确认要删除（停用）的仓库地址：
      </p>
      <p
        class="text-sm font-mono break-all bg-gray-50 border border-border rounded px-3 py-2 text-text"
        data-testid="gitlab-region-delete-repo-address"
      >
        {{ repoAddress }}
      </p>
      <p class="text-sm text-text-light">
        区域 <strong>{{ target.name }}</strong>
        <span v-if="target.slug">({{ target.slug }})</span>。
        此操作仅标记区域为停用，不会删除已有数据和租户资源。
      </p>
      <div class="flex gap-2 justify-end">
        <button
          type="button"
          class="px-4 py-2 border border-border rounded text-sm hover:bg-gray-50"
          data-testid="gitlab-region-delete-cancel"
          @click="emit('cancel')"
        >
          取消
        </button>
        <button
          type="button"
          class="px-4 py-2 bg-red-600 text-white rounded text-sm hover:bg-red-700 disabled:opacity-60"
          data-testid="gitlab-region-delete-confirm"
          :disabled="submitting"
          @click="emit('confirm', target.slug)"
        >
          {{ submitting ? '停用中...' : '确认停用' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { resolveGitlabRegionRepoAddress } from './gitlabRegionRepoAddress.js'

const props = defineProps({
  target: { type: Object, default: null },
  submitting: { type: Boolean, default: false },
})

const emit = defineEmits(['cancel', 'confirm'])

const repoAddress = computed(() => resolveGitlabRegionRepoAddress(props.target))
</script>
