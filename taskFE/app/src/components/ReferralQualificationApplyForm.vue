<template>
  <div class="space-y-3">
    <label class="block text-sm font-medium text-text" for="referral-legal-name">
      个人名称 <span class="text-red-500">*</span>
    </label>
    <p
      class="text-xs text-amber-800 leading-relaxed rounded-lg border border-amber-200 bg-amber-50 px-3 py-2"
      data-testid="referral-legal-name-warning"
    >
      请填写与<strong>微信实名认证</strong>完全一致的姓名。名称填错将导致分账失败，资金无法进入您的微信，且<strong>无法补分</strong>。
    </p>
    <input
      id="referral-legal-name"
      v-model="legalName"
      data-testid="referral-legal-name"
      type="text"
      maxlength="32"
      class="w-full px-3 py-2 border border-border rounded-lg text-sm"
      placeholder="与微信实名一致的姓名"
      :disabled="applying"
    />
    <label class="block text-sm font-medium text-text" for="referral-personal-intro">
      个人介绍 <span class="text-red-500">*</span>
    </label>
    <p class="text-xs text-text-light leading-relaxed">
      分成名额有限。请说明您的身份、使用平台的情况，以及希望如何推荐他人；审批将优先考虑活跃用户。
    </p>
    <textarea
      id="referral-personal-intro"
      v-model="personalIntro"
      data-testid="referral-personal-intro"
      class="w-full px-3 py-2 border border-border rounded-lg text-sm min-h-[8rem]"
      rows="5"
      maxlength="2000"
      placeholder="请填写至少 20 字的个人介绍"
      :disabled="applying"
    />
    <p class="text-xs" :class="countClass">{{ runeCount }} / 500（至少 20 字）</p>
    <div
      class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 space-y-2"
      data-testid="referral-identity-bind-notice"
    >
      <p class="text-xs text-amber-800 leading-relaxed">
        申请推荐资格时，系统将绑定您的<strong>账号标识</strong>（微信 OpenID）与<strong>用户身份信息</strong>（实名姓名，如已完成认证），并提交微信支付做一致性校验，以便向您分账收款。未授权则无法申请。
      </p>
      <label class="flex items-start gap-2 cursor-pointer select-none">
        <input
          id="referral-identity-bind-consent"
          v-model="identityBindConsent"
          data-testid="referral-identity-bind-consent"
          type="checkbox"
          class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
          :disabled="applying"
        />
        <span class="text-xs text-text leading-relaxed">
          我已阅读并同意绑定账号标识与用户身份信息，授权向微信支付传输上述信息做一致性校验。
        </span>
      </label>
    </div>
    <button
      type="button"
      class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
      data-testid="referral-apply-submit"
      :disabled="!canSubmit || applying"
      :aria-busy="applying ? 'true' : 'false'"
      @click="onSubmit"
    >
      {{ applying ? '申请中...' : submitLabel }}
    </button>
    <p v-if="policyMessage" class="text-xs text-text-light">{{ policyMessage }}</p>
    <p v-if="error" class="text-sm text-red-500" :data-traceId="errorTraceId || undefined">{{ error }}</p>
    <p v-if="success" class="text-sm text-green-600">{{ success }}</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { createClickGuard } from '../utils/clickGuard.js'

const props = defineProps({
  submitLabel: { type: String, default: '申请推荐资格' },
  applying: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
  success: { type: String, default: '' },
  policyMessage: { type: String, default: '' },
  apply: { type: Function, required: true },
})

const personalIntro = ref('')
const legalName = ref('')
const identityBindConsent = ref(false)
const applyGuard = createClickGuard()

const runeCount = computed(() => Array.from(personalIntro.value.trim()).length)
const legalNameCount = computed(() => Array.from(legalName.value.trim()).length)
const canSubmit = computed(() => (
  runeCount.value >= 20 && runeCount.value <= 500
  && legalNameCount.value >= 2 && legalNameCount.value <= 32
  && identityBindConsent.value
))
const countClass = computed(() => {
  if (runeCount.value === 0) return 'text-text-light'
  if (runeCount.value < 20 || runeCount.value > 500) return 'text-red-500'
  return 'text-text-light'
})

const onSubmit = async () => {
  if (!canSubmit.value || props.applying) return
  await applyGuard.run(async ({ idempotencyKey }) => {
    await props.apply({
      personalIntro: personalIntro.value.trim(),
      legalName: legalName.value.trim(),
      identityBindConsent: identityBindConsent.value,
      idempotencyKey,
    })
  })
}
</script>
