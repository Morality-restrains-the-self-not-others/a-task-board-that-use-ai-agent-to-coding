<template>
  <div data-alias="view-user-company-settings" class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="company-settings" class="shrink-0" />

      <main class="flex-1 max-w-3xl">
        <div class="bg-white rounded-xl shadow-sm border border-border p-6 mb-6">
          <h1 class="text-2xl font-semibold text-text mb-2">在各公司的设置</h1>
          <p class="text-sm text-text-light">为每个公司分别设置昵称与头像，仅在该公司相关展示中使用。</p>
        </div>

        <div v-if="isLoading" class="bg-white rounded-xl shadow-sm border border-border p-6 text-text-light">
          加载中...
        </div>

        <div v-else class="space-y-6">
          <UserProfileCompanySettingsPanel
            v-model:company-nicknames="companyNicknames"
            :personal-nickname="personalNickname"
            :is-saving="isSaving"
            @profile-updated="applyProfileData"
            @message="onChildMessage"
            @error="onChildError"
          />

          <div class="bg-white rounded-xl shadow-sm border border-border p-4 flex items-center justify-between">
            <p v-if="message" class="text-sm text-success">{{ message }}</p>
            <p v-else-if="errorMessage" class="text-sm text-danger" :data-traceId="errorTraceId || undefined">{{ errorMessage }}</p>
            <p v-else class="text-sm text-text-light">保存后各公司的昵称设置将立即生效。</p>
            <button
              :disabled="isSaving"
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
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import UserProfileCompanySettingsPanel from '../components/UserProfileCompanySettingsPanel.vue'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils'
import { safeJson, safeResponseJson } from '@/utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))

const profileUserId = ref('')
const isLoading = ref(false)
const isSaving = ref(false)
const message = ref('')
const errorMessage = ref('')
const errorTraceId = ref('')
const personalNickname = ref('')
const companyNicknames = ref([])
const email = ref('')
const avatarUrl = ref('')

// OPT-20260819-038: 保存公司设置是写操作，防连点/超时重试双发 PATCH
const saveProfileGuard = createClickGuard()

const onChildMessage = (msg) => {
  message.value = msg || ''
  if (msg) { errorMessage.value = ''; errorTraceId.value = '' }
}

const onChildError = (msg) => {
  errorMessage.value = msg || ''
  errorTraceId.value = ''
  if (msg) message.value = ''
}

const applyProfileData = (profileData) => {
  profileUserId.value = String(profileData.user_id || '')
  personalNickname.value = profileData.personal_nickname || ''
  email.value = profileData.email || ''
  avatarUrl.value = profileData.avatar_url || ''
  companyNicknames.value = (profileData.company_nicknames || []).map((item) => ({
    company_id: String(item.company_id || ''),
    company_name: item.company_name || '',
    member_name: item.member_name || '',
    member_avatar_url: item.member_avatar_url || '',
  }))
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
    applyProfileData(await safeJson(response, {}))
  } catch (error) {
    console.error('获取个人资料失败:', error)
    errorTraceId.value = error.traceId || errorTraceId.value
    errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '获取个人资料失败'
  } finally {
    isLoading.value = false
  }
}

const saveProfile = async () => {
  // OPT-20260819-038: 保存公司设置是写操作，防连点/超时重试双发 PATCH
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
          company_nicknames: companyNicknames.value.map((item) => ({
            company_id: item.company_id,
            member_name: item.member_name,
          })),
        }),
      })
      const { data: responseData, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        errorTraceId.value = traceId
        throw new Error(responseData.error || '保存失败')
      }
      applyProfileData(responseData)
      message.value = '已保存各公司设置'
    } catch (error) {
      console.error('保存公司设置失败:', error)
      errorTraceId.value = error.traceId || errorTraceId.value
      errorMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '保存失败'
    } finally {
      isSaving.value = false
    }
  })
}

onMounted(() => {
  fetchProfile()
})
</script>
