<template>
  <div data-alias="view-user-profile" class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="profile" class="shrink-0" />

      <main class="flex-1 max-w-3xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">个人资料</h1>
          <p class="text-sm text-text-light">可设置全局头像与个人昵称；在各公司内还可分别设置昵称与头像。</p>
          <p
            v-if="displayUserId"
            class="text-sm text-text mt-3 flex flex-wrap items-baseline gap-x-2 gap-y-1"
          >
            <span class="text-text-light shrink-0">用户 ID</span>
            <code class="px-2 py-0.5 rounded bg-gray-100 text-text font-mono text-xs select-all break-all">{{ displayUserId }}</code>
          </p>
        </div>

        <div v-if="isLoading" class="bg-white rounded-xl shadow-sm border border-border p-6 text-text-light">
          加载中...
        </div>

        <div v-else class="space-y-6">
          <!-- OPT-20260726-023: 引导用户到独立的访问令牌管理页面 -->
          <div class="bg-blue-50 border border-blue-200 rounded-xl p-4 flex flex-wrap items-center justify-between gap-3">
            <p class="text-sm text-blue-800">
              🔑 <strong>访问令牌管理</strong> 已迁移至独立页面。
              <router-link :to="`/profile/access-tokens/`" class="underline font-medium hover:text-blue-600">前往管理访问令牌 →</router-link>
            </p>
          </div>

          <div class="bg-white rounded-xl shadow-sm border border-border p-6">
            <h2 class="text-lg font-semibold text-text mb-4">头像</h2>
            <p class="text-sm text-text-light mb-4">支持 JPEG、PNG、WebP、GIF，最大 2MB。</p>
            <div class="flex flex-wrap items-center gap-4">
              <img
                :src="displayAvatarUrl"
                alt=""
                class="w-24 h-24 rounded-full object-cover border border-border shrink-0"
                width="96"
                height="96"
              >
              <div class="flex flex-wrap gap-2">
                <input
                  ref="avatarFileInput"
                  type="file"
                  class="hidden"
                  accept="image/jpeg,image/png,image/webp,image/gif"
                  @change="onAvatarFileChange"
                >
                <button
                  type="button"
                  :disabled="isAvatarBusy"
                  class="px-4 py-2 rounded-lg border border-border text-text hover:bg-gray-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                  @click="openAvatarPicker"
                >
                  {{ isAvatarBusy ? '处理中...' : '上传图片' }}
                </button>
                <button
                  v-if="avatarUrl"
                  type="button"
                  :disabled="isAvatarBusy || removeAvatarGuard.isBusy()"
                  class="px-4 py-2 rounded-lg border border-red-300 text-danger hover:bg-red-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                  @click="removeAvatar"
                >
                  移除头像
                </button>
              </div>
            </div>
          </div>

          <div
            id="phone-binding"
            :data-rg-key="PHONE_BINDING_RG_KEY"
          >
            <UserProfilePhoneBindingPanel
              :has-phone="hasPhone"
              :phone-masked="phoneMasked"
              @profile-updated="fetchProfile"
              @message="onChildMessage"
              @error="onChildError"
            />
          </div>

          <!-- OPT-20260806-066: 邮箱绑定面板（厂商门户前置；sso_error=email_required / #rg=profile.email_binding 引导滚至此） -->
          <div
            id="email-binding"
            ref="emailBindingSection"
            :data-rg-key="EMAIL_BINDING_RG_KEY"
          >
            <UserProfileEmailBindingPanel
              :has-email="hasEmail"
              :email="email"
              @profile-updated="fetchProfile"
              @message="onChildMessage"
              @error="onChildError"
            />
          </div>

          <UserProfileWechatBindingPanel
            :has-wechat="hasWechat"
            :wechat-apps="wechatApps"
            :bind-available="wechatBindAvailable"
            @wechat-updated="fetchProfile"
            @message="onChildMessage"
            @error="onChildError"
          />

          <UserProfilePersonalDataExportPanel
            @message="onChildMessage"
            @error="onChildError"
          />

          <UserProfileAccountDeletionPanel
            @message="onChildMessage"
            @error="onChildError"
          />

          <div class="bg-white rounded-xl shadow-sm border border-border p-6">
            <h2 class="text-lg font-semibold text-text mb-4">个人昵称</h2>
            <label class="block text-sm text-text-light mb-2">昵称</label>
            <input
              v-model="personalNickname"
              type="text"
              maxlength="255"
              placeholder="请输入你的个人昵称"
              class="w-full max-w-md px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary outline-none transition-colors"
            >
          </div>

          <div class="bg-white rounded-xl shadow-sm border border-border p-4 flex items-center justify-between">
            <p v-if="message" class="text-sm text-success">{{ message }}</p>
            <p v-else-if="errorMessage" class="text-sm text-danger" :data-traceId="errorTraceId || undefined">{{ errorMessage }}</p>
            <p v-else class="text-sm text-text-light">保存后会立即生效。</p>
            <button
              :disabled="isSaving || saveProfileGuard.isBusy()"
              class="bg-primary text-white px-5 py-2 rounded-lg hover:bg-primary/90 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
              @click="saveProfile"
            >
              {{ isSaving ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, useTemplateRef } from 'vue'
import { useRoute } from 'vue-router'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import UserProfileEmailBindingPanel from '../components/UserProfileEmailBindingPanel.vue'
import UserProfilePhoneBindingPanel from '../components/UserProfilePhoneBindingPanel.vue'
import UserProfileWechatBindingPanel from '../components/UserProfileWechatBindingPanel.vue'
import UserProfilePersonalDataExportPanel from '../components/UserProfilePersonalDataExportPanel.vue'
import UserProfileAccountDeletionPanel from '../components/UserProfileAccountDeletionPanel.vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { getCookie } from '../utils/cookieUtils'
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js'
import { safeJson, safeResponseJson } from '@/utils/safeResponseJson.js'
import { scrollToRgKey } from '../utils/rgDeepLink.js'
import toastService from '../utils/toastService'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { EMAIL_BINDING_RG_KEY, EMAIL_REQUIRED_SSO_ERROR, isEmailRequiredSsoError } from '../utils/emailBindingDeepLink.js'
import { PHONE_BINDING_RG_KEY, isPhoneBindingHash } from '../utils/phoneBindingDeepLink.js'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))

const profileUserId = ref('')
const displayUserId = computed(() => {
  if (profileUserId.value) return profileUserId.value
  const fromRoute = route.params.id != null && String(route.params.id).trim() !== ''
    ? String(route.params.id).trim()
    : ''
  if (fromRoute) return fromRoute
  return String(getCookie('userId') || '').trim()
})

const isLoading = ref(false)
const isSaving = ref(false)
// OPT-20260819-038: 上传头像/移除头像/保存资料 均为写操作，防连点双发
const uploadAvatarGuard = createClickGuard()
const removeAvatarGuard = createClickGuard()
const saveProfileGuard = createClickGuard()
const message = ref('')
const errorMessage = ref('')
const errorTraceId = ref('')
const personalNickname = ref('')
const companyNicknames = ref([])
const email = ref('')
// OPT-20260806-066: 邮箱绑定状态（profile.has_email；旧后端无该字段时兜底 email 非空）
const hasEmail = ref(false)
const hasPhone = ref(false)
const phoneMasked = ref('')
const hasWechat = ref(false)
const wechatApps = ref([])
const wechatBindAvailable = ref(true)
const avatarUrl = ref('')
const isAvatarBusy = ref(false)

const displayAvatarUrl = computed(() => {
  if (avatarUrl.value) return avatarUrl.value
  return initialsAvatarDataUri(personalNickname.value || email.value || 'user')
})

const onChildMessage = (msg) => {
  message.value = msg || ''
  if (msg) { errorMessage.value = ''; errorTraceId.value = '' }
}

const onChildError = (msg) => {
  errorMessage.value = msg || ''
  errorTraceId.value = ''
  if (msg) message.value = ''
}

const syncCurrentUserFromProfile = () => {
  if (typeof window !== 'undefined' && window.currentUser && window.currentUser.isAuthenticated) {
    window.currentUser.username = personalNickname.value || email.value || window.currentUser.username
    window.currentUser.avatarUrl = avatarUrl.value || null
  }
}

const applyProfileData = (profileData) => {
  profileUserId.value = String(profileData.user_id || '')
  personalNickname.value = profileData.personal_nickname || ''
  email.value = profileData.email || ''
  // 旧后端无 has_email 字段时兜底：email 非空即视为已绑定
  hasEmail.value = profileData.has_email === true || Boolean(profileData.email)
  hasPhone.value = Boolean(profileData.has_phone)
  phoneMasked.value = profileData.phone_masked || ''
  hasWechat.value = Boolean(profileData.has_wechat)
  wechatApps.value = profileData.wechat_apps || []
  // 旧后端无 wechat_bind_available 字段时默认 true（不隐藏入口）
  wechatBindAvailable.value = profileData.wechat_bind_available ?? true
  avatarUrl.value = profileData.avatar_url || ''
  companyNicknames.value = (profileData.company_nicknames || []).map((item) => ({
    company_id: String(item.company_id || ''),
    company_name: item.company_name || '',
    member_name: item.member_name || '',
    member_avatar_url: item.member_avatar_url || '',
  }))
  syncCurrentUserFromProfile()
}

// OPT-20260806-046: 上传 UI 还原（后端安全加固完成后：magic bytes + 解码校验）
const avatarFileInput = useTemplateRef('avatarFileInput')
// OPT-20260806-066: 邮箱绑定面板锚点（sso_error=email_required 引导滚动定位）
const emailBindingSection = useTemplateRef('emailBindingSection')

const openAvatarPicker = () => {
  errorMessage.value = ''
  errorTraceId.value = ''
  message.value = ''
  avatarFileInput.value?.click()
}

const onAvatarFileChange = async (event) => {
  const input = event.target
  const file = input.files && input.files[0]
  input.value = ''
  if (!file) return
  // OPT-20260819-038: 上传头像是写操作，防连点双发
  await uploadAvatarGuard.run(async ({ idempotencyKey }) => {
    isAvatarBusy.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    message.value = ''
    try {
      const body = new FormData()
      body.append('avatar', file)
      const response = await apiFetch('/api/accounts/users/profile/avatar/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
        body,
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: null })
      if (!response.ok) {
        errorTraceId.value = traceId
        throw new Error(data?.error || '上传失败')
      }
      applyProfileData(data)
      message.value = '头像已更新'
    } catch (error) {
      console.error('上传头像失败:', error)
      errorTraceId.value = error.traceId || errorTraceId.value
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '上传失败'
    } finally {
      isAvatarBusy.value = false
    }
  })
}

const removeAvatar = async () => {
  // OPT-20260819-038: 移除头像是写操作，防连点/超时重试双发 DELETE
  await removeAvatarGuard.run(async ({ idempotencyKey }) => {
    isAvatarBusy.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    message.value = ''
    try {
      const response = await apiFetch('/api/accounts/users/profile/avatar/', {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders({ Accept: 'application/json' }, idempotencyKey),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: null })
      if (!response.ok) {
        errorTraceId.value = traceId
        throw new Error(data?.error || '移除失败')
      }
      applyProfileData(data)
      message.value = '已移除头像'
    } catch (error) {
      console.error('移除头像失败:', error)
      errorTraceId.value = error.traceId || errorTraceId.value
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '移除失败'
    } finally {
      isAvatarBusy.value = false
    }
  })
}

const fetchProfile = async () => {
  isLoading.value = true
  errorMessage.value = ''
  errorTraceId.value = ''
  message.value = ''
  try {
    const response = await apiFetch('/api/accounts/users/profile/', {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      const { traceId } = await safeResponseJson(response, { fallback: {} })
      errorTraceId.value = traceId
      throw new Error('获取个人资料失败')
    }
    const profileData = await safeJson(response, null)
    if (profileData === null) throw new Error('获取个人资料失败')
    applyProfileData(profileData)
  } catch (error) {
    console.error('获取个人资料失败:', error)
    errorTraceId.value = error.traceId || errorTraceId.value
    errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '获取个人资料失败'
  } finally {
    isLoading.value = false
  }
}

const saveProfile = async () => {
  // OPT-20260819-038: 保存个人资料是写操作，防连点/超时重试双发 PATCH
  await saveProfileGuard.run(async ({ idempotencyKey }) => {
    isSaving.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    message.value = ''
    try {
      const response = await apiFetch('/api/accounts/users/profile/', {
        method: 'PATCH',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          personal_nickname: personalNickname.value,
        }),
      })
      const { data: responseData, traceId } = await safeResponseJson(response, { fallback: null })
      if (!response.ok) {
        errorTraceId.value = traceId
        throw new Error(responseData?.error || '保存失败')
      }
      applyProfileData(responseData)
      message.value = '已保存个人资料'
    } catch (error) {
      console.error('保存个人资料失败:', error)
      errorTraceId.value = error.traceId || errorTraceId.value
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '保存失败'
    } finally {
      isSaving.value = false
    }
  })
}

const focusEmailBindingIfRequired = async () => {
  // 邮箱绑定区在 isLoading 解除后才挂载；须等 fetchProfile 完成再定位，
  // 否则 nextTick 时 ref 仍为 null，用户落在页顶需手动查找。
  if (!isEmailRequiredSsoError(route.query)) return
  await nextTick()
  if (!scrollToRgKey(EMAIL_BINDING_RG_KEY, { scrollOptions: { block: 'center' } })) {
    emailBindingSection.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

const focusPhoneBindingIfNeeded = async () => {
  if (typeof window === 'undefined' || !isPhoneBindingHash(window.location.hash)) return
  await nextTick()
  if (!scrollToRgKey(PHONE_BINDING_RG_KEY, { scrollOptions: { block: 'center' } })) {
    document.getElementById('phone-binding')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

onMounted(async () => {
  // OPT-20260806-065/066：厂商门户需绑定邮箱账号。taskAuth SSO bridge 对无邮箱
  // 登录方式的用户（微信扫码）拒绝签发并重定向到个人中心（?sso_error=email_required）。
  // 引导用户到本页邮箱绑定面板，资料加载后再滚动定位并高亮。
  if (isEmailRequiredSsoError(route.query)) {
    toastService.warning('厂商门户需绑定邮箱账号后使用，请在下方邮箱绑定区域完成绑定')
  }
  await fetchProfile()
  await focusEmailBindingIfRequired()
  await focusPhoneBindingIfNeeded()
})
</script>
