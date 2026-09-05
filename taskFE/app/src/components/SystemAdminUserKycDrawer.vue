<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex justify-end bg-black/40"
    data-alias="SystemAdminUserKycDrawer"
    @click.self="$emit('close')"
  >
    <div class="h-full w-full max-w-xl bg-white shadow-xl flex flex-col">
      <div class="border-b border-gray-100 px-5 py-4 space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-lg font-semibold text-gray-900">KYC 身份等级</h3>
            <p class="text-xs text-gray-500 mt-0.5">用户 ID：{{ userId }}</p>
          </div>
          <button type="button" class="text-gray-500 hover:text-gray-700 shrink-0" @click="$emit('close')">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div
          data-testid="kyc-tier-help"
          class="rounded-md border border-gray-100 bg-gray-50 px-3 py-2"
          aria-label="KYC 各等级说明"
        >
          <p class="text-xs font-medium text-gray-700">各等级说明</p>
          <ul class="mt-1.5 space-y-1 text-xs text-gray-600 leading-relaxed">
            <li v-for="row in tierHelpRows" :key="row.tier">
              <span class="font-medium text-gray-800">{{ row.label }}：</span>{{ row.description }}
              <template v-if="row.maxSingle != null || row.maxDaily != null">
                <span class="text-gray-500">（单笔≤{{ formatLimitYuan(row.maxSingle) }}元 · 日累计≤{{ formatLimitYuan(row.maxDaily) }}元）</span>
              </template>
            </li>
          </ul>
          <p class="mt-1.5 text-[11px] text-gray-400">单笔/日累计限额以限额策略配置为准，可随策略调整。</p>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4 space-y-5">
        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500">
          <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mr-3" />
          加载中…
        </div>
        <p
          v-else-if="error"
          class="text-sm text-red-600"
          v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
        >
          {{ error }}
        </p>
        <template v-else>
          <section class="rounded-lg border border-gray-200 p-4 space-y-2">
            <h4 class="text-sm font-medium text-gray-800">当前状态</h4>
            <div class="flex flex-wrap gap-2 text-sm">
              <span class="px-2 py-0.5 rounded-full bg-blue-50 text-blue-800">{{ tierLabel(profile.tier) }}</span>
              <span class="px-2 py-0.5 rounded-full bg-gray-100 text-gray-800">{{ statusLabel(profile.status) }}</span>
            </div>
            <p v-if="latestAml" class="text-xs text-gray-500">
              最近 AML：{{ amlLabel(latestAml.result) }}
              <span v-if="latestAml.checked_at"> · {{ formatDate(latestAml.checked_at) }}</span>
            </p>
            <button
              type="button"
              class="mt-2 text-sm text-primary hover:underline disabled:opacity-50"
              :disabled="evaluating"
              @click="runEvaluate"
            >
              {{ evaluating ? '评估中…' : '触发自动评估（手机验证→T1）' }}
            </button>
          </section>

          <section class="rounded-lg border border-gray-200 p-4 space-y-3">
            <h4 class="text-sm font-medium text-gray-800">人工覆盖</h4>
            <div class="grid grid-cols-2 gap-3">
              <label class="text-xs text-gray-600 block">
                等级
                <select v-model="overrideForm.tier" class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm">
                  <option value="T0_unverified">T0 未验证</option>
                  <option value="T1_basic">T1 基础</option>
                  <option value="T2_enhanced">T2 增强</option>
                </select>
              </label>
              <label class="text-xs text-gray-600 block">
                状态
                <select v-model="overrideForm.status" class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm">
                  <option value="none">无</option>
                  <option value="pending">审核中</option>
                  <option value="approved">已通过</option>
                  <option value="rejected">已拒绝</option>
                  <option value="needs_review">需复核</option>
                  <option value="expired">已过期</option>
                </select>
              </label>
            </div>
            <label class="text-xs text-gray-600 block">
              原因说明
              <input
                v-model="overrideForm.reason_detail"
                type="text"
                class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm"
                placeholder="简要说明（勿填证件全文）"
              />
            </label>
            <p
              v-if="actionError"
              class="text-xs text-red-600"
              v-bind="actionErrorTraceId ? { 'data-traceId': actionErrorTraceId } : {}"
            >
              {{ actionError }}
            </p>
            <button
              type="button"
              class="px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              :disabled="overriding"
              @click="submitOverride"
            >
              {{ overriding ? '提交中…' : '提交覆盖' }}
            </button>
          </section>

          <section class="rounded-lg border border-gray-200 p-4 space-y-3">
            <h4 class="text-sm font-medium text-gray-800">AML 筛查登记</h4>
            <label class="text-xs text-gray-600 block">
              结果
              <select v-model="amlForm.result" class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm">
                <option value="pending">待处理</option>
                <option value="clear">通过</option>
                <option value="review">复核</option>
                <option value="hit">命中</option>
              </select>
            </label>
            <label class="text-xs text-gray-600 block">
              备注
              <input
                v-model="amlForm.notes"
                type="text"
                class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm"
                placeholder="可选备注"
              />
            </label>
            <button
              type="button"
              class="px-3 py-1.5 text-sm bg-gray-800 text-white rounded-lg hover:bg-gray-700 disabled:opacity-50"
              :disabled="recordingAml"
              @click="submitAml"
            >
              {{ recordingAml ? '登记中…' : '登记 AML 结果' }}
            </button>
          </section>

          <section class="space-y-2">
            <h4 class="text-sm font-medium text-gray-800">审计时间线</h4>
            <div v-if="!audit.length" class="text-sm text-gray-500 py-6 text-center">暂无审计记录</div>
            <ol class="space-y-2">
              <li
                v-for="row in audit"
                :key="row.id"
                class="rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-xs text-gray-700"
              >
                <div class="flex justify-between gap-2">
                  <span class="font-medium">{{ formatDate(row.created_at) }}</span>
                  <span class="text-gray-500">{{ triggerLabel(row.trigger_source) }}</span>
                </div>
                <div class="mt-1">
                  {{ tierLabel(row.old_tier) }} → {{ tierLabel(row.new_tier) }}
                  · {{ statusLabel(row.old_status) }} → {{ statusLabel(row.new_status) }}
                </div>
                <div v-if="row.reason_code || row.reason_detail" class="mt-0.5 text-gray-500">
                  {{ row.reason_code }}{{ row.reason_detail ? `：${row.reason_detail}` : '' }}
                </div>
              </li>
            </ol>
          </section>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { KYC_TIER_HELP } from '../utils/kycTierHelp.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  userId: { type: [String, Number], default: '' },
})

defineEmits(['close'])

const loading = ref(false)
const error = ref('')
const errorTraceId = ref('')
const actionError = ref('')
const actionErrorTraceId = ref('')
const profile = ref({})
const audit = ref([])
const latestAml = ref(null)
const limitPolicies = ref([])
const evaluating = ref(false)
const overriding = ref(false)
const recordingAml = ref(false)

// OPT-20260819-029：等级说明与 auth_kyc_limit_policy 同源，金额不硬编码。
const tierHelpRows = computed(() => {
  const byTier = new Map((limitPolicies.value || []).map((p) => [String(p.tier), p]))
  return KYC_TIER_HELP.map((row) => {
    const p = byTier.get(row.tier)
    return {
      ...row,
      maxSingle: p ? Number(p.max_single_yuan) : null,
      maxDaily: p ? Number(p.max_daily_yuan) : null,
    }
  })
})

function formatLimitYuan(v) {
  if (v === null || v === undefined || Number.isNaN(Number(v))) return null
  return Number(v).toLocaleString('zh-CN')
}

const overrideForm = ref({
  tier: 'T1_basic',
  status: 'approved',
  reason_detail: '',
})

const amlForm = ref({
  result: 'clear',
  notes: '',
})

const TIER_MAP = {
  T0_unverified: 'T0 未验证',
  T1_basic: 'T1 基础',
  T2_enhanced: 'T2 增强',
}

const STATUS_MAP = {
  none: '无',
  pending: '审核中',
  approved: '已通过',
  rejected: '已拒绝',
  needs_review: '需复核',
  expired: '已过期',
}

const AML_MAP = {
  pending: '待处理',
  clear: '通过',
  review: '复核',
  hit: '命中',
}

const TRIGGER_MAP = {
  system: '系统',
  admin: '人工',
  evaluate: '评估',
  aml: 'AML',
}

function tierLabel(v) {
  return TIER_MAP[v] || v || '—'
}
function statusLabel(v) {
  return STATUS_MAP[v] || v || '—'
}
function amlLabel(v) {
  return AML_MAP[v] || v || '—'
}
function triggerLabel(v) {
  return TRIGGER_MAP[v] || v || '—'
}

function formatDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function setActionError(message, source) {
  actionError.value = message
  actionErrorTraceId.value = extractTraceId(source) || ''
}

async function load() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  loading.value = true
  error.value = ''
  errorTraceId.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
  profile.value = {}
  audit.value = []
  latestAml.value = null
  try {
    const response = await apiFetch(`/api/kyc/admin/users/${encodeURIComponent(uid)}/`, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      error.value = data.detail || data.error || '加载 KYC 失败'
      errorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      return
    }
    profile.value = data.profile || {}
    audit.value = Array.isArray(data.audit) ? data.audit : []
    latestAml.value = data.latest_aml || null
    limitPolicies.value = Array.isArray(data.limit_policies) ? data.limit_policies : []
    if (profile.value.tier) overrideForm.value.tier = profile.value.tier
    if (profile.value.status) overrideForm.value.status = profile.value.status
  } catch (e) {
    error.value = '网络错误，请稍后重试'
    errorTraceId.value = extractTraceId(e) || ''
  } finally {
    loading.value = false
  }
}

const submitOverrideGuard = createClickGuard()
const submitAmlGuard = createClickGuard()
const runEvaluateGuard = createClickGuard()

async function submitOverride() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  // OPT-20260819-038: KYC 覆盖是安全审计写路径，防连点双发 POST
  await submitOverrideGuard.run(async ({ idempotencyKey }) => {
    overriding.value = true
    actionError.value = ''
    actionErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/kyc/admin/users/${encodeURIComponent(uid)}/override/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({
          tier: overrideForm.value.tier,
          status: overrideForm.value.status,
          reason_code: 'ADMIN_OVERRIDE',
          reason_detail: overrideForm.value.reason_detail || '',
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        setActionError(data.detail || data.error || '覆盖失败', response.ok ? data : response)
        return
      }
      await load()
    } catch (e) {
      setActionError('网络错误，请稍后重试', e)
    } finally {
      overriding.value = false
    }
  })
}

async function submitAml() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  // OPT-20260819-038: AML 登记是合规写路径，防连点双发 POST
  await submitAmlGuard.run(async ({ idempotencyKey }) => {
    recordingAml.value = true
    actionError.value = ''
    actionErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/kyc/admin/users/${encodeURIComponent(uid)}/aml/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({
          result: amlForm.value.result,
          provider: 'manual',
          notes: amlForm.value.notes || '',
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        setActionError(data.detail || data.error || 'AML 登记失败', response.ok ? data : response)
        return
      }
      await load()
    } catch (e) {
      setActionError('网络错误，请稍后重试', e)
    } finally {
      recordingAml.value = false
    }
  })
}

async function runEvaluate() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  // OPT-20260819-038: KYC 评估是资源写路径，防连点双发 POST
  await runEvaluateGuard.run(async ({ idempotencyKey }) => {
    evaluating.value = true
    actionError.value = ''
    actionErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/kyc/admin/users/${encodeURIComponent(uid)}/evaluate/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({}),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        setActionError(data.detail || data.error || '评估失败', response.ok ? data : response)
        return
      }
      await load()
    } catch (e) {
      setActionError('网络错误，请稍后重试', e)
    } finally {
      evaluating.value = false
    }
  })
}

watch(
  () => [props.visible, props.userId],
  ([v]) => {
    if (v) load()
  }
)
</script>
