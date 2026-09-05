<template>
  <!-- Teleport-OK: fixed-overlay — 隐私政策阻断须盖住整页，不受父级 overflow/z-index 限制 -->
  <Teleport to="body">
    <div
      v-if="showBlocker && policy"
      class="app-modal-overlay z-[10050] flex items-center justify-center bg-black/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="privacy-gate-title"
    >
      <div class="max-h-[90vh] w-full max-w-2xl overflow-hidden rounded-2xl bg-white shadow-2xl flex flex-col">
        <h2 id="privacy-gate-title" class="shrink-0 border-b border-gray-100 px-6 py-4 text-lg font-semibold text-gray-900">
          隐私政策更新
        </h2>
        <div class="min-h-0 flex-1 overflow-y-auto px-6 py-4 text-sm text-gray-800 whitespace-pre-wrap">
          <p class="mb-2 text-xs text-amber-800 bg-amber-50 border border-amber-100 rounded-md px-3 py-2">
            我们更新了隐私政策（{{ policy.version }}，含实质性变更）。请阅读完整
            <button
              type="button"
              class="text-primary underline hover:text-primary/80 font-medium"
              @click="showReader = true"
            >
              隐私条款
            </button>
            并同意后继续使用。
          </p>
          <h3 class="font-medium text-base mb-2">{{ policy.title }}</h3>
          <p class="text-gray-600 line-clamp-6">{{ policy.content }}</p>
          <button
            type="button"
            class="mt-2 text-sm text-primary hover:text-primary/80 underline"
            @click="showReader = true"
          >
            查看完整隐私条款
          </button>
        </div>
        <div class="shrink-0 border-t border-gray-100 px-6 py-4 flex justify-end gap-3">
          <button
            type="button"
            class="px-4 py-2.5 rounded-lg bg-primary text-white text-sm font-medium disabled:opacity-50"
            :disabled="consenting"
            @click="submitConsent"
          >
            {{ consenting ? '提交中...' : '同意并继续' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- Teleport-OK: fixed-overlay — 隐私条款阅读器须盖住整页 -->
  <Teleport to="body">
    <div
      v-if="showReader && policy"
      class="fixed inset-0 z-[10060] flex items-end sm:items-center justify-center bg-black/50 p-0 sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="privacy-reader-title"
      @click.self="showReader = false"
    >
      <div class="w-full sm:max-w-2xl max-h-[85vh] flex flex-col rounded-t-2xl sm:rounded-2xl bg-white shadow-2xl">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h2 id="privacy-reader-title" class="text-base font-semibold text-gray-900">
            {{ policy.title || '隐私条款' }}
          </h2>
          <button type="button" class="text-sm text-primary hover:text-primary/80" @click="showReader = false">
            关闭
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-800 whitespace-pre-wrap">
          <template v-if="!policy.content">正在加载…</template>
          <template v-else>{{ policy.content }}</template>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils.js'
import modalService from '../utils/modalService.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const route = useRoute()

const showBlocker = ref(false)
const showReader = ref(false)
const policy = ref(null)
const consenting = ref(false)
const inFlight = ref(false)
// OPT-20260819-038: 隐私同意为合规写路径，createClickGuard 防连点双发 POST。
const consentGuard = createClickGuard()
// OPT-20260728-001: Persist consentGiven in sessionStorage to survive page refreshes.
// This prevents the privacy policy popup from reappearing after the user clicks
// "同意并继续" and refreshes the page within the backend cache TTL window.
const SESSION_KEY = 'privacy_reconsent_dismissed_at'
const consentGiven = ref(_readSessionConsent())

function _readSessionConsent() {
  try {
    const ts = sessionStorage.getItem(SESSION_KEY)
    if (ts) {
      const elapsed = Date.now() - parseInt(ts, 10)
      // Treat consent as still valid if dismissed within the last 30 seconds
      if (elapsed < 30_000) return true
    }
  } catch { /* ignore */ }
  return false
}

function _writeSessionConsent() {
  try {
    sessionStorage.setItem(SESSION_KEY, String(Date.now()))
  } catch { /* ignore */ }
}

const isAuthPage = computed(() => {
  const n = route.name
  return n === 'login' || n === 'auth_login' || n === 'register' || n === 'auth_register' || n === 'people_join'
})

async function loadPending() {
  if (inFlight.value || isAuthPage.value || consentGiven.value) return
  const userId = String(getCookie('userId') || '').trim()
  if (!userId) {
    showBlocker.value = false
    policy.value = null
    return
  }
  if (!userId) {
    return
  }
  inFlight.value = true
  try {
    const res = await apiFetch(`/api/accounts/users/me/`, {
      method: 'GET',
      headers: { Accept: 'application/json' }
    })
    if (!res.ok) {
      showBlocker.value = false
      policy.value = null
      return
    }
    const data = await res.json().catch(() => ({}))
    const p = data.pending_privacy_policy
    if (p && p.id) {
      policy.value = p
      showBlocker.value = true
    } else {
      showBlocker.value = false
      policy.value = null
    }
  } finally {
    inFlight.value = false
  }
}

watch(
  () => [route.fullPath, isAuthPage.value],
  () => {
    loadPending()
  },
  { immediate: true }
)

async function submitConsent() {
  if (!policy.value) return
  await consentGuard.run(async ({ idempotencyKey }) => {
    consenting.value = true
    try {
      const res = await apiFetch('/api/privacy-policy/consent/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ privacy_policy_id: policy.value.id })
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        const msg = (err && (err.detail || err.accepted_privacy_policy_id)) || '操作失败'
        const text = typeof msg === 'string' ? msg : JSON.stringify(msg)
        modalService.alert(text, '错误', { traceId: res.traceId || err.trace_id || err.traceId })
        return
      }
      consentGiven.value = true
      _writeSessionConsent()
      showBlocker.value = false
      policy.value = null
    } catch (e) {
      modalService.alert('网络错误，请重试', '错误', { traceId: e?.traceId })
    } finally {
      consenting.value = false
    }
  })
}
</script>
