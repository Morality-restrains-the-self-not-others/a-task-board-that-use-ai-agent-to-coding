<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <!-- OPT-20260726-023: 移动端响应式 — 小屏时侧边栏折叠，主内容区全宽 -->
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="access-tokens" class="w-full lg:w-auto shrink-0" />

      <main class="flex-1 max-w-3xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">访问令牌</h1>
          <p class="text-sm text-text-light">访问令牌可用于替代密码登录或 API 认证。创建后仅完整展示一次，请妥善保管。</p>
        </div>

        <AccessTokenManagementPanel />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AccessTokenManagementPanel from '../components/AccessTokenManagementPanel.vue'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))
</script>
