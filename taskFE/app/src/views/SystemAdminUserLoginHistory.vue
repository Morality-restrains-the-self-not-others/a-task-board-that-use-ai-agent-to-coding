<template>
  <div class="p-6" data-testid="system-admin-user-login-history">
    <div class="mb-6">
      <!-- Anti-Replay-OK: real href back to user list -->
      <a href="/system-admin/users/" class="text-sm text-primary hover:text-primary/90">← 返回用户列表</a>
      <h2 class="text-xl font-semibold text-gray-900 mt-3">登录历史</h2>
      <p class="text-sm text-gray-500 mt-1">用户 ID：{{ userId }}</p>
    </div>
    <LoginHistoryPanel v-if="userId" :api-url="apiUrl" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import LoginHistoryPanel from '../components/LoginHistoryPanel.vue'

const route = useRoute()
const userId = computed(() => String(route.params.userId || '').trim())
const apiUrl = computed(() => `/api/system-admin/users/${encodeURIComponent(userId.value)}/login-history/`)
</script>
