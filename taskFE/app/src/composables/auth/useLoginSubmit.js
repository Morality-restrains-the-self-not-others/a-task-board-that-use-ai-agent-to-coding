import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { resolvePostLoginRedirectUrl } from '../../utils/resolvePostLoginRedirect.js'
import { persistLoginSuccessCredentials } from '../../domain/auth/services/activate_session_service.js'
import {
  promptPostLoginPhoneVerify,
  shouldSkipPhoneVerifyPrompt,
} from '../../domain/auth/services/post_login_phone_verify_prompt_service.js'
import { apiFetch, extractErrorMessage } from '../../utils/apiUtils.js'
import modalService from '../../utils/modalService.js'
import { extractTraceId } from '../../utils/traceId.js'

function readIsImpersonating() {
  try {
    return Boolean(
      typeof sessionStorage !== 'undefined' && sessionStorage.getItem('impersonatorAccountBackup'),
    )
  } catch {
    return false
  }
}

export function useLoginSubmit({
  emit,
  loginMethod,
  phoneFields,
  legalConsent,
  accessTokenUsernameInput,
  accessTokenInput,
  // OPT-20260824-001: 管理员登录入口分离。true 时提交到 /api/auth/admin-login/
  // （后端仅放行平台管理员/员工账号），并跳过客户入口的法律条款勾选门禁。
  adminLogin = false,
}) {
  const route = useRoute()
  const isLoading = ref(false)
  const showResendButton = ref(false)
  const lastLoginEmail = ref('')
  const isResendingEmail = ref(false)

  const {
    isPhoneValidForPassword,
    loginPhonePasswordForApi,
  } = phoneFields

  const {
    acceptPrivacy,
    acceptLicense,
    currentPrivacyPolicy,
    currentLicenseAgreement,
    privacyLoadError,
    licenseLoadError,
  } = legalConsent

  const isLoginSubmitDisabled = computed(() => {
    if (isLoading.value) {
      return true
    }
    // 仅当法律文档已成功加载且用户未勾选同意时才禁用按钮。
    // 若法律文档加载失败或不存在，允许提交（后端将根据用户角色决定是否强制验证）。
    // 管理员入口无法律条款勾选 UI，不适用此门禁。
    if (!adminLogin && !privacyLoadError.value && currentPrivacyPolicy.value && !acceptPrivacy.value) {
      return true
    }
    if (!adminLogin && !licenseLoadError.value && currentLicenseAgreement.value && !acceptLicense.value) {
      return true
    }
    if (loginMethod.value === 'phonePassword') {
      return !isPhoneValidForPassword.value
    }
    if (loginMethod.value === 'accessToken') {
      return !accessTokenUsernameInput.value.trim() || !accessTokenInput.value.trim()
    }
    return false
  })

  const loginSubmitDisabledReason = computed(() => {
    if (isLoading.value) {
      return '登录处理中，请稍候'
    }
    if (!adminLogin && !acceptPrivacy.value && !privacyLoadError.value && currentPrivacyPolicy.value) {
      return '请勾选同意隐私政策'
    }
    if (!adminLogin && !acceptLicense.value && !licenseLoadError.value && currentLicenseAgreement.value) {
      return '请勾选同意服务协议'
    }
    if (loginMethod.value === 'phonePassword' && !isPhoneValidForPassword.value) {
      return '请填写正确手机号'
    }
    if (loginMethod.value === 'accessToken') {
      if (!accessTokenUsernameInput.value.trim()) {
        return '请填写账号（用户名或邮箱）'
      }
      if (!accessTokenInput.value.trim()) {
        return '请填写访问令牌'
      }
    }
    return '提交登录'
  })

  const handleResendActivationEmail = async () => {
    try {
      if (!lastLoginEmail.value) {
        modalService.alert('请先输入邮箱并尝试登录')
        return
      }

      isResendingEmail.value = true

      const response = await apiFetch('/api/accounts/users/resend_activation_email/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          Accept: 'application/json',
        },
        body: JSON.stringify({ email: lastLoginEmail.value }),
      })

      const responseData = await response.json().catch(() => ({}))

      if (response.ok) {
        modalService.alert('激活邮件已重新发送，请查收')
      } else {
        const errorMessage = responseData.error || '发送激活邮件失败'
        modalService.alert(`发送失败：${errorMessage}`)
      }
    } catch (error) {
      console.error('Resend activation email error:', error)
      modalService.alert('发送激活邮件失败：网络错误，请稍后重试')
    } finally {
      isResendingEmail.value = false
    }
  }

  const handleSubmit = async (event) => {
    try {
      if (event) {
        event.preventDefault()
      }
      window.userInitiatedLogin = true
      isLoading.value = true

      // 管理员入口无法律条款勾选 UI（OPT-20260824-001），跳过勾选门禁
      if (!adminLogin && !acceptPrivacy.value && !privacyLoadError.value && currentPrivacyPolicy.value) {
        modalService.alert('请阅读并勾选同意隐私政策后再登录')
        isLoading.value = false
        return
      }

      if (!adminLogin && !acceptLicense.value && !licenseLoadError.value && currentLicenseAgreement.value) {
        modalService.alert('请阅读并勾选同意服务协议后再登录')
        isLoading.value = false
        return
      }

      let formData = {}
      if (loginMethod.value === 'emailPassword') {
        formData = {
          email: document.getElementById('email')?.value || '',
          password: document.getElementById('password')?.value || '',
          rememberMe: document.getElementById('remember-me')?.checked || false,
        }
      } else if (loginMethod.value === 'phonePassword') {
        if (!isPhoneValidForPassword.value) {
          modalService.alert('请填写带国家码的有效手机号')
          isLoading.value = false
          return
        }
        formData = {
          phone: loginPhonePasswordForApi.value,
          password: document.getElementById('password')?.value || '',
          rememberMe: document.getElementById('remember-me')?.checked || false,
        }
      } else if (loginMethod.value === 'accessToken') {
        if (!accessTokenUsernameInput.value.trim()) {
          modalService.alert('请填写账号（用户名或邮箱）')
          isLoading.value = false
          return
        }
        if (!accessTokenInput.value.trim()) {
          modalService.alert('请输入访问令牌')
          isLoading.value = false
          return
        }
        formData = {
          username: accessTokenUsernameInput.value.trim(),
          access_token: accessTokenInput.value.trim(),
          rememberMe: document.getElementById('remember-me')?.checked || false,
        }
      }

      let requestBody = {}

      if (loginMethod.value === 'emailPassword') {
        requestBody = {
          username: formData.email,
          password: formData.password,
          rememberMe: formData.rememberMe,
          accepted_privacy_policy_id: String(currentPrivacyPolicy.value?.id || ''),
          accepted_license_agreement_id: String(currentLicenseAgreement.value?.id || ''),
        }
      } else if (loginMethod.value === 'phonePassword') {
        requestBody = {
          phone: formData.phone,
          password: formData.password,
          rememberMe: formData.rememberMe,
          accepted_privacy_policy_id: String(currentPrivacyPolicy.value?.id || ''),
          accepted_license_agreement_id: String(currentLicenseAgreement.value?.id || ''),
        }
      } else if (loginMethod.value === 'accessToken') {
        requestBody = {
          username: formData.username,
          access_token: formData.access_token,
        }
      }

      // OPT-20260824-001: 入口分离 — 管理员表单提交到专用端点 /api/auth/admin-login/
      // （后端仅放行管理员账号）；客户表单仍走 /api/auth/ 或 access-token 端点。
      const loginEndpoint = adminLogin
        ? '/api/auth/admin-login/'
        : loginMethod.value === 'accessToken'
          ? '/api/accounts/users/login-with-access-token/'
          : '/api/auth/'

      const response = await apiFetch(loginEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          Accept: 'application/json',
        },
        body: JSON.stringify(requestBody),
      })

      // 使用 response.json() 以保证 apiFetch 注入的 _traceId 生效（response.text() 会绕过注入）
      let data = {}
      try {
        data = await response.json()
      } catch (error) {
        console.error('解析响应JSON失败:', error)
      }
      console.log('[Login] 响应 status=', response.status, 'hasToken=', Boolean(data.token), 'userId=', data?.user?.id || null)

      if (response.ok) {
        emit('login-success')
        showResendButton.value = false

        // 等待凭据落盘（activate-session 的 HttpOnly userId+token 会话 cookie / 账号槽）
        // 再跳转：此前 fire-and-forget + 立即 location.href 会取消在途 activate-session
        // 请求，Set-Cookie 未落 → 跳转后仅剩前端裸 userId cookie，网关 forward-auth
        // 解析失败 → 全站「无法解析登录凭据，请重新登录」。
        // 上限 5s：activate-session 正常 <1s；后端异常慢时不阻塞登录跳转。
        await Promise.race([
          persistLoginSuccessCredentials(data),
          new Promise((resolve) => setTimeout(resolve, 5000)),
        ])
        // OPT-20260807-015：多账号槽位已彻底下线，登录不再接受 add_account 续加槽位。
        const redirectUrl = resolvePostLoginRedirectUrl(route.query?.next, data || {}, localStorage)
        const skipPrompt = shouldSkipPhoneVerifyPrompt({
          adminLogin,
          loginMethod: loginMethod.value,
          isImpersonating: readIsImpersonating(),
        })
        const decision = await promptPostLoginPhoneVerify({
          skipPrompt,
          source: data?.user,
          redirectUrl,
          acknowledge: (message, title, confirmText) =>
            modalService.alert(message, title, { confirmText, showCloseButton: false }),
        })
        console.log('[Login] 登录成功，重定向到:', decision.href, 'phoneVerify=', decision.action)
        window.location.href = decision.href
      } else {
        const rawMessage = extractErrorMessage(data, response, '登录失败')
        // 按元规则从 apiFetch 挂载的 response.traceId 获取（优先于 headers，因 CORS 可能未暴露 X-Trace-Id）；
        // response.traceId 由 apiFetch 的 attachTraceIdToResponse 设置，回退链路：响应头→请求头→响应体
        const traceId = response.traceId || extractTraceId(data) || extractTraceId(response.headers) || ''
        // 避免「登录失败：登录失败」的冗余展示：当后端已返回含"登录失败"前缀的消息时不重复拼接
        const errorMessage = rawMessage.startsWith('登录失败') ? rawMessage : `登录失败：${rawMessage}`
        emit('login-failure', errorMessage)

        console.log('[Login] 登录失败，错误信息:', errorMessage, 'traceId:', traceId)

        if (rawMessage.includes('邮箱未验证') || rawMessage.includes('email not verified')) {
          if (loginMethod.value === 'emailPassword') {
            lastLoginEmail.value = document.getElementById('email')?.value || ''
          }
          showResendButton.value = true
          modalService.alert({
            message: errorMessage,
            traceId,
            showCancel: false,
            confirmText: '确定',
            additionalActions: [
              {
                text: '重新发送激活邮件',
                callback: handleResendActivationEmail,
              },
            ],
            onConfirm: () => {},
          })
        } else {
          showResendButton.value = false
          modalService.alert({ message: errorMessage, traceId })
        }
      }
    } catch (error) {
      console.error('Login error:', error)
      emit('login-failure', error.message)
      modalService.alert('登录失败：网络错误，请稍后重试')
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading,
    showResendButton,
    isResendingEmail,
    isLoginSubmitDisabled,
    loginSubmitDisabledReason,
    handleSubmit,
    handleResendActivationEmail,
  }
}
