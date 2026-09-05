<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex justify-end bg-black/40"
    data-alias="SystemAdminUserRechargeDrawer"
    @click.self="$emit('close')"
  >
    <div class="h-full w-full max-w-xl bg-white shadow-xl flex flex-col">
      <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900">支付与签署</h3>
          <p class="text-xs text-gray-500 mt-0.5">用户 ID：{{ userId }}</p>
        </div>
        <button type="button" class="text-gray-500 hover:text-gray-700" @click="$emit('close')">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4 space-y-4">
        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500">
          <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mr-3" />
          加载中…
        </div>
        <div v-else-if="error" class="space-y-2">
          <p
            class="text-sm text-red-600"
            v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
          >
            {{ error }}
          </p>
          <!-- Anti-Replay-OK: GET 只读重试 -->
          <button
            type="button"
            class="text-sm text-primary hover:underline"
            data-testid="recharge-load-retry"
            @click="load"
          >
            重试
          </button>
        </div>
        <template v-else>
          <div v-if="recharges.length === 0" class="text-sm text-gray-500 py-8 text-center">
            暂无支付记录
          </div>
          <div
            v-for="(row, idx) in recharges"
            :key="rowKey(row, idx)"
            class="rounded-lg border border-gray-200 p-4 space-y-2"
          >
            <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm">
              <span class="text-gray-900 font-medium">
                {{ formatAmount(row) }}
              </span>
              <span class="text-gray-600">金额 {{ formatYuanCents(row) }}</span>
              <span class="text-gray-500">渠道 {{ formatChannel(row) }}</span>
            </div>
            <div class="text-xs text-gray-500 flex flex-wrap gap-x-4 gap-y-1">
              <span>时间 {{ formatDate(row.created_at || row.paid_at) }}</span>
              <span>有效期至 {{ formatDate(row.expires_at) }}</span>
            </div>
            <p v-if="row.description" class="text-xs text-gray-600">{{ row.description }}</p>
            <div v-if="row.consent" class="pt-2 border-t border-gray-100">
              <!-- Anti-Replay-OK: 只读展开条款正文，无写请求 -->
              <button
                type="button"
                class="text-sm text-primary hover:underline"
                @click="toggleConsent(rowKey(row, idx))"
              >
                {{ expanded[rowKey(row, idx)] ? '收起签署条款' : '查看签署条款' }}
                （v{{ row.consent.agreement_version || '—' }}）
              </button>
              <div
                v-if="expanded[rowKey(row, idx)]"
                class="mt-2 max-h-48 overflow-y-auto rounded bg-gray-50 p-3 text-xs text-gray-700 whitespace-pre-wrap"
              >
                {{ row.consent.content_snapshot || '无正文快照' }}
              </div>
            </div>
            <p v-else-if="isAdminGrant(row)" class="text-xs text-amber-700">
              {{ row.consent_note || '系统赠送，无需支付签署' }}
            </p>
            <p v-else class="text-xs text-gray-400">无关联签署记录</p>
          </div>

          <div v-if="orphanConsents.length" class="pt-4 border-t border-gray-200">
            <h4
              data-testid="orphan-consent-heading"
              class="text-sm font-medium text-gray-800 mb-1"
            >
              {{ orphanConsentHeading }}
            </h4>
            <p class="text-xs text-gray-500 mb-2 leading-relaxed">{{ orphanConsentHint }}</p>
            <div
              v-for="c in orphanConsents"
              :key="c.id"
              class="rounded-lg border border-dashed border-gray-200 p-3 mb-2"
            >
              <!-- Anti-Replay-OK: 只读展开条款正文，无写请求 -->
              <button
                type="button"
                class="text-sm text-primary hover:underline"
                @click="toggleConsent('orphan-' + c.id)"
              >
                {{ expanded['orphan-' + c.id] ? '收起' : '展开' }}
                {{ licenseDocumentKindLabel(c.document_kind) }}
                v{{ c.agreement_version || '—' }} · {{ formatDate(c.consented_at) }}
              </button>
              <div
                v-if="expanded['orphan-' + c.id]"
                class="mt-2 max-h-48 overflow-y-auto rounded bg-gray-50 p-3 text-xs text-gray-700 whitespace-pre-wrap"
              >
                {{ c.content_snapshot || '无正文快照' }}
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { messageFromFailedResponse } from '../utils/httpError.js'
import {
  formatRechargeChannel,
  isAdminGrantRecharge,
  licenseDocumentKindLabel,
  unboundPaymentConsents,
  ORPHAN_CONSENT_HEADING,
  ORPHAN_CONSENT_HINT,
} from '../utils/adminUserRechargeDisplay.js'
import { formatYuanFromCents } from '../utils/formatYuanCents.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  userId: { type: [String, Number], default: '' },
})

defineEmits(['close'])

const loading = ref(false)
const error = ref('')
const errorTraceId = ref('')
const recharges = ref([])
const consents = ref([])
const expanded = ref({})

const orphanConsentHeading = ORPHAN_CONSENT_HEADING
const orphanConsentHint = ORPHAN_CONSENT_HINT

const orphanConsents = computed(() => unboundPaymentConsents(recharges.value, consents.value))

function rowKey(row, idx) {
  return String(row.transaction_id || row.id || row.provider_ref || idx)
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

function formatAmount(row) {
  if (isAdminGrantRecharge(row)) return '系统赠送'
  const yuan = row.amount_yuan ?? row.amount ?? row.paid_amount
  if (yuan == null || yuan === '') return '金额 —'
  return `金额 ${yuan}`
}

function formatYuanCents(row) {
  const pts = row.points ?? row.amount_points ?? row.remaining_points
  return formatYuanFromCents(pts)
}

function formatChannel(row) {
  return formatRechargeChannel(row)
}

function isAdminGrant(row) {
  return isAdminGrantRecharge(row)
}

function toggleConsent(key) {
  expanded.value = { ...expanded.value, [key]: !expanded.value[key] }
}

async function load() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  loading.value = true
  error.value = ''
  errorTraceId.value = ''
  recharges.value = []
  consents.value = []
  expanded.value = {}
  try {
    const response = await apiFetch(`/api/system-admin/users/${encodeURIComponent(uid)}/recharges/`, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      error.value = messageFromFailedResponse(response, '加载支付记录失败')
      errorTraceId.value = extractTraceId(response) || extractTraceId(response._errorData) || ''
      return
    }
    const data = await response.json().catch(() => ({}))
    recharges.value = Array.isArray(data.recharges) ? data.recharges : []
    consents.value = Array.isArray(data.consents) ? data.consents : []
  } catch (e) {
    error.value = '网络错误，请稍后重试'
    errorTraceId.value = extractTraceId(e) || ''
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.visible, props.userId],
  ([v]) => {
    if (v) load()
  }
)
</script>
