<template>
  <div data-alias="view-system-admin-oidc-extension" class="p-6">
    <!-- 页面标题 -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">浏览器插件白名单</h1>
      <p class="text-sm text-gray-500 mt-1">
        管理 Task Chrome 插件 OIDC 登录允许的扩展 ID（32 位小写 a-p）。
        未登记的插件 ID 登录时会被拒绝（redirect_uri not allowed / invalid_client）。
      </p>
    </div>

    <!-- 加载失败 -->
    <div v-if="loadError" class="mb-6 p-4 bg-red-50 text-red-700 rounded-lg text-sm" :data-traceId="loadErrorTraceId || undefined">
      {{ loadError }}
    </div>

    <template v-if="!loading">
      <!-- 已登记插件 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-2">已登记插件</h2>
        <p class="text-sm text-gray-500 mb-3">
          当前 {{ extensionIds.length }} / 20 个&nbsp;·&nbsp;client: <span class="font-mono">{{ clientId }}</span>&nbsp;·&nbsp;托管: {{ managedBy === 'admin' ? '管理员（DB 为准）' : 'bootstrap（conf 自愈）' }}
        </p>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="id in extensionIds"
            :key="id"
            class="inline-flex items-center px-3 py-1 bg-primary/10 text-primary rounded-full text-sm font-mono"
          >{{ id }}</span>
          <span v-if="!extensionIds.length" class="text-sm text-gray-400">暂无登记插件</span>
        </div>
      </div>

      <!-- 编辑区 -->
      <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">编辑插件 ID 列表</h2>
        <textarea
          v-model="draft"
          rows="6"
          class="w-full border border-gray-300 rounded-lg p-3 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/30"
          placeholder="每行一个 32 位扩展 ID（小写 a-p），如&#10;aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        ></textarea>
        <p class="text-sm text-gray-500 mt-2">每行一个 ID；保存时自动去重，上限 20 个，至少 1 个。</p>

        <div v-if="validationError" class="mt-3 p-3 bg-red-50 text-red-700 rounded-lg text-sm">{{ validationError }}</div>

        <div class="mt-4 flex items-center gap-3">
          <button
            @click="handleSave"
            :disabled="saving || saveGuard.isBusy()"
            class="px-4 py-2 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            保存
          </button>
          <span v-if="saving" class="text-sm text-gray-500">保存中…</span>
          <span
            v-else-if="saveResult"
            class="text-sm"
            :class="saveResult.success ? 'text-green-700' : 'text-red-700'"
            :data-traceId="saveResult.traceId || undefined"
          >{{ saveResult.message }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
/* @alias:view-system-admin-oidc-extension */
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const API = '/api/system-admin/oidc-extension/'
const EXT_ID_RE = /^[a-p]{32}$/
const MAX_IDS = 20

const loading = ref(true)
const loadError = ref('')
const loadErrorTraceId = ref('')
const clientId = ref('')
const managedBy = ref('bootstrap')
const extensionIds = ref([])
const draft = ref('')
const saving = ref(false)
const validationError = ref('')
const saveResult = ref(null)

const extractError = (data, fallback) =>
  (data && (data.error || data.detail || data.message)) || fallback

const loadState = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await apiFetch(API, { method: 'GET', headers: { Accept: 'application/json' } })
    const { data, traceId } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      loadErrorTraceId.value = traceId
      loadError.value = extractError(data, `加载失败（${response.status}）`)
      return
    }
    clientId.value = data.client_id || ''
    managedBy.value = data.managed_by || 'bootstrap'
    extensionIds.value = Array.isArray(data.extension_ids) ? data.extension_ids : []
    draft.value = extensionIds.value.join('\n')
  } catch (error) {
    loadErrorTraceId.value = error.traceId || ''
    loadError.value = error?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

// parseDraft 前端预校验（与后端权威校验同规则：正则/非空/去重/≤20/≥1）
const parseDraft = () => {
  const raw = draft.value
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s !== '')
  if (raw.length === 0) return { ok: false, error: '至少需要一个插件 ID' }
  const seen = new Set()
  const ids = []
  for (const id of raw) {
    if (!EXT_ID_RE.test(id)) return { ok: false, error: `非法插件 ID：${id}（需 32 位小写 a-p）` }
    if (!seen.has(id)) {
      seen.add(id)
      ids.push(id)
    }
  }
  if (ids.length > MAX_IDS) return { ok: false, error: `插件 ID 最多 ${MAX_IDS} 个，当前 ${ids.length} 个` }
  return { ok: true, ids }
}

const saveGuard = createClickGuard()

const handleSave = async () => {
  // OPT-20260819-038: 保存插件白名单是写操作，防连点双发 PUT
  await saveGuard.run(async ({ idempotencyKey }) => {
    validationError.value = ''
    saveResult.value = null
    const parsed = parseDraft()
    if (!parsed.ok) {
      validationError.value = parsed.error
      return
    }
    saving.value = true
    try {
      const response = await apiFetch(API, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ extension_ids: parsed.ids }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (response.ok) {
        const ids = Array.isArray(data.extension_ids) ? data.extension_ids : parsed.ids
        extensionIds.value = ids
        managedBy.value = data.managed_by || 'admin'
        draft.value = ids.join('\n')
        saveResult.value = { success: true, message: `已保存 ${ids.length} 个插件 ID` }
      } else {
        saveResult.value = {
          success: false,
          message: extractError(data, `保存失败（${response.status}）`),
          traceId,
        }
      }
    } catch (error) {
      saveResult.value = {
        success: false,
        message: error?.message || '网络错误，请稍后重试',
        traceId: error.traceId || '',
      }
    } finally {
      saving.value = false
    }
  })
}

onMounted(loadState)
</script>
