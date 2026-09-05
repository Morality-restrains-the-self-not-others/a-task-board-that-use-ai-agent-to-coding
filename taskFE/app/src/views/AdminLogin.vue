<template>
  <div data-alias="cmp-admin-login-form" class="bg-gradient-to-br from-gray-900 to-gray-800 min-h-screen flex flex-col">
    <main class="flex-grow flex items-start sm:items-center justify-center pt-8 sm:pt-12 pb-10 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-6 sm:space-y-8 bg-white p-6 sm:p-10 rounded-2xl shadow-xl border border-gray-100">
        <div class="text-center">
          <img :src="faviconUrl" alt="云端开发" class="w-16 h-16 rounded-2xl mx-auto mb-3" />
          <h1 class="text-xl sm:text-2xl font-semibold text-text mb-2">管理员登录</h1>
          <p class="text-text-light text-sm">仅限系统管理员使用此入口登录</p>
        </div>

        <form class="space-y-6" @submit="handleSubmit">
          <LoginEmailPasswordFields
            :show-email-password="showEmailPassword"
            @update:show-email-password="showEmailPassword = $event"
          />

          <div>
            <button
              type="submit"
              class="w-full py-3 text-base rounded-lg font-medium transition-colors duration-200"
              :class="isLoading ? 'bg-gray-400 text-white cursor-not-allowed' : 'bg-gray-800 text-white hover:bg-gray-900'"
              :disabled="isLoading"
            >
              {{ isLoading ? '登录中...' : '管理员登录' }}
            </button>
          </div>

          <div v-if="showResendButton" class="mt-4">
            <button
              type="button"
              :disabled="isResendingEmail"
              class="w-full py-2.5 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 transition-colors duration-200"
              @click="handleResendActivationEmail"
            >
              {{ isResendingEmail ? '发送中...' : '重新发送激活邮件' }}
            </button>
          </div>
        </form>

        <div class="text-center pt-4 border-t border-gray-100">
          <router-link
            to="/auth/login/"
            class="text-sm text-primary hover:underline"
          >
            ← 返回普通用户登录
          </router-link>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
/* @alias:cmp-admin-login-form */
import { ref } from 'vue'
import LoginEmailPasswordFields from '../components/auth/LoginEmailPasswordFields.vue'
import { useLoginSubmit } from '../composables/auth/useLoginSubmit.js'

// src/public/img/icon128.png 站点根静态资源（源：app/static/img/icon128.png）：
// JS 字符串避免模板 transformAssetUrls 构建期解析
// （root-relative src 在 vite build 下 resolve 失败，阻断生产构建）
const faviconUrl = '/img/icon128.png'

console.log(`AdminLogin.vue 组件已加载 - ${new Date().toISOString()}`)

defineProps({})

const emit = defineEmits(['login-success', 'login-failure'])

// 管理员登录仅支持邮箱+密码方式
const loginMethod = ref('emailPassword')
const showEmailPassword = ref(false)

// 为 useLoginSubmit 提供最小依赖（管理员登录不用手机号/法律条款/访问令牌）
const emptyRef = ref('')
const falseRef = ref(false)
const nullRef = ref(null)

const phoneFields = {
  isPhoneValidForPassword: falseRef,
  loginPhonePasswordForApi: emptyRef,
}

const legalConsent = {
  acceptPrivacy: falseRef,
  acceptLicense: falseRef,
  currentPrivacyPolicy: nullRef,
  currentLicenseAgreement: nullRef,
  privacyLoadError: emptyRef,
  licenseLoadError: emptyRef,
}

const {
  isLoading,
  showResendButton,
  isResendingEmail,
  handleSubmit,
  handleResendActivationEmail,
} = useLoginSubmit({
  emit,
  loginMethod,
  phoneFields,
  legalConsent,
  accessTokenUsernameInput: emptyRef,
  accessTokenInput: emptyRef,
  // OPT-20260824-001: 管理员入口分离 — 提交到 /api/auth/admin-login/，
  // 后端仅放行平台管理员/员工账号，客户账号一律 403。
  adminLogin: true,
})
</script>
