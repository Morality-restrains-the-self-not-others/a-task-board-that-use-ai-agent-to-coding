<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="login-history" class="w-full lg:w-auto shrink-0" />
      <main class="flex-1 max-w-4xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">登录历史</h1>
          <p class="text-sm text-text-light">每次成功登录会记录入口、时间和 IP。失败登录不会出现在这里。</p>
        </div>
        <LoginHistoryPanel :api-url="apiUrl" />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import LoginHistoryPanel from '../components/LoginHistoryPanel.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))
const apiUrl = '/api/auth/login-history/'
</script>
