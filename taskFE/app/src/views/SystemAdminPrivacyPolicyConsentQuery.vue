<template>
  <div class="mb-8 rounded-lg border border-gray-200 bg-gray-50/80 p-4">
    <h3 class="text-sm font-medium text-gray-800 mb-3">用户同意记录查询</h3>
    <div class="flex flex-wrap items-end gap-3">
      <div>
        <label for="consent-user-id" class="block text-xs text-gray-500 mb-1">用户 ID</label>
        <input
          id="consent-user-id"
          v-model="consentUserId"
          type="text"
          class="w-56 px-3 py-2 rounded-lg border border-gray-300 text-sm"
          placeholder="数字 ID"
          @keydown.enter="fetchUserConsents"
        >
      </div>
      <button
        type="button"
        class="px-4 py-2 bg-gray-800 text-white text-sm rounded-lg hover:bg-gray-700 disabled:opacity-50"
        :disabled="consentLoading"
        @click="fetchUserConsents"
      >
        {{ consentLoading ? '查询中...' : '查询' }}
      </button>
    </div>
    <p v-if="consentError" class="text-sm text-amber-800 mt-2" :data-traceId="consentErrorTraceId">{{ consentError }}</p>
    <div v-if="consentRows.length" class="mt-4 overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead>
          <tr class="text-left text-gray-500 border-b">
            <th class="pr-3 py-2">同意时间</th>
            <th class="pr-3 py-2">场景</th>
            <th class="pr-3 py-2">条款版本</th>
            <th class="pr-3 py-2">实质性变更</th>
            <th class="py-2">IP</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in consentRows" :key="row.id" class="border-b border-gray-100">
            <td class="pr-3 py-2 whitespace-nowrap">{{ formatDate(row.consented_at) }}</td>
            <td class="pr-3 py-2">{{ consentContextLabel(row.context) }}</td>
            <td class="pr-3 py-2">{{ row.privacy_version }} / {{ row.privacy_title }}</td>
            <td class="pr-3 py-2">{{ row.is_material_change ? '是' : '否' }}</td>
            <td class="py-2 text-gray-500">{{ row.client_ip || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-else-if="consentSearched && !consentError" class="mt-2 text-sm text-gray-500">无记录</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeJson } from '../utils/safeResponseJson.js'

const consentUserId = ref('')
const consentRows = ref([])
const consentLoading = ref(false)
const consentError = ref('')
const consentErrorTraceId = ref('')
const consentSearched = ref(false)

const formatDate = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const CONSENT_CONTEXT_LABELS = {
  register_phone: '手机号注册',
  register_email: '邮箱注册',
  login_password: '密码登录',
  post_login_reconsent: '登录后实质性更新确认'
}

const consentContextLabel = (code) => CONSENT_CONTEXT_LABELS[code] || String(code || '—')

const fetchUserConsents = async () => {
  const uid = String(consentUserId.value || '').trim()
  if (!uid) {
    consentError.value = '请输入用户 ID'
    consentErrorTraceId.value = ''
    return
  }
  consentLoading.value = true
  consentError.value = ''
  consentErrorTraceId.value = ''
  consentSearched.value = false
  try {
    const response = await apiFetch(
      `/api/system-admin/privacy-policy/user-consents/?user_id=${encodeURIComponent(uid)}`,
      {
        method: 'GET',
        headers: { Accept: 'application/json' },
        credentials: 'include'
      }
    )
    if (!response.ok) {
      const errData = await safeJson(response, {})
      const err = new Error(errData.detail || '查询失败')
      err.traceId = errData._traceId || response.traceId
      throw err
    }
    consentRows.value = await safeJson(response, [])
    consentSearched.value = true
  } catch (err) {
    consentError.value = err.message || '查询失败'
    consentErrorTraceId.value = err.traceId || ''
    consentRows.value = []
  } finally {
    consentLoading.value = false
  }
}
</script>
