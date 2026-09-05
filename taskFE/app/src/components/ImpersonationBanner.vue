<template>
  <div
    v-if="impersonating"
    data-testid="impersonation-banner"
    class="bg-amber-100 border-b border-amber-300 text-amber-900 px-4 py-2 flex items-center justify-between gap-3 z-[60]"
  >
    <p class="text-sm">
      正在以 <span class="font-semibold">{{ targetUsername || '该用户' }}</span> 的身份查看
    </p>
    <button
      type="button"
      data-testid="stop-impersonation-btn"
      class="px-3 py-1.5 text-sm bg-amber-800 text-white rounded-md hover:bg-amber-900 disabled:opacity-60"
      :disabled="stopping"
      :aria-busy="stopping ? 'true' : 'false'"
      @click="stopImpersonation"
    >
      {{ stopping ? '正在退出...' : '退出模拟登录' }}
    </button>
  </div>
  <p
    v-if="error"
    class="px-4 py-1 text-sm text-red-700"
    :data-trace-id="errorTraceId || undefined"
  >{{ error }}</p>
</template>

<script setup>
/* @alias:cmp-impersonation-banner */
import { useImpersonationStatus } from '../composables/useImpersonationStatus.js'

const {
  impersonating,
  targetUsername,
  stopping,
  error,
  errorTraceId,
  stopImpersonation,
  bindMounted,
} = useImpersonationStatus()
bindMounted()
</script>
