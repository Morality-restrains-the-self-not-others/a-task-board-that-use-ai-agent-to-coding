<template>
  <div class="bg-white rounded-xl shadow-sm border border-red-200 p-6">
    <h2 class="text-lg font-semibold text-danger mb-2">账号注销</h2>
    <p class="text-sm text-text-light mb-4">
      注销后账号将进入归档状态，个人信息将被脱敏处理。提交后有 {{ cooldownDays }} 天冷却期，期间可随时撤回。
    </p>

    <div v-if="isLoading" class="text-sm text-text-light">检查注销条件…</div>

    <div v-else-if="status === 'pending_cooldown'" class="space-y-4">
      <p class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
        注销申请已提交，预计于 <strong>{{ effectiveAtLabel }}</strong> 生效。冷却期内可撤回。
      </p>
      <button
        type="button"
        :disabled="isBusy"
        class="px-4 py-2 rounded-lg border border-border text-text hover:bg-gray-50 disabled:opacity-60"
        @click="cancelDeletion"
      >
        {{ isBusy ? '处理中…' : '撤回注销申请' }}
      </button>
    </div>

    <div v-else-if="status === 'completed'" class="text-sm text-text-light">
      该账号已完成注销流程。
    </div>

    <div v-else class="space-y-4">
      <div v-if="blockers.length" class="rounded-lg border border-amber-200 bg-amber-50 p-4 space-y-2">
        <p class="text-sm font-medium text-amber-900">请先处理以下事项后再申请注销：</p>
        <ul class="space-y-2">
          <li v-for="(item, idx) in blockers" :key="idx" class="text-sm text-amber-900">
            {{ item.message }}
            <!-- Anti-Replay-OK: 真实 a[href] 原生导航，非写操作按钮 -->
            <a
              v-if="blockerActionHref(item)"
              :href="blockerActionHref(item)"
              class="ml-2 underline text-primary"
              data-testid="account-deletion-blocker-action"
            >前往处理</a>
          </li>
        </ul>
      </div>

      <div v-else class="space-y-3">
        <label class="block text-sm text-text-light">
          请输入「注销」并验证密码以确认身份
        </label>
        <input
          v-model="confirmationText"
          type="text"
          placeholder="注销"
          class="w-full max-w-md px-4 py-2 border border-border rounded-lg"
        >
        <input
          v-model="password"
          type="password"
          autocomplete="current-password"
          placeholder="当前密码"
          class="w-full max-w-md px-4 py-2 border border-border rounded-lg"
        >
        <button
          type="button"
          :disabled="isBusy || !canSubmit"
          class="px-4 py-2 rounded-lg bg-danger text-white hover:bg-danger/90 disabled:opacity-60"
          @click="submitDeletion"
        >
          {{ isBusy ? '提交中…' : '申请注销账号' }}
        </button>
      </div>
    </div>

    <p v-if="inlineMessage" class="text-sm text-success mt-3">{{ inlineMessage }}</p>
    <p v-if="inlineError" class="text-sm text-danger mt-3" :data-traceId="traceId || undefined">{{ inlineError }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '@/utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { resolveAccountDeletionActionUrl } from '../utils/accountDeletionActionUrl.js'

const emit = defineEmits(['message', 'error'])

const blockerActionHref = (item) => resolveAccountDeletionActionUrl(item)

const isLoading = ref(true)
const isBusy = ref(false)
const blockers = ref([])
const cooldownDays = ref(15)
const status = ref('none')
const effectiveAt = ref('')
const confirmationText = ref('')
const password = ref('')
const inlineMessage = ref('')
const inlineError = ref('')
const traceId = ref('')

const effectiveAtLabel = computed(() => {
  if (!effectiveAt.value) return ''
  const d = new Date(effectiveAt.value)
  return Number.isNaN(d.getTime()) ? effectiveAt.value : d.toLocaleString()
})

const canSubmit = computed(() => confirmationText.value.trim() === '注销' && password.value.trim() !== '')

const loadPrecheck = async () => {
  const response = await apiFetch('/api/accounts/users/me/account-deletion/precheck/', {
    headers: { Accept: 'application/json' },
  })
  const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
  if (!response.ok) {
    traceId.value = tid
    throw new Error(data?.detail || data?.error || '无法检查注销条件')
  }
  blockers.value = Array.isArray(data.blockers) ? data.blockers.filter((b) => b?.blocking) : []
  cooldownDays.value = Number(data.cooldown_days) || 15
}

const loadStatus = async () => {
  const response = await apiFetch('/api/accounts/users/me/account-deletion/status/', {
    headers: { Accept: 'application/json' },
  })
  const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
  if (!response.ok) {
    traceId.value = tid
    throw new Error(data?.detail || data?.error || '无法获取注销状态')
  }
  status.value = data.status || 'none'
  effectiveAt.value = data.effective_at || ''
}

const refresh = async () => {
  isLoading.value = true
  inlineMessage.value = ''
  inlineError.value = ''
  traceId.value = ''
  try {
    await loadStatus()
    if (status.value === 'none') {
      await loadPrecheck()
    }
  } catch (error) {
    inlineError.value = humanizeRequestErrorMessage(error.message || '加载失败')
    emit('error', inlineError.value)
  } finally {
    isLoading.value = false
  }
}

const submitDeletion = async () => {
  isBusy.value = true
  inlineMessage.value = ''
  inlineError.value = ''
  traceId.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/me/account-deletion/request/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        confirmation_text: confirmationText.value.trim(),
        password: password.value,
      }),
    })
    const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      traceId.value = tid
      if (Array.isArray(data?.blockers)) {
        blockers.value = data.blockers.filter((b) => b?.blocking)
      }
      throw new Error(data?.detail || data?.error || '提交失败')
    }
    status.value = data.status || 'pending_cooldown'
    effectiveAt.value = data.effective_at || ''
    password.value = ''
    confirmationText.value = ''
    inlineMessage.value = '注销申请已提交'
    emit('message', inlineMessage.value)
  } catch (error) {
    inlineError.value = humanizeRequestErrorMessage(error.message || '提交失败')
    emit('error', inlineError.value)
  } finally {
    isBusy.value = false
  }
}

const cancelDeletion = async () => {
  isBusy.value = true
  inlineMessage.value = ''
  inlineError.value = ''
  traceId.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/me/account-deletion/cancel/', {
      method: 'POST',
      headers: { Accept: 'application/json' },
    })
    const { data, traceId: tid } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      traceId.value = tid
      throw new Error(data?.detail || data?.error || '撤回失败')
    }
    status.value = 'none'
    effectiveAt.value = ''
    inlineMessage.value = '已撤回注销申请'
    emit('message', inlineMessage.value)
    await loadPrecheck()
  } catch (error) {
    inlineError.value = humanizeRequestErrorMessage(error.message || '撤回失败')
    emit('error', inlineError.value)
  } finally {
    isBusy.value = false
  }
}

onMounted(refresh)
</script>
