<template>
  <div class="people-join-container p-6">
    <h2 class="text-2xl font-bold mb-6">加入 {{ teamName }}</h2>
    
    <div class="bg-white rounded-xl shadow-md p-6">
      <div v-if="loading" class="flex justify-center items-center py-10">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
      </div>
      
      <div v-else-if="error" class="bg-error/10 border border-error text-error rounded-lg p-4 mb-4" :data-traceId="traceId || undefined">
        {{ error }}
      </div>
      
      <div v-else-if="success" class="bg-success/10 border border-success text-success rounded-lg p-4 mb-4">
        {{ success }}
      </div>
      
      <div v-else-if="inviteValid">
        <!-- 已登录状态 -->
        <div v-if="isAuthenticated">
          <p class="mb-4">您正在被邀请加入 {{ teamName }} 团队，点击下方按钮确认加入：</p>
          <p v-if="remainingHint" class="mb-4 text-sm text-text-light" data-testid="people-join-remaining">{{ remainingHint }}</p>
          
          <div class="flex justify-center gap-4">
            <button
              type="button"
              data-testid="people-join-confirm-btn"
              @click="joinTeam"
              class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors flex items-center"
              :disabled="isJoining || joinGuard.isBusy()"
            >
              <span v-if="isJoining">加入中...</span>
              <span v-else>确认加入 {{ teamName }}</span>
            </button>
            <button
              type="button"
              data-testid="people-join-reject-btn"
              @click="rejectInvitation"
              class="bg-gray-200 text-gray-800 px-6 py-2 rounded-lg hover:bg-gray-300 transition-colors"
            >
              拒绝邀请
            </button>
          </div>
        </div>
        
        <!-- 未登录状态：须提供登录确认 + 拒绝，避免仅文案无 CTA（线上回归） -->
        <div v-else>
          <p class="mb-4">您正在被邀请加入 {{ teamName }} 团队，请先注册或登录后确认加入</p>
          <div class="flex justify-center gap-4">
            <button
              type="button"
              data-testid="people-join-login-btn"
              @click="goToLogin"
              class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
            >
              登录/注册后确认加入
            </button>
            <button
              type="button"
              data-testid="people-join-reject-btn"
              @click="rejectInvitation"
              class="bg-gray-200 text-gray-800 px-6 py-2 rounded-lg hover:bg-gray-300 transition-colors"
            >
              拒绝邀请
            </button>
          </div>
        </div>
      </div>
      
      <div v-else-if="!inviteValid && !error">
        <p class="mb-4">正在验证邀请信息...</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch, clearCachedAuthToken } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { clearStoredUserId, getStoredUserId } from '../utils/sessionUserIdUtils.js'
import { savePostLoginRedirect } from '../utils/authReturnUrl.js'
import { AuthSessionGuardService } from '../domain/auth/services/auth_session_guard_service.js'
import { extractTraceId } from '../utils/traceId.js'

const route = useRoute()
const router = useRouter()
const tenantId = route.params.tenant
const inviteToken = route.query.token

const loading = ref(true)
const error = ref('')
const success = ref('')
const isJoining = ref(false)
const isAuthenticated = ref(false)
const teamName = ref('')
const inviteValid = ref(false)
const remainingHint = ref('')
/** 最近一次失败请求的 traceId（Loki 全链路检索），成功/新请求时清空。 */
const traceId = ref('')

/** 与 AuthSessionGuard 同源：localStorage/可读 cookie + /me/，失败再走 profile（覆盖 HttpOnly userId）。 */
const authSessionGuard = new AuthSessionGuardService({
  apiFetch,
  getCookie: getStoredUserId,
  clearUserIdCookie: clearStoredUserId,
  clearAuthToken: clearCachedAuthToken,
})

/** 当前邀请页路径（含 query），供登录后回跳；须为站内相对路径。 */
function inviteReturnPath() {
  if (typeof window === 'undefined') return '/'
  return `${window.location.pathname}${window.location.search}`
}

const goToLogin = () => {
  const returnPath = inviteReturnPath()
  savePostLoginRedirect(returnPath)
  window.location.href = `/auth/login/?next=${encodeURIComponent(returnPath)}`
}

const checkAuthStatus = async () => {
  try {
    isAuthenticated.value = await authSessionGuard.isAuthenticated()
  } catch (err) {
    console.error('检查认证状态失败:', err)
    traceId.value = extractTraceId(err)
    isAuthenticated.value = false
  }
}

const fetchTeamName = async () => {
  try {
    const response = await apiFetch(`/api/tenant/${tenantId}/accounts/companies/current/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (response.ok) {
      const data = await response.json()
      teamName.value = data.name || '团队'
    }
  } catch (error) {
    console.error('获取团队名称失败:', error)
    traceId.value = extractTraceId(error)
    teamName.value = '团队'
  }
}

const validateInviteToken = async () => {
  if (!inviteToken) {
    return false
  }
  
  try {
    // 调用后端API验证邀请令牌
    const response = await apiFetch(`/api/tenant/${tenantId}/accounts/members/validate-invite/?token=${inviteToken}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.link_kind === 'open') {
        if (data.remaining_uses == null) {
          remainingHint.value = '此为开放邀请链接，可供多人加入。'
        } else {
          remainingHint.value = `此为开放邀请链接，剩余可加入人数：${data.remaining_uses}`
        }
      } else {
        remainingHint.value = ''
      }
      return data.valid === true || data.valid === 'True'
    }
    traceId.value = extractTraceId(response)
  } catch (error) {
    console.error('验证邀请令牌失败:', error)
    traceId.value = extractTraceId(error)
  }

  return false
}

const checkInviteStatus = async () => {
  traceId.value = '' // 新一次检查清空上次失败 traceId
  // 检查邀请令牌是否存在
  if (!inviteToken) {
    error.value = '邀请链接无效，请检查链接是否正确'
    loading.value = false
    return
  }
  
  try {
    // 并行执行所有异步请求，提高加载速度
    const [authStatus, teamData, inviteStatus] = await Promise.all([
      checkAuthStatus(),
      fetchTeamName(),
      validateInviteToken()
    ])
    
    // 检查邀请令牌状态
    if (!inviteStatus) {
      error.value = '邀请链接无效或已过期'
      return
    }
    
    inviteValid.value = true
    
    // 未登录：预写站内相对回跳路径（与登录按钮一致；禁止存绝对外链 URL）
    if (!isAuthenticated.value) {
      const returnPath = inviteReturnPath()
      savePostLoginRedirect(returnPath)
      console.log('[PeopleJoin] 已设置登录后重定向到邀请页面:', returnPath)
    }
  } catch (err) {
    console.error('检查邀请状态失败:', err)
    error.value = '检查邀请状态失败，请稍后重试'
    traceId.value = extractTraceId(err)
  } finally {
    loading.value = false
  }
}

const joinGuard = createClickGuard()

const joinTeam = async () => {
  // OPT-20260819-038: 加入团队是写操作，防连点双发 POST
  await joinGuard.run(async ({ idempotencyKey }) => {
    if (!inviteToken || !inviteValid.value) {
      error.value = '邀请链接无效，请检查链接是否正确'
      return
    }

    isJoining.value = true

    try {
      // 调用后端API加入团队
      const response = await apiFetch(`/api/tenant/${tenantId}/accounts/members/join/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          token: inviteToken
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        traceId.value = extractTraceId(response)
        throw new Error(errorData.error || '加入团队失败')
      }

      const result = await response.json()
      success.value = `成功加入 ${teamName.value} 团队！`
      traceId.value = ''

      // 跳转到工作面板页面
      setTimeout(() => {
        router.push(`/tenant/${tenantId}/work-panel/`)
      }, 2000)
    } catch (err) {
      // 禁止 catch (error)：会遮蔽同名 ref，导致失败文案写不进 UI
      console.error('加入团队失败:', err)
      error.value = err?.message || '加入团队失败'
      if (!traceId.value) traceId.value = extractTraceId(err)
    } finally {
      isJoining.value = false
    }
  })
}

const rejectInvitation = async () => {
  // 这里可以添加拒绝邀请的逻辑
  // 例如，记录拒绝状态到后端
  success.value = '邀请已拒绝'
  
  // 跳转到首页
  setTimeout(() => {
    router.push('/')
  }, 1500)
}

onMounted(() => {
  checkInviteStatus()
})
</script>

<style scoped>
.people-join-container {
  min-height: 80vh;
}
</style>
