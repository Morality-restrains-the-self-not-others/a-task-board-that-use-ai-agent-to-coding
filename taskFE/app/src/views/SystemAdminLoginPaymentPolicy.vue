<template>
  <div data-alias="view-system-admin-login-payment-policy" class="p-6">
    <!-- 面包屑导航 -->
    <nav class="flex items-center space-x-2 text-sm text-gray-500 mb-4 py-3 px-1" aria-label="面包屑">
      <router-link to="/system-admin/" class="hover:text-primary transition-colors">系统管理</router-link>
      <span class="text-gray-300">/</span>
      <router-link to="/system-admin/" class="hover:text-primary transition-colors">安全策略</router-link>
      <span class="text-gray-300">/</span>
      <span class="text-gray-700 font-medium">登录与支付策略</span>
    </nav>

    <!-- 页面标题 -->
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-gray-900">登录与支付策略</h1>
      <p class="text-gray-600 mt-2">控制微信登录、邮箱注册、手机号登录与支付前手机号验证策略。</p>
    </div>

    <!-- 登录与支付策略 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">策略配置</h2>
          <p class="text-sm text-gray-500 mt-1">控制邮箱注册、手机号登录与支付前手机号验证策略。</p>
        </div>
        <button
          type="button"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
          :disabled="loadingPolicy || savingPolicy"
          @click="saveFeaturePolicy"
        >
          {{ savingPolicy ? '保存中...' : '保存策略' }}
        </button>
      </div>

      <div class="space-y-5">
        <div class="flex items-center justify-between gap-4">
          <div>
            <label for="enable-email-register-switch" class="block text-sm font-medium text-gray-900">
              邮箱注册
            </label>
            <p class="text-sm text-gray-500 mt-1">关闭后，用户不可使用邮箱注册新账户。</p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input
              id="enable-email-register-switch"
              v-model="featurePolicy.enable_email_register"
              data-testid="enable-email-register-switch"
              type="checkbox"
              class="sr-only peer"
              :disabled="loadingPolicy || savingPolicy"
            >
            <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
          </label>
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <label for="enable-wechat-login-switch" class="block text-sm font-medium text-gray-900">
              微信登录
            </label>
            <p class="text-sm text-gray-500 mt-1">开启后，用户可使用微信扫码登录。默认为开启状态。</p>
            <p
              v-if="wechatLoginUnconfiguredHint"
              class="text-sm text-amber-600 mt-1"
              data-testid="wechat-unconfigured-hint"
            >
              微信应用未配置，扫码入口暂不可用（请先在 conf 中配置 wechat.apps）
            </p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input
              id="enable-wechat-login-switch"
              v-model="featurePolicy.enable_wechat_login"
              data-testid="enable-wechat-login-switch"
              type="checkbox"
              class="sr-only peer"
              :disabled="loadingPolicy || savingPolicy"
            >
            <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
          </label>
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <label for="enable-phone-login-switch" class="block text-sm font-medium text-gray-900">
              手机号登录
            </label>
            <p class="text-sm text-gray-500 mt-1">关闭后，用户不可使用手机号快捷登录。</p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input
              id="enable-phone-login-switch"
              v-model="featurePolicy.enable_phone_login"
              data-testid="enable-phone-login-switch"
              type="checkbox"
              class="sr-only peer"
              :disabled="loadingPolicy || savingPolicy"
            >
            <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
          </label>
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <label
              for="enable-recharge-phone-verification-switch"
              class="block text-sm font-medium text-gray-900"
            >
              支付前手机号验证
            </label>
            <p class="text-sm text-gray-500 mt-1">开启后，用户支付前需完成手机号验证。</p>
            <p v-if="rechargePhoneVerificationDisabled" class="text-sm text-amber-600 mt-1">
              需先开启手机号登录
            </p>
          </div>
          <label
            class="relative inline-flex items-center"
            :class="rechargePhoneVerificationDisabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'"
          >
            <input
              id="enable-recharge-phone-verification-switch"
              v-model="featurePolicy.enable_recharge_phone_verification"
              data-testid="enable-recharge-phone-verification-switch"
              type="checkbox"
              class="sr-only peer"
              :disabled="loadingPolicy || savingPolicy || rechargePhoneVerificationDisabled"
            >
            <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
          </label>
        </div>
        <!-- 手机号开放区域 -->
        <div class="pt-5 border-t border-gray-100">
          <div class="mb-3">
            <label class="block text-sm font-medium text-gray-900">
              允许的手机号区域
            </label>
            <p class="text-sm text-gray-500 mt-1">选择允许注册和登录的手机号国家/地区。留空表示允许所有地区。</p>
            <p v-if="!featurePolicy.enable_phone_login" class="text-sm text-amber-600 mt-1">
              手机号登录未开启时，区域设置不生效
            </p>
          </div>

          <!-- Selected tags -->
          <div v-if="featurePolicy.allowed_phone_country_codes.length" class="flex flex-wrap gap-2 mb-3">
            <span
              v-for="code in featurePolicy.allowed_phone_country_codes"
              :key="code"
              class="inline-flex items-center gap-1 px-3 py-1.5 bg-primary/10 text-primary text-sm rounded-full"
            >
              {{ getCountryName(code) || code }}
              <button
                type="button"
                class="ml-0.5 text-primary/60 hover:text-red-500 focus:outline-none"
                :disabled="loadingPolicy || savingPolicy"
                @click="removeCountryCode(code)"
                :title="'移除 ' + code"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </span>
          </div>
          <p v-else class="text-sm text-gray-400 mb-3">当前允许所有地区</p>

          <!-- Add dropdown -->
          <div class="relative" v-if="availableCountryCodes.length">
            <select
              class="w-full max-w-xs border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
              :disabled="loadingPolicy || savingPolicy"
              @change="addCountryCode($event)"
            >
              <option value="">+ 添加区域</option>
              <option
                v-for="opt in availableCountryCodes"
                :key="opt.code"
                :value="opt.code"
              >
                {{ opt.name }} ({{ opt.code }})
              </option>
            </select>
          </div>
        </div>
      </div>

      <div v-if="loadingPolicy" class="text-sm text-gray-500 mt-4">策略加载中...</div>
      <div v-if="policyErrorMessage" class="mt-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm" :data-traceId="policyErrorTraceId || undefined">
        {{ policyErrorMessage }}
      </div>
      <div v-if="policySuccessMessage" class="mt-4 p-3 bg-green-50 text-green-700 rounded-lg text-sm">
        {{ policySuccessMessage }}
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:view-system-admin-login-payment-policy */
import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { COUNTRY_DIAL_OPTIONS } from '../utils/countryDialCodes.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const FEATURE_POLICY_API = '/api/system-admin/system-feature-policy/'

const loadingPolicy = ref(false)
const savingPolicy = ref(false)
const policyErrorMessage = ref('')
const policyErrorTraceId = ref('')
const policySuccessMessage = ref('')

// 首帧占位默认值（后端 API 响应前的展示态，与后端产品默认一致）：
// 邮箱注册=关、微信登录=开、手机号区域=仅 +86（中国）。
const featurePolicy = ref({
  enable_phone_login: false,
  enable_recharge_phone_verification: false,
  enable_email_register: false,
  enable_wechat_login: true,
  allowed_phone_country_codes: ['+86'],
  wechat_app_configured: true,
})

const rechargePhoneVerificationDisabled = computed(() => !featurePolicy.value.enable_phone_login)

// OPT-20260806-029: 开关开启但微信应用未配置时提示（登录页入口会隐藏，管理员困惑）
const wechatLoginUnconfiguredHint = computed(
  () => featurePolicy.value.enable_wechat_login && !featurePolicy.value.wechat_app_configured,
)

const extractErrorMessage = (data, fallback) => {
  if (!data) return fallback
  if (typeof data.message === 'string' && data.message.trim()) return data.message.trim()
  if (typeof data.detail === 'string' && data.detail.trim()) return data.detail.trim()
  if (typeof data.error === 'string' && data.error.trim()) return data.error.trim()
  if (Array.isArray(data.non_field_errors) && data.non_field_errors.length) {
    return String(data.non_field_errors[0])
  }
  return fallback
}

const normalizeFeaturePolicy = (policy) => ({
  enable_phone_login: Boolean(policy?.enable_phone_login),
  enable_recharge_phone_verification:
    Boolean(policy?.enable_phone_login) && Boolean(policy?.enable_recharge_phone_verification),
  enable_email_register: policy?.enable_email_register !== false,
  enable_wechat_login: Boolean(policy?.enable_wechat_login),
  allowed_phone_country_codes: Array.isArray(policy?.allowed_phone_country_codes)
    ? policy.allowed_phone_country_codes
    : [],
  // OPT-20260806-029: 后端 admin 端点返回；旧后端无该字段时默认 true（不误报）
  wechat_app_configured: policy?.wechat_app_configured !== false,
})

// Available country codes not yet selected
const availableCountryCodes = computed(() =>
  COUNTRY_DIAL_OPTIONS.filter(
    (opt) => !featurePolicy.value.allowed_phone_country_codes.includes(opt.code)
  )
)

function getCountryName(code) {
  const opt = COUNTRY_DIAL_OPTIONS.find((o) => o.code === code)
  return opt ? opt.name : null
}

function addCountryCode(event) {
  const code = event.target.value
  if (!code) return
  if (!featurePolicy.value.allowed_phone_country_codes.includes(code)) {
    featurePolicy.value.allowed_phone_country_codes = [
      ...featurePolicy.value.allowed_phone_country_codes,
      code,
    ]
  }
  event.target.value = '' // reset select
}

function removeCountryCode(code) {
  featurePolicy.value.allowed_phone_country_codes =
    featurePolicy.value.allowed_phone_country_codes.filter((c) => c !== code)
}

watch(
  () => featurePolicy.value.enable_phone_login,
  (enabled) => {
    if (!enabled) {
      featurePolicy.value.enable_recharge_phone_verification = false
    }
  }
)

const loadFeaturePolicy = async () => {
  loadingPolicy.value = true
  policyErrorMessage.value = ''
  policyErrorTraceId.value = ''
  policySuccessMessage.value = ''
  try {
    const response = await apiFetch(FEATURE_POLICY_API, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      policyErrorMessage.value = extractErrorMessage(data, `策略加载失败（${response.status}）`)
      policyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      return
    }
    featurePolicy.value = normalizeFeaturePolicy(data?.data || data)
  } catch (error) {
    policyErrorMessage.value = error?.message || '策略加载失败'
    policyErrorTraceId.value = extractTraceId(error) || ''
  } finally {
    loadingPolicy.value = false
  }
}

const saveFeaturePolicyGuard = createClickGuard()

const saveFeaturePolicy = async () => {
  // OPT-20260819-038: 连点/超时重试会双发 POST — createClickGuard 在途锁 +
  // debounce + Idempotency-Key（后端幂等去重）
  await saveFeaturePolicyGuard.run(async ({ idempotencyKey }) => {
    policyErrorMessage.value = ''
    policyErrorTraceId.value = ''
    policySuccessMessage.value = ''
    savingPolicy.value = true
    try {
      const payload = normalizeFeaturePolicy(featurePolicy.value)
      const response = await apiFetch(FEATURE_POLICY_API, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(payload),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        policyErrorMessage.value = extractErrorMessage(data, `保存失败（${response.status}）`)
        policyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      featurePolicy.value = normalizeFeaturePolicy(data?.data || payload)
      policySuccessMessage.value =
        (typeof data?.message === 'string' && data.message.trim()) ? data.message.trim() : '保存成功'
    } catch (error) {
      policyErrorMessage.value = error?.message || '保存失败'
      policyErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      savingPolicy.value = false
    }
  })
}

onMounted(() => {
  loadFeaturePolicy()
})
</script>

<style scoped>
/* 组件内样式可以在这里添加 */
</style>
