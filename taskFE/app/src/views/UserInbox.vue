<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="inbox" class="w-full lg:w-auto shrink-0" />
      <main class="flex-1 max-w-3xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">收信箱</h1>
          <p class="text-sm text-text-light">系统通知与管理员代登说明会出现在这里。</p>
        </div>
        <div v-if="loadError" class="bg-white rounded-xl border border-red-200 p-4 text-red-600" :data-trace-id="loadErrorTraceId || undefined">
          {{ loadError }}
        </div>
        <div v-else-if="loading" class="bg-white rounded-xl shadow-sm border border-border p-6 text-text-light">
          加载中...
        </div>
        <div v-else-if="messages.length === 0" class="bg-white rounded-xl shadow-sm border border-border p-6 text-text-light">
          暂无信件
        </div>
        <ul v-else class="space-y-4">
          <li
            v-for="msg in messages"
            :key="msg.id"
            class="bg-white rounded-xl shadow-sm border border-border p-5"
            data-testid="inbox-message"
          >
            <h2 class="text-lg font-semibold text-text">{{ msg.title }}</h2>
            <p class="text-xs text-text-light mt-1">{{ msg.created_at }}</p>
            <p class="text-sm text-text mt-3 whitespace-pre-wrap">{{ displayInboxBody(msg) }}</p>
            <p v-if="msg.reason" class="text-sm text-text mt-3" data-testid="inbox-message-reason">
              <span class="text-text-light">理由：</span>{{ msg.reason }}
            </p>
          </li>
        </ul>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import { apiFetch } from '../utils/apiUtils.js'
import { inboxBodyWithoutDuplicatedReason } from '../utils/inboxMessageDisplay.js'
import { messageFromFailedResponse } from '../utils/httpError.js'
import { extractTraceId } from '../utils/traceId.js'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))
const loading = ref(true)
const loadError = ref('')
const loadErrorTraceId = ref('')
const messages = ref([])

function displayInboxBody(msg) {
  return inboxBodyWithoutDuplicatedReason(msg.body, msg.reason)
}

onMounted(async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await apiFetch('/api/auth/inbox/', {
      method: 'GET',
      headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      credentials: 'include',
    })
    if (!response.ok) {
      loadError.value = messageFromFailedResponse(response, '加载收信箱失败')
      loadErrorTraceId.value = extractTraceId(response) || ''
      return
    }
    const data = await response.json().catch(() => ({}))
    messages.value = Array.isArray(data.results) ? data.results : []
  } catch (err) {
    loadError.value = err?.message || '加载收信箱失败'
  } finally {
    loading.value = false
  }
})
</script>
