<template>
  <div data-alias="view-system-admin-dashboard" class="p-8">
    <!-- 页面标题 -->
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-gray-900">系统管理</h1>
      <p class="text-gray-600 mt-2">欢迎使用系统管理控制台</p>
    </div>

    <!-- 统计卡片区域 -->
    <div v-if="loadingDashboard" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary"></div>
      <span class="ml-3 text-gray-500">加载统计数据中...</span>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
      <!-- 用户统计卡片 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-500">总用户数</p>
            <h3 class="text-2xl font-bold text-gray-900 mt-1">{{ dashboardStats.total_users ?? '—' }}</h3>
          </div>
          <div class="bg-primary/10 p-3 rounded-full">
            <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"></path>
            </svg>
          </div>
        </div>
      </div>

      <!-- 云平台授权统计卡片 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-500">云平台授权数</p>
            <h3 class="text-2xl font-bold text-gray-900 mt-1">{{ dashboardStats.cloud_authorizations ?? '—' }}</h3>
          </div>
          <div class="bg-primary/10 p-3 rounded-full">
            <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
            </svg>
          </div>
        </div>
      </div>

      <!-- 交付物系统统计卡片 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-500">交付物系统数</p>
            <h3 class="text-2xl font-bold text-gray-900 mt-1">{{ dashboardStats.deliverable_systems ?? '—' }}</h3>
          </div>
          <div class="bg-primary/10 p-3 rounded-full">
            <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
            </svg>
          </div>
        </div>
      </div>
    </div>

    <div v-if="dashboardError" class="mb-8 p-4 bg-red-50 text-red-700 rounded-lg text-sm" :data-traceId="dashboardErrorTraceId || undefined">
      {{ dashboardError }}
    </div>

    <!-- 快捷操作区域 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">快捷操作</h2>
        <div class="grid grid-cols-2 gap-4">
          <!-- 用户管理快捷入口 -->
          <a href="/system-admin/users/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">用户管理</span>
          </a>

          <!-- 浏览器插件白名单快捷入口（OPT-20260808-028） -->
          <a href="/system-admin/oidc-extension/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">浏览器插件</span>
          </a>

          <!-- 云平台授权管理快捷入口 -->
          <a href="/system-admin/cloud-authorizations/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">云平台授权</span>
          </a>

          <!-- 交付物系统管理快捷入口 -->
          <a href="/system-admin/deliverable-system/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">交付物系统</span>
          </a>
          <!-- 推荐码管理快捷入口 -->
          <a href="/system-admin/referral-management/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">推荐码管理</span>
          </a>
          <!-- 登录与支付策略快捷入口 -->
          <router-link to="/system-admin/login-payment-policy/" class="flex flex-col items-center p-4 bg-gray-50 rounded-lg hover:bg-primary/5 transition-colors">
            <div class="bg-primary/10 p-3 rounded-full mb-2">
              <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
              </svg>
            </div>
            <span class="text-sm font-medium text-gray-900">登录与支付策略</span>
          </router-link>
        </div>
      </div>

      <!-- 系统信息区域 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">系统信息</h2>
        <div class="space-y-4">
          <div class="flex justify-between items-center">
            <span class="text-sm text-gray-600">系统版本</span>
            <span class="text-sm font-medium text-gray-900">{{ systemInfo.version || 'v1.0.0' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-sm text-gray-600">最后更新时间</span>
            <span class="text-sm font-medium text-gray-900">{{ systemInfo.last_deploy_time || '—' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-sm text-gray-600">在线管理员</span>
            <span class="text-sm font-medium text-gray-900">{{ dashboardStats.online_admins || '1' }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-sm text-gray-600">系统状态</span>
            <span class="flex items-center text-sm font-medium" :class="systemInfo.status === 'running' ? 'text-green-600' : 'text-red-600'">
              <span class="w-2 h-2 rounded-full mr-2" :class="systemInfo.status === 'running' ? 'bg-green-500' : 'bg-red-500'"></span>
              {{ systemInfo.status === 'running' ? '正常运行' : systemInfo.status }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <SystemAdminRegistrationInvitePanel />

  </div>
</template>

<script setup>
/* @alias:view-system-admin-dashboard */
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import SystemAdminRegistrationInvitePanel from '../components/SystemAdminRegistrationInvitePanel.vue'

const DASHBOARD_API = '/api/system-admin/dashboard/'

const loadingDashboard = ref(false)
const dashboardError = ref('')
const dashboardErrorTraceId = ref('')

const dashboardStats = ref({
  total_users: null,
  cloud_authorizations: null,
  deliverable_systems: null,
  online_admins: null,
})

const systemInfo = ref({
  version: '',
  last_deploy_time: '',
  status: 'running',
})

const extractErrorMessage = (data, fallback) => {
  if (!data) return fallback
  if (typeof data.message === 'string' && data.message.trim()) return data.message.trim()
  if (typeof data.detail === 'string' && data.detail.trim()) return data.detail.trim()
  if (typeof data.error === 'string' && data.error.trim()) return data.error.trim()
  if (Array.isArray(data.non_field_errors) && data.non_field_errors.length) {
    return String(data.non_field_errors[0])
  }
  return fallback
}

const loadDashboardStats = async () => {
  loadingDashboard.value = true
  dashboardError.value = ''
  dashboardErrorTraceId.value = ''
  try {
    const response = await apiFetch(DASHBOARD_API, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const { data, traceId } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      // 网关 forward-auth 会话失效（无法解析登录凭据）由 apiFetch 统一引导重新登录，
      // 此处不再逐页接线（见 utils/apiUtils.js / requestErrorDisplay.js）。
      dashboardErrorTraceId.value = traceId
      dashboardError.value = extractErrorMessage(data, `数据加载失败（${response.status}）`)
      return
    }
    if (data.stats) {
      dashboardStats.value = {
        total_users: data.stats.total_users ?? null,
        cloud_authorizations: data.stats.cloud_authorizations ?? null,
        deliverable_systems: data.stats.deliverable_systems ?? null,
        online_admins: data.stats.online_admins ?? '1',
      }
    }
    if (data.system_info) {
      systemInfo.value = {
        version: data.system_info.version || 'v1.0.0',
        last_deploy_time: data.system_info.last_deploy_time || '',
        status: data.system_info.status || 'running',
      }
    }
  } catch (error) {
    dashboardErrorTraceId.value = error.traceId || ''
    dashboardError.value = error?.message || '数据加载失败'
  } finally {
    loadingDashboard.value = false
  }
}

onMounted(() => {
  loadDashboardStats()
})
</script>

<style scoped>
/* 组件内样式可以在这里添加 */
</style>
