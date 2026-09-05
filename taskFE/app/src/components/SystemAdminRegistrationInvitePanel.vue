<template>
  <div
    data-testid="system-admin-registration-invite-panel"
    class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8"
  >
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-lg font-semibold text-gray-900">注册邀请码</h2>
        <p class="text-sm text-gray-500 mt-1">
          控制是否开启注册邀请码、每日放量，并查看邀请关系。
        </p>
      </div>
      <button
        type="button"
        class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
        data-testid="registration-invite-policy-save"
        :disabled="loading || saving"
        @click="savePolicy"
      >
        {{ saving ? '保存中...' : '保存策略' }}
      </button>
    </div>

    <div class="space-y-5 mb-6">
      <div class="flex items-center justify-between gap-4">
        <div>
          <label for="enable-registration-invite-switch" class="block text-sm font-medium text-gray-900">
            开启注册邀请码
          </label>
          <p class="text-sm text-gray-500 mt-1">开启后，新用户注册必须填写有效邀请码。</p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            id="enable-registration-invite-switch"
            v-model="policy.enabled"
            data-testid="enable-registration-invite-switch"
            type="checkbox"
            class="sr-only peer"
            :disabled="loading || saving"
          >
          <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
        </label>
      </div>

      <div>
        <label for="daily-invite-quota" class="block text-sm font-medium text-gray-900 mb-1">
          每天放量
        </label>
        <input
          id="daily-invite-quota"
          v-model.number="policy.daily_quota"
          data-testid="daily-invite-quota-input"
          type="number"
          min="0"
          step="1"
          class="w-40 px-3 py-2 border border-gray-300 rounded-lg"
          :disabled="loading || saving"
        >
        <p class="text-sm text-gray-500 mt-1">
          今日已剩余：{{ policy.remaining_today ?? '—' }}（自然日 Asia/Shanghai）
        </p>
      </div>
    </div>

    <div v-if="loading" class="text-sm text-gray-500">策略加载中...</div>
    <div
      v-if="errorMessage"
      class="mt-2 p-3 bg-red-50 text-red-700 rounded-lg text-sm"
      :data-traceId="errorTraceId || undefined"
    >
      {{ errorMessage }}
    </div>
    <div v-if="successMessage" class="mt-2 p-3 bg-green-50 text-green-700 rounded-lg text-sm">
      {{ successMessage }}
    </div>

    <div class="mt-8">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-gray-900">注册邀请关系</h3>
        <button
          type="button"
          class="text-sm text-primary hover:underline disabled:opacity-50"
          :disabled="loadingRelations"
          data-testid="registration-invite-relations-refresh"
          @click="loadRelations"
        >
          {{ loadingRelations ? '刷新中...' : '刷新' }}
        </button>
      </div>
      <div v-if="loadingRelations" class="text-sm text-gray-500">关系加载中...</div>
      <div v-else-if="!relations.length" class="text-sm text-gray-500">暂无邀请关系。</div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm" data-testid="registration-invite-relations-table">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="py-2 pr-3 font-medium">邀请码</th>
              <th class="py-2 pr-3 font-medium">发放人</th>
              <th class="py-2 pr-3 font-medium">状态</th>
              <th class="py-2 pr-3 font-medium">使用人</th>
              <th class="py-2 pr-3 font-medium">发放日</th>
              <th class="py-2 font-medium">核销时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in relations" :key="row.id || row.code" class="border-b border-gray-100">
              <td class="py-2 pr-3 font-mono">{{ row.code }}</td>
              <td class="py-2 pr-3 font-mono text-xs">{{ row.issuer_user_id || '—' }}</td>
              <td class="py-2 pr-3">{{ row.status || '—' }}</td>
              <td class="py-2 pr-3 font-mono text-xs">{{ row.redeemed_by_user_id || '—' }}</td>
              <td class="py-2 pr-3">{{ row.issued_day || '—' }}</td>
              <td class="py-2">{{ row.redeemed_at || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:comp-system-admin-registration-invite-panel */
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const POLICY_API = '/api/system-admin/registration-invite-policy/'
const RELATIONS_API = '/api/system-admin/registration-invite-relations/'

const loading = ref(false)
const saving = ref(false)
const loadingRelations = ref(false)
const errorMessage = ref('')
const errorTraceId = ref('')
const successMessage = ref('')
const relations = ref([])

// OPT-20260819-038: 保存策略是写操作，防连点/超时重试双发 PUT
const savePolicyGuard = createClickGuard()
const policy = ref({
  enabled: false,
  daily_quota: 0,
  remaining_today: 0,
})

const extractTrace = (response, data) =>
  response?.headers?.get?.('X-Trace-Id') || data?.trace_id || data?.traceId || ''

const extractError = (data, fallback) => {
  if (typeof data?.error === 'string' && data.error.trim()) return data.error.trim()
  if (typeof data?.detail === 'string' && data.detail.trim()) return data.detail.trim()
  if (typeof data?.message === 'string' && data.message.trim()) return data.message.trim()
  return fallback
}

const loadPolicy = async () => {
  loading.value = true
  errorMessage.value = ''
  errorTraceId.value = ''
  try {
    const response = await apiFetch(POLICY_API, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      // 网关 forward-auth 会话失效（无法解析登录凭据）由 apiFetch 统一引导重新登录，
      // 此处不再逐页接线（见 utils/apiUtils.js / requestErrorDisplay.js）。
      errorMessage.value = extractError(data, `策略加载失败（${response.status}）`)
      errorTraceId.value = extractTrace(response, data)
      return
    }
    policy.value = {
      enabled: Boolean(data.enabled),
      daily_quota: Number(data.daily_quota) || 0,
      remaining_today: Number(data.remaining_today) || 0,
    }
  } catch (err) {
    errorMessage.value = err?.message || '策略加载失败'
  } finally {
    loading.value = false
  }
}

const savePolicy = async () => {
  // OPT-20260819-038: 保存策略是写操作，防连点/超时重试双发 PUT
  await savePolicyGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    successMessage.value = ''
    try {
      const response = await apiFetch(POLICY_API, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({
          enabled: Boolean(policy.value.enabled),
          daily_quota: Math.max(0, Number(policy.value.daily_quota) || 0),
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorMessage.value = extractError(data, `保存失败（${response.status}）`)
        errorTraceId.value = extractTrace(response, data)
        return
      }
      policy.value = {
        enabled: Boolean(data.enabled),
        daily_quota: Number(data.daily_quota) || 0,
        remaining_today: Number(data.remaining_today) || 0,
      }
      successMessage.value = '策略已保存'
    } catch (err) {
      errorMessage.value = err?.message || '保存失败'
    } finally {
      saving.value = false
    }
  })
}

const loadRelations = async () => {
  loadingRelations.value = true
  try {
    const response = await apiFetch(`${RELATIONS_API}?page=1&page_size=50`, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorMessage.value = extractError(data, `关系加载失败（${response.status}）`)
      errorTraceId.value = extractTrace(response, data)
      relations.value = []
      return
    }
    relations.value = Array.isArray(data?.results) ? data.results : []
  } catch (err) {
    errorMessage.value = err?.message || '关系加载失败'
    relations.value = []
  } finally {
    loadingRelations.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadPolicy(), loadRelations()])
})
</script>
