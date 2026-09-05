<template>
  <!-- Teleport-OK: fixed-overlay — 手机号验证门禁须盖住整页，不受父级 overflow 裁剪 -->
  <Teleport to="body">
    <div
      v-if="showBlocker"
      class="app-modal-overlay z-[10040] flex items-center justify-center bg-black/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="phone-verify-gate-title"
      data-testid="phone-verify-access-gate"
    >
      <div class="w-full max-w-md rounded-2xl bg-white p-6 shadow-2xl">
        <h2 id="phone-verify-gate-title" class="text-lg font-semibold text-gray-900">
          {{ LOGIN_PHONE_VERIFY_PROMPT_TITLE }}
        </h2>
        <p class="mt-3 text-sm text-gray-700">{{ LOGIN_PHONE_VERIFY_PROMPT_MESSAGE }}</p>
        <div class="mt-6 flex justify-end">
          <!-- Anti-Replay-OK: navigation — 整页跳资料绑定，无写请求 -->
          <a
            class="px-4 py-2.5 rounded-lg bg-primary text-white text-sm font-medium"
            :href="bindingHref"
            @click="onGoVerify"
          >{{ LOGIN_PHONE_VERIFY_PROMPT_CONFIRM }}</a>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils.js'
import { getStoredUserId } from '../utils/sessionUserIdUtils.js'
import {
  LOGIN_PHONE_VERIFY_PROMPT_CONFIRM,
  LOGIN_PHONE_VERIFY_PROMPT_MESSAGE,
  LOGIN_PHONE_VERIFY_PROMPT_TITLE,
} from '../domain/auth/services/post_login_phone_verify_prompt_service.js'
import { shouldBlockForUnverifiedPhone } from '../domain/auth/services/phone_verify_access_gate.js'
import {
  buildPhoneBindingRedirectUrl,
  savePhoneVerifyRedirect,
} from '../utils/phoneBindingDeepLink.js'

const route = useRoute()
const showBlocker = ref(false)
const inFlight = ref(false)
const bindingHref = buildPhoneBindingRedirectUrl()

function isImpersonating() {
  try {
    return Boolean(sessionStorage.getItem('impersonatorAccountBackup'))
  } catch {
    return false
  }
}

function onGoVerify() {
  const nextPath = `${route.path}${route.hash || ''}`
  savePhoneVerifyRedirect(nextPath)
}

async function evaluate() {
  const userId = String(getStoredUserId() || getCookie('userId') || '').trim()
  if (!userId) {
    showBlocker.value = false
    return
  }
  if (inFlight.value) return
  inFlight.value = true
  try {
    const res = await apiFetch('/api/accounts/users/me/', {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    if (!res.ok) {
      showBlocker.value = false
      return
    }
    const data = await res.json().catch(() => ({}))
    showBlocker.value = shouldBlockForUnverifiedPhone({
      sessionKnown: true,
      source: data,
      path: route.path,
      query: route.query,
      isImpersonating: isImpersonating(),
    })
  } catch {
    showBlocker.value = false
  } finally {
    inFlight.value = false
  }
}

watch(
  () => route.fullPath,
  () => {
    evaluate()
  },
  { immediate: true },
)
</script>
