<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="referral" class="shrink-0" />

      <main class="flex-1 max-w-3xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">推荐</h1>
          <p class="text-sm text-text-light">为不同渠道创建推荐码，按渠道与时间查看推荐人数和分账。</p>
          <ReferralRateChangeNotice class="mt-2" />
        </div>

        <ReferralChannelPanel
          :fallback-code="referralCode"
          @channels-loaded="onChannelsLoaded"
        />

        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6 space-y-4">
          <div v-if="statusLoading" class="text-sm text-text-light">加载推荐资格状态...</div>
          <div v-else-if="hasActiveQualification && expiresSoon" class="bg-amber-50 border border-amber-300 rounded-lg p-3">
            <span class="text-sm text-amber-700 font-medium">推荐资格即将过期（剩余 {{ expiresInDays }} 天）</span>
            <p class="text-xs text-amber-600 mt-1" data-testid="referral-expiry-notice">过期后，被推荐用户下单将<strong>不再产生收益分成</strong>。请及时续期。</p>
          </div>
          <div v-else-if="hasActiveQualification" class="bg-green-50 border border-green-200 rounded-lg p-3">
            <span class="text-sm text-green-700 font-medium">已获得推荐资格</span>
            <span v-if="expiresInDays" class="text-xs text-green-600">（剩余 {{ expiresInDays }} 天）</span>
            <p class="text-xs text-green-600 mt-1">通过您的推荐码注册并消费的用户将为您带来 <strong data-testid="referral-rate-copy">{{ referralRateDisplay || '—' }}</strong> 的收益分成。</p>
            <ReferralRateChangeNotice class="mt-1" />
            <p
              v-if="wechatReceiverStatus === 'pending_openid'"
              class="text-xs text-amber-700 mt-2"
              data-testid="wechat-receiver-pending-openid"
            >尚未绑定微信登录，无法在微信分账后台登记为接收方。请先用微信扫码登录后再刷新本页。</p>
            <p
              v-else-if="wechatReceiverStatus === 'failed'"
              class="text-xs text-amber-700 mt-2"
              data-testid="wechat-receiver-failed"
            >微信分账接收方登记失败，刷新本页将重试。{{ wechatReceiverReason ? '（' + wechatReceiverReason + '）' : '' }}</p>
            <p
              v-else-if="wechatReceiverStatus === 'skipped_not_live'"
              class="text-xs text-amber-700 mt-2"
              data-testid="wechat-receiver-skipped"
            >微信支付未处于 live 模式，暂未向商户平台登记。配置完成后刷新本页将自动登记。</p>
            <p
              v-else-if="wechatReceiverStatus === 'registered'"
              class="text-xs text-green-700 mt-2"
              data-testid="wechat-receiver-registered"
            >已在微信分账后台登记为接收方。</p>
          </div>
          <div v-else class="bg-amber-50 border border-amber-200 rounded-lg p-3">
            <span class="text-sm text-amber-700 font-medium">暂无推荐资格</span>
            <p class="text-xs text-amber-600 mt-1" data-testid="referral-no-qualification-notice">
              <strong>注意：</strong>您当前暂无推荐资格，被推荐用户此时下单不会产生分成。申请通过后，已推荐用户的后续下单仍可为您分成。请前往下方「推荐资格」区域申请。
            </p>
          </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h2 class="text-lg font-semibold text-text mb-4">推荐资格</h2>
          <ReferralProfitSharingConfirmNotice class="mb-4" />
          <p v-if="statusLoading" class="text-sm text-text-light">加载推荐资格状态...</p>
          <div v-else-if="hasActiveQualification" class="space-y-3">
            <div class="flex items-center gap-3">
              <span class="px-3 py-1 bg-green-100 text-green-700 rounded-full text-xs font-medium">已获得推荐资格</span>
              <p v-if="expiresInDays" class="text-sm text-text-light">
                有效期至 {{ expiresAtFormatted }}（剩余 {{ expiresInDays }} 天）
              </p>
            </div>
            <!-- OPT-20260823-048: 存量已获资格用户补填个人名称，确保微信分账接收方带 Name -->
            <div
              v-if="!legalName"
              class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 space-y-3"
              data-testid="referral-legal-name-backfill"
            >
              <p class="text-xs text-amber-800 leading-relaxed" data-testid="referral-legal-name-backfill-warning">
                请补填与<strong>微信实名认证</strong>完全一致的姓名。名称填错将导致分账失败，资金无法进入您的微信，且<strong>无法补分</strong>。
              </p>
              <input
                id="referral-legal-name-backfill-input"
                v-model="legalNameBackfill"
                data-testid="referral-legal-name-backfill-input"
                type="text"
                maxlength="32"
                class="w-full px-3 py-2 border border-border rounded-lg text-sm"
                placeholder="与微信实名一致的姓名"
                :disabled="legalNameUpdating"
              />
              <button
                type="button"
                class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                data-testid="referral-legal-name-backfill-submit"
                :disabled="!legalNameBackfillCanSubmit || legalNameUpdating"
                :aria-busy="legalNameUpdating ? 'true' : 'false'"
                @click="onSubmitLegalName"
              >
                {{ legalNameUpdating ? '保存中...' : '保存个人名称' }}
              </button>
            </div>
            <p v-if="legalNameUpdateError" class="text-sm text-red-500" :data-traceId="legalNameUpdateErrorTraceId || undefined" data-testid="referral-legal-name-backfill-error">{{ legalNameUpdateError }}</p>
            <p v-if="legalNameUpdateSuccess" class="text-sm text-green-600" data-testid="referral-legal-name-backfill-success">{{ legalNameUpdateSuccess }}</p>
          </div>
          <div v-else-if="applicationStatus === 'pending'" class="space-y-3">
            <span class="px-3 py-1 bg-yellow-100 text-yellow-700 rounded-full text-xs font-medium">资格审批中</span>
            <p class="text-sm text-text-light">您的推荐资格申请正在等待管理员审核，审核通过后将自动开通。</p>
            <p v-if="submittedIntro" class="text-sm text-text bg-gray-50 border border-border rounded-lg p-3 whitespace-pre-wrap" data-testid="referral-submitted-intro">{{ submittedIntro }}</p>
          </div>
          <div v-else-if="applicationStatus === 'expired' || (applicationStatus === 'approved' && !hasActiveQualification)" class="space-y-3">
            <span class="px-3 py-1 bg-gray-100 text-gray-600 rounded-full text-xs font-medium">资格已过期</span>
            <p class="text-sm text-text-light">您的推荐资格已超过 6 个月有效期，需要重新申请。</p>
            <ReferralServiceAccountFollowGate
              :bound="serviceAccountBound"
              :busy="followChecking"
              :error="followError"
              :error-trace-id="followErrorTraceId"
              @confirm="onConfirmFollowed"
            >
              <ReferralQualificationApplyForm
                submit-label="重新申请"
                :applying="applying"
                :error="applyError"
                :error-trace-id="applyErrorTraceId"
                :success="applySuccess"
                :policy-message="policyMessage"
                :apply="submitApply"
              />
            </ReferralServiceAccountFollowGate>
          </div>
          <div v-else-if="applicationStatus === 'rejected'" class="space-y-3">
            <span class="px-3 py-1 bg-red-100 text-red-500 rounded-full text-xs font-medium">申请被拒绝</span>
            <p v-if="lastRejected" class="text-sm text-red-500">拒绝原因：{{ lastRejected }}</p>
            <p v-if="canReapplyAfter" class="text-sm text-text-light">可在 {{ canReapplyAfter }} 后重新申请</p>
            <ReferralServiceAccountFollowGate
              v-if="!canReapplyAfter"
              :bound="serviceAccountBound"
              :busy="followChecking"
              :error="followError"
              :error-trace-id="followErrorTraceId"
              @confirm="onConfirmFollowed"
            >
              <ReferralQualificationApplyForm
                submit-label="重新申请"
                :applying="applying"
                :error="applyError"
                :error-trace-id="applyErrorTraceId"
                :success="applySuccess"
                :policy-message="policyMessage"
                :apply="submitApply"
              />
            </ReferralServiceAccountFollowGate>
          </div>
          <div v-else class="space-y-3">
            <span class="px-3 py-1 bg-gray-100 text-gray-600 rounded-full text-xs font-medium">未获得推荐资格</span>
            <p class="text-sm text-text-light">
              您尚未申请推荐资格。拥有推荐资格后，您推荐的用户消费时将为您带来 <strong class="text-primary" data-testid="referral-rate-apply-copy">{{ referralRateDisplay || '—' }}</strong> 的收益分成。
            </p>
            <ReferralRateChangeNotice />
            <ReferralServiceAccountFollowGate
              :bound="serviceAccountBound"
              :busy="followChecking"
              :error="followError"
              :error-trace-id="followErrorTraceId"
              @confirm="onConfirmFollowed"
            >
              <ReferralQualificationApplyForm
                submit-label="申请推荐资格"
                :applying="applying"
                :error="applyError"
                :error-trace-id="applyErrorTraceId"
                :success="applySuccess"
                :policy-message="policyMessage"
                :apply="submitApply"
              />
            </ReferralServiceAccountFollowGate>
          </div>
        </div>

        <ReferralStatsPanel
          :user-id="resolvedUserId"
          :has-active-qualification="hasActiveQualification"
          :channels="channelOptions"
        />

        <ReferralProfitSharingOrdersPanel />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { messageFromFailedResponse } from '../utils/httpError.js'
import { getStoredUserId, resolveAuthenticatedUserId } from '../utils/sessionUserIdUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { normalizeUrlAccessCodeToOwn } from '../utils/referralAccessCodeUtils.js'

import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import ReferralQualificationApplyForm from '../components/ReferralQualificationApplyForm.vue'
import ReferralServiceAccountFollowGate from '../components/ReferralServiceAccountFollowGate.vue'
import ReferralChannelPanel from '../components/ReferralChannelPanel.vue'
import ReferralStatsPanel from '../components/ReferralStatsPanel.vue'
import ReferralRateChangeNotice from '../components/ReferralRateChangeNotice.vue'
import ReferralProfitSharingConfirmNotice from '../components/ReferralProfitSharingConfirmNotice.vue'
import ReferralProfitSharingOrdersPanel from '../components/ReferralProfitSharingOrdersPanel.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))

const statusLoading = ref(true)
const applying = ref(false)
const applyError = ref('')
const applyErrorTraceId = ref('')
const applySuccess = ref('')
const legalName = ref('')
const legalNameBackfill = ref('')
const legalNameUpdating = ref(false)
const legalNameUpdateError = ref('')
const legalNameUpdateErrorTraceId = ref('')
const legalNameUpdateSuccess = ref('')
const legalNameUpdateGuard = createClickGuard()
const applicationStatus = ref(null)
const hasActiveQualification = ref(false)
const expiresAt = ref(null)
const expiresInDays = ref(null)
const expiresSoon = computed(() => hasActiveQualification.value && expiresInDays.value != null && expiresInDays.value <= 30)
const lastRejected = ref('')
const canReapplyAfter = ref(null)
const policyMessage = ref('')
const submittedIntro = ref('')
const resolvedUserId = ref(getStoredUserId())
const apiAccessCode = ref('')
const channelOptions = ref([])
const referralRateDisplay = ref('')
const wechatReceiverStatus = ref('')
const wechatReceiverReason = ref('')
const serviceAccountBound = ref(false)
const followChecking = ref(false)
const followError = ref('')
const followErrorTraceId = ref('')

const referralCode = computed(() => String(apiAccessCode.value || '').trim())
const legalNameBackfillCanSubmit = computed(() => {
  const n = Array.from(legalNameBackfill.value.trim())
  return n.length >= 2 && n.length <= 32
})
const expiresAtFormatted = computed(() => {
  if (!expiresAt.value) return ''
  const d = new Date(expiresAt.value)
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
})

const onChannelsLoaded = (list) => {
  channelOptions.value = Array.isArray(list) ? list : []
}

// OPT-20260824-005: 地址栏 ?accessCode= 是分享链接溯源参数（他人推荐码，可能已删除/失效），
// 与本页展示的「自己的推荐码」语义不同。登录态统一由共享工具 normalizeUrlAccessCodeToOwn
// 归一化（main.js 全局 afterEach 已覆盖各页，本页传 ownCode 复用刚取到的码，避免重复请求）。

const fetchReferralStatus = async () => {
  statusLoading.value = true
  try {
    const response = await apiFetch(
      '/api/accounts/users/referral-codes/status/',
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      statusLoading.value = false
      return
    }
    const data = await response.json()
    apiAccessCode.value = String(data.access_code || '').trim()
    applicationStatus.value = data.application_status || null
    hasActiveQualification.value = data.has_active_code || false
    expiresAt.value = data.expires_at || null
    expiresInDays.value = data.expires_in_days || null
    if (data.application_status === 'rejected') {
      lastRejected.value = data.reject_reason || '管理员拒绝了您的申请'
      canReapplyAfter.value = data.can_reapply_after || null
    }
    if (data.policy_message) {
      policyMessage.value = data.policy_message
    }
    submittedIntro.value = String(data.personal_intro || '').trim()
    referralRateDisplay.value = String(data.referral_rate_display || '').trim()
    wechatReceiverStatus.value = String(data.wechat_receiver_status || '').trim()
    wechatReceiverReason.value = String(data.wechat_receiver_reason || '').trim()
    legalName.value = String(data.legal_name || '').trim()
    serviceAccountBound.value = Boolean(data.service_account_bound)
  } catch (error) {
    console.error('获取推荐资格状态失败:', error)
  } finally {
    statusLoading.value = false
  }
}

const onConfirmFollowed = async () => {
  followChecking.value = true
  followError.value = ''
  followErrorTraceId.value = ''
  try {
    const response = await apiFetch(
      '/api/auth/wechat/mp/follow-status/',
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      followError.value = messageFromFailedResponse(response, '确认关注失败，请稍后重试')
      followErrorTraceId.value = extractTraceId(response)
      return
    }
    const data = await response.json()
    if (data && data.bound) {
      serviceAccountBound.value = true
      followError.value = ''
      return
    }
    if (data && data.ticket_status === 'conflict') {
      followError.value = data.message || '该微信已绑定其他账号。请用已绑定该微信的账号登录后再扫码。'
      return
    }
    if (data && data.ticket_status === 'expired') {
      followError.value = data.message || '二维码已过期，请刷新页面重新获取。'
      return
    }
    if (data && data.has_unionid === false) {
      followError.value = '当前账号尚未绑定微信登录。请先点「绑定微信登录」，再用同一微信关注服务号后点「我已关注」。'
      return
    }
    followError.value = '尚未确认关注。请用微信再扫一次本页服务号二维码（已关注也会推送 SCAN），然后点「我已关注」。'
  } catch (error) {
    followError.value = '确认关注失败，请稍后重试'
    followErrorTraceId.value = extractTraceId(error)
  } finally {
    followChecking.value = false
  }
}

const submitApply = async ({ personalIntro, legalName, identityBindConsent, idempotencyKey }) => {
  applying.value = true
  applyError.value = ''
  applyErrorTraceId.value = ''
  applySuccess.value = ''
  try {
    const response = await apiFetch(
      '/api/accounts/users/referral-codes/apply/',
      {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json', 'Content-Type': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({
          personal_intro: personalIntro,
          legal_name: legalName,
          identity_bind_consent: Boolean(identityBindConsent),
        }),
      },
    )
    if (!response.ok) {
      applyError.value = messageFromFailedResponse(response, '申请失败，请稍后重试')
      applyErrorTraceId.value = extractTraceId(response)
      return
    }
    const data = await response.json()
    applyErrorTraceId.value = ''
    submittedIntro.value = personalIntro
    if (data.status === 'approved') {
      applySuccess.value = data.message || '推荐资格已开通'
      hasActiveQualification.value = true
      applicationStatus.value = 'approved'
      expiresAt.value = data.expires_at
      await fetchReferralStatus()
    } else if (data.status === 'pending') {
      applySuccess.value = data.message || '申请已提交'
      applicationStatus.value = 'pending'
    }
  } catch (error) {
    console.error('申请推荐资格失败:', error)
    applyError.value = '网络错误，请稍后重试'
    applyErrorTraceId.value = extractTraceId(error)
  } finally {
    applying.value = false
  }
}

// OPT-20260823-048: 存量已获资格用户补填个人名称，更新后重新 ensure 微信接收方。
const onSubmitLegalName = async () => {
  if (!legalNameBackfillCanSubmit.value || legalNameUpdating.value) return
  await legalNameUpdateGuard.run(async ({ idempotencyKey }) => {
    legalNameUpdating.value = true
    legalNameUpdateError.value = ''
    legalNameUpdateErrorTraceId.value = ''
    legalNameUpdateSuccess.value = ''
    try {
      const response = await apiFetch(
        '/api/referral/legal-name/',
        {
          method: 'POST',
          credentials: 'include',
          headers: mergeIdempotencyHeaders(
            { Accept: 'application/json', 'Content-Type': 'application/json' },
            idempotencyKey,
          ),
          body: JSON.stringify({ legal_name: legalNameBackfill.value.trim() }),
        },
      )
      if (!response.ok) {
        legalNameUpdateError.value = messageFromFailedResponse(response, '保存失败，请稍后重试')
        legalNameUpdateErrorTraceId.value = extractTraceId(response)
        return
      }
      const data = await response.json()
      legalNameUpdateSuccess.value = '个人名称已保存，微信分账接收方将按实名信息登记。'
      legalName.value = String(data.legal_name || legalNameBackfill.value.trim() || '').trim()
      legalNameBackfill.value = ''
      if (data.wechat_receiver_status) {
        wechatReceiverStatus.value = String(data.wechat_receiver_status || '').trim()
        wechatReceiverReason.value = String(data.wechat_receiver_reason || '').trim()
      }
      await fetchReferralStatus()
    } catch (error) {
      console.error('补填个人名称失败:', error)
      legalNameUpdateError.value = '网络错误，请稍后重试'
      legalNameUpdateErrorTraceId.value = extractTraceId(error)
    } finally {
      legalNameUpdating.value = false
    }
  })
}

onMounted(async () => {
  if (!String(resolvedUserId.value || '').trim()) {
    resolvedUserId.value = await resolveAuthenticatedUserId()
  }
  await fetchReferralStatus()
  normalizeUrlAccessCodeToOwn({ ownCode: referralCode.value }).catch(() => {})
  if (!String(referralCode.value || '').trim()) {
    console.warn('[UserReferral] share accessCode still empty after status resolve')
  }
})
</script>
