<template>
  <div data-testid="system-admin-referral-management" class="p-8">
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-gray-900">推荐资格管理</h1>
      <p class="text-gray-600 mt-2">
        管理推荐资格管控策略、固定分成比例与佣金入账周期。推荐码申请审批请前往
        <a href="/system-admin/users/?tab=referral-apps" class="text-primary hover:underline">用户列表 → 推荐码申请</a>
        。
      </p>
    </div>

    <!-- 策略配置 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">管控策略</h2>
          <p class="text-sm text-gray-500 mt-1">选择限额放开（自动开通）或审批模式（管理员逐条审核）。</p>
        </div>
        <button
          type="button"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
          :disabled="loadingPolicy || savingPolicy"
          @click="savePolicy"
        >
          {{ savingPolicy ? '保存中...' : '保存策略' }}
        </button>
      </div>

      <div class="space-y-5">
        <div class="flex items-center gap-4">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="policyMode"
              type="radio"
              value="open"
              class="w-4 h-4 text-primary"
              :disabled="loadingPolicy || savingPolicy"
            >
            <span class="text-sm font-medium text-gray-900">限额放开</span>
            <span class="text-xs text-gray-500">用户申请即自动开通，无需审批</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="policyMode"
              type="radio"
              value="approval"
              class="w-4 h-4 text-primary"
              :disabled="loadingPolicy || savingPolicy"
            >
            <span class="text-sm font-medium text-gray-900">审批模式</span>
            <span class="text-xs text-gray-500">用户提交申请后需管理员审批</span>
          </label>
        </div>

        <div>
          <label for="policy-message" class="block text-sm font-medium text-gray-900 mb-1">
            策略提示（显示给用户）
          </label>
          <input
            id="policy-message"
            v-model="policyMessageText"
            type="text"
            class="w-full max-w-lg px-3 py-2 border border-gray-300 rounded-lg"
            placeholder="如：推荐码每日限量 100 个"
            :disabled="loadingPolicy || savingPolicy"
          >
        </div>
      </div>

      <div v-if="loadingPolicy" class="text-sm text-gray-500 mt-4">策略加载中...</div>
      <div v-if="policyError" class="mt-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm" :data-traceId="policyErrorTraceId || undefined">{{ policyError }}</div>
      <div v-if="policySuccess" class="mt-4 p-3 bg-green-50 text-green-700 rounded-lg text-sm">{{ policySuccess }}</div>
    </div>

    <!-- Anti-Replay-OK: 只读展示固定 5%，无写操作 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8" data-testid="system-admin-referral-ratio-card">
      <div class="mb-4">
        <h2 class="text-lg font-semibold text-gray-900">分成比例</h2>
        <p class="text-sm text-gray-500 mt-1">
          分成比例固定为 5%。点数计提与微信打款均使用该比例（打款不超过商户上限）。
        </p>
      </div>
      <p class="text-2xl font-semibold text-gray-900" data-testid="referral-rate-fixed">5%</p>
    </div>

    <!-- 入账周期配置 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">佣金入账周期</h2>
          <p class="text-sm text-gray-500 mt-1">
            推荐佣金在用户消费后延迟入账的天数。当前：{{ settleDelayDays }} 天（最低 8 天）。
          </p>
        </div>
      </div>
      <div class="flex items-center gap-4">
        <input
          v-model.number="settleDelayInput"
          type="number"
          min="8"
          step="1"
          class="w-24 px-3 py-2 border border-gray-300 rounded-lg text-sm"
          :disabled="loadingSettle || savingSettle"
        >
        <span class="text-sm text-gray-500">天</span>
        <button
          class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90 disabled:opacity-50"
          :disabled="loadingSettle || savingSettle || settleDelayInput < 8"
          @click="saveSettleConfig"
        >
          {{ savingSettle ? '保存中...' : '保存' }}
        </button>
      </div>
      <div v-if="settleError" class="mt-3 p-3 bg-red-50 text-red-700 rounded-lg text-sm" :data-traceId="settleErrorTraceId || undefined">{{ settleError }}</div>
      <div v-if="settleSuccess" class="mt-3 p-3 bg-green-50 text-green-700 rounded-lg text-sm">{{ settleSuccess }}</div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { messageFromFailedResponse } from '../utils/httpError.js'

const policyGuard = createClickGuard()
const settleGuard = createClickGuard()

const loadingPolicy = ref(false)
const savingPolicy = ref(false)
const policyError = ref('')
const policyErrorTraceId = ref('')
const policySuccess = ref('')
const policyMode = ref('approval')
const policyMessageText = ref('')

const loadingSettle = ref(false)
const savingSettle = ref(false)
const settleError = ref('')
const settleErrorTraceId = ref('')
const settleSuccess = ref('')
const settleDelayDays = ref(15)
const settleDelayInput = ref(15)

const loadPolicy = async () => {
  loadingPolicy.value = true
  policyError.value = ''
  policyErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/system-admin/referral/policy/', {
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      policyError.value = messageFromFailedResponse(response, '加载策略失败')
      policyErrorTraceId.value = extractTraceId(response) || ''
      return
    }
    const data = await response.json().catch(() => ({}))
    const p = data.data || data
    policyMode.value = p.mode || 'approval'
    policyMessageText.value = p.message || ''
  } catch (error) {
    policyError.value = error?.message || '加载策略失败'
    policyErrorTraceId.value = extractTraceId(error) || ''
  } finally {
    loadingPolicy.value = false
  }
}

const savePolicy = async () => {
  await policyGuard.run(async ({ idempotencyKey }) => {
    savingPolicy.value = true
    policyError.value = ''
    policyErrorTraceId.value = ''
    policySuccess.value = ''
    try {
      const response = await apiFetch('/api/system-admin/referral/policy/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          Accept: 'application/json',
        }, idempotencyKey),
        body: JSON.stringify({
          mode: policyMode.value,
          message: policyMessageText.value,
        }),
      })
      if (!response.ok) {
        policyError.value = messageFromFailedResponse(response, '保存策略失败')
        policyErrorTraceId.value = extractTraceId(response) || ''
        return
      }
      const data = await response.json().catch(() => ({}))
      policySuccess.value = '策略保存成功'
      setTimeout(() => { policySuccess.value = '' }, 3000)
    } catch (error) {
      policyError.value = error?.message || '保存策略失败'
      policyErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      savingPolicy.value = false
    }
  })
}

const loadSettleConfig = async () => {
  loadingSettle.value = true
  settleError.value = ''
  settleErrorTraceId.value = ''
  try {
    const resp = await apiFetch('/api/system-admin/referral/config/', {
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) {
      settleError.value = messageFromFailedResponse(resp, '加载配置失败')
      settleErrorTraceId.value = extractTraceId(resp) || ''
      return
    }
    const json = await resp.json().catch(() => ({}))
    const d = json.data || json
    settleDelayDays.value = d.settle_delay_days || 15
    settleDelayInput.value = d.settle_delay_days || 15
  } catch (error) {
    settleError.value = error?.message || '加载配置失败'
    settleErrorTraceId.value = extractTraceId(error) || ''
  } finally {
    loadingSettle.value = false
  }
}

const saveSettleConfig = async () => {
  await settleGuard.run(async ({ idempotencyKey }) => {
    savingSettle.value = true
    settleError.value = ''
    settleErrorTraceId.value = ''
    settleSuccess.value = ''
    try {
      const resp = await apiFetch('/api/system-admin/referral/config/update/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          Accept: 'application/json',
        }, idempotencyKey),
        body: JSON.stringify({ settle_delay_days: settleDelayInput.value }),
      })
      if (!resp.ok) {
        settleError.value = messageFromFailedResponse(resp, '保存配置失败')
        settleErrorTraceId.value = extractTraceId(resp) || ''
        return
      }
      const json = await resp.json().catch(() => ({}))
      const d = json.data || json
      settleDelayDays.value = d.settle_delay_days || settleDelayInput.value
      settleDelayInput.value = settleDelayDays.value
      settleSuccess.value = '入账周期已更新'
      setTimeout(() => { settleSuccess.value = '' }, 3000)
    } catch (error) {
      settleError.value = error?.message || '保存配置失败'
      settleErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      savingSettle.value = false
    }
  })
}

onMounted(() => {
  loadPolicy()
  loadSettleConfig()
})
</script>
