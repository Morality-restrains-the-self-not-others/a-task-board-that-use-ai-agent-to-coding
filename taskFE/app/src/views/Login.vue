<template>
  <div data-alias="cmp-login-form" class="bg-gradient-to-br from-primary/5 to-background min-h-screen flex flex-col">
    <main class="flex-grow flex items-start sm:items-center justify-center pt-8 sm:pt-12 pb-10 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-6 sm:space-y-8 bg-white p-6 sm:p-10 rounded-2xl shadow-xl border border-gray-100">
        <div class="text-center">
          <img :src="faviconUrl" alt="云端开发" class="w-16 h-16 rounded-2xl mx-auto mb-3" />
          <h1 class="text-xl sm:text-2xl font-semibold text-text mb-2">欢迎登录</h1>
          <p v-if="tenantContextLabel" class="inline-flex items-center rounded-full bg-primary/10 text-primary text-xs px-3 py-1 mb-3">
            当前工作区：{{ tenantContextLabel }}
          </p>
          <p class="text-text-light">登录您的账户，管理您的AI项目</p>
          <p
            v-if="oidcResumeHint"
            class="text-sm text-primary bg-primary/10 rounded-lg px-3 py-2 mt-3"
            data-testid="login-oidc-resume-hint"
          >
            这是从自建 GitLab 跳过来的登录。请使用本平台账号登录，成功后会自动回到 GitLab。
          </p>
        </div>

        <LoginMethodSelector
          v-model="loginMethod"
          :allow-phone-login="allowPhoneLogin"
          @method-changed="handleMethodChanged"
        />

        <!-- 微信扫码登录入口（管理员开启策略且微信应用已配置时展示） -->
        <div v-if="loginMethod !== 'accessToken' && allowWechatLogin" class="flex flex-col items-center gap-2">
          <button
            type="button"
            class="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg border border-green-500 text-green-600 hover:bg-green-50 transition-colors duration-200 text-sm font-medium disabled:opacity-60"
            :disabled="wechatLoginBusy"
            @click="handleWeChatLogin"
          >
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.742 2.04 2.439 3.379 4.265 4.135a8.81 8.81 0 002.465.357c.276 0 .543-.027.811-.05l1.903 1.114a.326.326 0 00.167.054c.163 0 .29-.13.29-.295a.707.707 0 00-.048-.213l-.39-1.48a.59.59 0 01.213-.665c1.504-1.104 2.428-2.604 2.428-4.304 0-3.636-3.458-6.624-7.044-6.659zm-3.377 2.976c.519 0 .94.43.94.959a.95.95 0 01-.94.958.949.949 0 01-.94-.958c0-.53.421-.959.94-.959zm4.697 0c.519 0 .94.43.94.959a.95.95 0 01-.94.958.949.949 0 01-.94-.958c0-.53.421-.959.94-.959z"/>
            </svg>
            {{ wechatLoginBusy ? '正在跳转…' : (isWeChatBrowser() ? '微信一键登录' : '微信扫码登录') }}
          </button>
          <p
            v-if="wechatLoginError"
            class="text-sm text-red-600 text-center"
            role="alert"
            :data-traceId="wechatLoginErrorTraceId || undefined"
          >{{ wechatLoginError }}</p>
        </div>

        <div v-if="allowWechatLogin" class="relative flex items-center my-4">
          <div class="flex-grow border-t border-gray-200"></div>
          <span class="flex-shrink mx-4 text-xs text-gray-400">或</span>
          <div class="flex-grow border-t border-gray-200"></div>
        </div>

        <form class="space-y-6" @submit="handleSubmit">
          <LoginEmailPasswordFields
            v-if="loginMethod === 'emailPassword'"
            :show-email-password="showEmailPassword"
            @update:show-email-password="showEmailPassword = $event"
          />

          <LoginPhonePasswordFields
            v-else-if="loginMethod === 'phonePassword'"
            v-model:login-phone-prefix="loginPhonePrefix"
            :phone-national-password="phoneNationalPassword"
            v-model:show-phone-password="showPhonePassword"
            :country-dial-options="filteredCountryOptions"
            :country-loading="countryLoading"
            :country-error-hint="countryErrorHint"
            :national-placeholder="nationalPlaceholder"
            :max-national-len="maxNationalLen"
            @national-pwd-input="onLoginPhoneNationalPwdInput"
            @national-pwd-paste="onLoginPhoneNationalPwdPaste"
          />

          <LoginAccessTokenFields
            v-else-if="loginMethod === 'accessToken'"
            v-model:access-token-username-input="accessTokenUsernameInput"
            v-model:access-token-input="accessTokenInput"
            v-model:show-access-token="showAccessToken"
          />

          <LoginLegalConsentPanel
            :accept-all="acceptAll"
            :accept-privacy="acceptPrivacy"
            :accept-license="acceptLicense"
            :privacy-load-error="privacyLoadError"
            :license-load-error="licenseLoadError"
            :current-privacy-policy="currentPrivacyPolicy"
            :current-license-agreement="currentLicenseAgreement"
            @update:accept-privacy="acceptPrivacy = $event"
            @update:accept-license="acceptLicense = $event"
            @update:accept-all="
              (v) => {
                acceptPrivacy = v
                acceptLicense = v
              }
            "
            @open-privacy="showPrivacyReader = true"
            @open-license="showLicenseReader = true"
          />

          <div>
            <button
              type="submit"
              class="btn-primary w-full py-3 text-base disabled:opacity-60 disabled:cursor-not-allowed"
              :disabled="isLoginSubmitDisabled"
              :title="loginSubmitDisabledReason"
            >
              {{ isLoading ? '登录中...' : '登录' }}
            </button>
          </div>

          <LoginRegisterInviteSection />

          <div v-if="showResendButton" class="mt-4">
            <button
              type="button"
              :disabled="isResendingEmail"
              class="btn-secondary w-full py-3 text-base"
              @click="handleResendActivationEmail"
            >
              {{ isResendingEmail ? '发送中...' : '重新发送激活邮件' }}
            </button>
          </div>
        </form>

        <div class="text-center pt-4 mt-4 border-t border-gray-100">
          <router-link
            to="/auth/admin-login/"
            class="text-xs text-gray-400 hover:text-gray-500 transition-colors"
          >
            管理员登录
          </router-link>
        </div>
      </div>
    </main>

    <LoginPolicyReaderModals
      :show-privacy-reader="showPrivacyReader"
      :show-license-reader="showLicenseReader"
      :current-privacy-policy="currentPrivacyPolicy"
      :current-license-agreement="currentLicenseAgreement"
      @close-privacy="showPrivacyReader = false"
      @close-license="showLicenseReader = false"
    />

    <LoginMarketingFooter :is-mobile-viewport="isMobileViewport" />
  </div>
</template>

<script setup>
/* @alias:cmp-login-form */
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'

// src/public/img/icon128.png 站点根静态资源（源：app/static/img/icon128.png）：
// JS 字符串避免模板 transformAssetUrls 构建期解析
// （root-relative src 在 vite build 下 resolve 失败，阻断生产构建）
const faviconUrl = '/img/icon128.png'
import { useRoute } from 'vue-router'
import LoginEmailPasswordFields from '../components/auth/LoginEmailPasswordFields.vue'
import LoginLegalConsentPanel from '../components/auth/LoginLegalConsentPanel.vue'
import LoginMarketingFooter from '../components/auth/LoginMarketingFooter.vue'
import LoginMethodSelector from '../components/auth/LoginMethodSelector.vue'
import LoginPolicyReaderModals from '../components/auth/LoginPolicyReaderModals.vue'
import LoginRegisterInviteSection from '../components/auth/LoginRegisterInviteSection.vue'
import LoginPhonePasswordFields from '../components/auth/LoginPhonePasswordFields.vue'
import LoginAccessTokenFields from '../components/auth/LoginAccessTokenFields.vue'
import modalService from '../utils/modalService.js'
import { useLoginPhoneFields } from '../composables/auth/useLoginPhoneFields.js'
import {
  useLoginFeaturePolicy,
  EMAIL_PASSWORD_METHOD,
  PHONE_PASSWORD_METHOD,
  isPhoneLoginMethod,
} from '../composables/auth/useLoginFeaturePolicy.js'
import { useLoginOidcResume } from '../composables/auth/useLoginOidcResume.js'
import { useAlreadyLoggedInPrompt } from '../composables/auth/useAlreadyLoggedInPrompt.js'
import { useLoginLegalConsent } from '../composables/auth/useLoginLegalConsent.js'
import { useLoginSubmit } from '../composables/auth/useLoginSubmit.js'
import {
  buildWechatOAuthUrl,
  navigateWechatOAuth,
  resolveWechatCallbackRedirect,
} from '../utils/wechatLoginFlow.js'
import { captureReferralAccessCodeFromSearch } from '../utils/referralAccessCodeUtils.js'

console.log(`Login.vue 组件已加载 - ${new Date().toISOString()}`)

defineProps({
  user: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: '',
    }),
  },
})

const emit = defineEmits(['login-success', 'login-failure'])

// 微信登录（v64 多应用）— 微信内置浏览器走公众号授权 (inapp)，其余走扫码 (web)
const isWeChatBrowser = () => /MicroMessenger/i.test(navigator.userAgent)
const wechatLoginBusy = ref(false)
const wechatLoginError = ref('')
const wechatLoginErrorTraceId = ref('')
const handleWeChatLogin = async () => {
  if (wechatLoginBusy.value) return
  wechatLoginError.value = ''
  wechatLoginErrorTraceId.value = ''
  const app = isWeChatBrowser() ? 'inapp' : 'web'
  // 携带 next：OAuth 全程跨页面跳转（微信 → 后端回调 → /auth/login/?wechat_token=…），
  // next 经 handleWeChatLogin → state → handleWeChatCallback 原样带回（taskAuth 保证）。
  const next = new URLSearchParams(window.location.search).get('next')
  const url = buildWechatOAuthUrl(app, next)
  wechatLoginBusy.value = true
  try {
    const result = await navigateWechatOAuth(url)
    if (result.status === 'unavailable') {
      wechatLoginError.value = result.message
      wechatLoginErrorTraceId.value = result.traceId || ''
    }
  } finally {
    wechatLoginBusy.value = false
  }
}

// 处理微信回调参数 (wechat_token / wechat_error / wechat_bound)
const handleWeChatCallback = () => {
  const urlParams = new URLSearchParams(window.location.search)
  const wechatToken = urlParams.get('wechat_token')
  const wechatError = urlParams.get('wechat_error')
  const wechatBound = urlParams.get('wechat_bound')
  const next = urlParams.get('next')

  // OPT-20260806-009: 回调参数状态上报 — 线上「微信登录 401」事件此前无法定位
  // 是 token 缺失、动态 import 异常还是凭据落盘失败，只能靠猜；统一前缀
  // [wechat-callback] 便于 Grafana 全文检索（与 taskAuth event=wechat_state_* 对照）。
  console.log(
    `[wechat-callback] params token=${wechatToken ? 'present' : 'absent'}` +
      ` error=${wechatError ? 'present' : 'absent'} bound=${wechatBound || 'absent'}` +
      ` next=${next ? 'present' : 'absent'}`,
  )

  if (wechatBound === '1') {
    console.log('[wechat-callback] branch=bound')
    modalService.alert({
      message: '微信绑定成功',
      showCancel: false,
      confirmText: '确定',
    })
    const newUrl = window.location.pathname + window.location.search.replace(/[?&]wechat_bound=[^&]*/g, '').replace(/\?$/, '')
    window.history.replaceState({}, document.title, newUrl)
  }

  if (wechatError) {
    console.log('[wechat-callback] branch=error detail=' + wechatError.slice(0, 200))
    modalService.alert({
      message: decodeURIComponent(wechatError),
      showCancel: false,
      confirmText: '确定',
    })
    // 清除 URL 参数
    const newUrl = window.location.pathname + window.location.search.replace(/[?&]wechat_error=[^&]*/g, '').replace(/[?&]wechat_token=[^&]*/g, '').replace(/\?$/, '')
    window.history.replaceState({}, document.title, newUrl)
    return
  }

  if (wechatToken) {
    console.log('[wechat-callback] branch=token')
    // 微信登录成功，使用 token 完成登录
    // 落点优先级与正常登录一致（resolveWechatCallbackRedirect）：
    // 后端回调携带的 next（用户原始回跳或按角色计算的落点）→ localStorage → 默认。
    // 修复：此前 next||'/' 兜底到公开营销首页，导致「微信扫码登录后跳转到首页」。
    import('../utils/finishWechatCallbackLogin.js').then(async ({ finishWechatCallbackLogin }) => {
      await finishWechatCallbackLogin({ wechatToken, next })
    }).catch((err) => {
      // 动态 import 或凭据解析异常：降级为内存 token（仅当前页有效），
      // 跳转后凭据未落盘的场景仍可能 401 — 此日志即排障入口
      console.error('[wechat-callback] branch=token import_or_resolve_failed:', err)
      import('../utils/apiUtils.js').then(({ setCachedAuthToken }) => setCachedAuthToken(wechatToken))
      window.location.href = resolveWechatCallbackRedirect(next)
    })
  }
}

const route = useRoute()
const savedPreferredLoginMethod = localStorage.getItem('preferredLoginMethod') || EMAIL_PASSWORD_METHOD

const loginMethod = ref(
  isPhoneLoginMethod(savedPreferredLoginMethod) ? EMAIL_PASSWORD_METHOD : savedPreferredLoginMethod,
)
const showEmailPassword = ref(false)
const showPhonePassword = ref(false)
const showAccessToken = ref(false)
const accessTokenUsernameInput = ref('')
const accessTokenInput = ref('')
const isMobileViewport = ref(false)

const phoneFields = useLoginPhoneFields()
const {
  filteredCountryOptions,
  fetchAllowedCodes,
  countryLoading,
  countryError,
  loginPhonePrefix,
  phoneNationalPassword,
  maxNationalLen,
  nationalPlaceholder,
  onLoginPhoneNationalPwdInput,
  onLoginPhoneNationalPwdPaste,
} = phoneFields

const legalConsent = useLoginLegalConsent()
const {
  currentPrivacyPolicy,
  privacyLoadError,
  acceptPrivacy,
  showPrivacyReader,
  currentLicenseAgreement,
  licenseLoadError,
  acceptLicense,
  showLicenseReader,
  acceptAll,
  loadCurrentPrivacy,
  loadCurrentLicenseAgreement,
} = legalConsent

const {
  allowPhoneLogin,
  allowWechatLogin,
  loadPublicFeaturePolicy,
  enforcePhonePolicy,
  handleMethodChanged,
} = useLoginFeaturePolicy({ loginMethod, phoneFields })

const { readOidcResumeNextFromRoute, resumeOidcFlowIfReady } = useLoginOidcResume()
const oidcResumeHint = computed(() => Boolean(readOidcResumeNextFromRoute()))

// OPT-20260810-044：已登录命中登录页时弹窗由用户决定是否跳转（不静默 replace）
const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt()

const {
  isLoading,
  showResendButton,
  isResendingEmail,
  isLoginSubmitDisabled,
  loginSubmitDisabledReason,
  handleSubmit,
  handleResendActivationEmail,
} = useLoginSubmit({
  emit,
  loginMethod,
  phoneFields,
  legalConsent,
  accessTokenUsernameInput,
  accessTokenInput,
})

const tenantContextLabel = computed(() => {
  const tenant = route.params.tenant
  if (!tenant) {
    return ''
  }
  return String(tenant)
})

const countryErrorHint = computed(() => {
  if (countryError.value) {
    return '区域限制暂不可用，显示全部地区'
  }
  return ''
})

const updateViewportState = () => {
  isMobileViewport.value = window.innerWidth < 640
}

onMounted(() => {
  updateViewportState()
  window.addEventListener('resize', updateViewportState)

  const savedMethod = localStorage.getItem('preferredLoginMethod')
  if (savedMethod && [EMAIL_PASSWORD_METHOD, PHONE_PASSWORD_METHOD].includes(savedMethod)) {
    loginMethod.value = savedMethod
  }
  enforcePhonePolicy()
  loadPublicFeaturePolicy()
  fetchAllowedCodes()
  resumeOidcFlowIfReady()
  loadCurrentPrivacy()
  loadCurrentLicenseAgreement()
  // OPT-20260820-036: 分享链接带 ?accessCode= 时写入 sessionStorage，
  // 供微信 OAuth 入口透传到 taskAuth 回填推荐关系。
  captureReferralAccessCodeFromSearch(window.location.search)
  handleWeChatCallback()

  const urlParams = new URLSearchParams(window.location.search)
  const activated = urlParams.get('activated')
  const email = urlParams.get('email')
  const error = urlParams.get('error')

  console.log('Login.vue - URL参数检查:', { activated, email, error })

  if (activated === 'true') {
    console.log('Login.vue - 激活成功，显示提示消息')
    modalService.alert({
      message: '邮箱激活成功！请使用您的账户登录。',
      showCancel: false,
      confirmText: '确定',
    })

    if (email && loginMethod.value === 'emailPassword') {
      setTimeout(() => {
        const emailInput = document.getElementById('email')
        if (emailInput) {
          emailInput.value = email
        }
      }, 100)
    }
  } else if (activated === 'false' && error) {
    console.log('Login.vue - 激活失败，显示错误消息')
    modalService.alert({
      message: `激活失败：${error}`,
      showCancel: false,
      confirmText: '确定',
    })
  }

  // 已登录用户打开登录页：询问是否前往 next / 工作面板；确认才跳转，取消留在登录页。
  maybePromptAlreadyLoggedIn().catch(() => {
    /* 静默失败：弹窗失败不影响登录页渲染 */
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateViewportState)
})
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>
