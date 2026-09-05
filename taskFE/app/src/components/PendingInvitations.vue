<template>
  <div class="bg-white rounded-xl shadow-md p-6">
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-lg font-semibold">待处理邀请</h3>
      <button 
        @click="fetchPendingInvitations"
        class="text-primary hover:text-primary/80 flex items-center"
      >
        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
        </svg>
        刷新
      </button>
    </div>
    
    <div v-if="invitationLoading" class="text-center py-8">
      加载中...
    </div>
    
    <div v-else-if="pendingInvitations.length > 0" class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b border-border">
            <th class="text-left py-3 px-4 font-medium text-text-light">工作空间名称</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">邀请成员名称</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">邀请渠道</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">人数</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">邀请目标</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">邀请链接</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">角色</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">创建时间</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">过期时间</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">状态</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">投递时间</th>
            <th class="text-right py-3 px-4 font-medium text-text-light">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="invite in pendingInvitations" :key="invite.id" class="border-b border-border hover:bg-primary/5">
            <td class="py-3 px-4">{{ invite.workspace_name || '-' }}</td>
            <td class="py-3 px-4">{{ invite.company_member_name || invite.username || '-' }}</td>
            <td class="py-3 px-4">{{ formatInviteMethod(invite) }}</td>
            <td class="py-3 px-4" data-testid="pending-invite-uses">{{ formatInviteUseLabel(invite) }}</td>
            <td class="py-3 px-4">{{ invite.invite_target || '-' }}</td>
            <td class="py-3 px-4">
              <div class="flex items-start gap-2">
                <a 
                  :href="`/tenant/${tenantId}/people/join/?token=${invite.token}`" 
                  target="_blank"
                  class="text-primary hover:underline text-sm break-all flex-1"
                >
                  {{ invite.token }}
                </a>
                <button 
                  @click="copyInviteLink(invite.token)"
                  class="text-gray-500 hover:text-primary transition-colors p-1"
                  title="复制邀请链接"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                  </svg>
                </button>
              </div>
            </td>
            <td class="py-3 px-4">
              <span v-if="invite.is_admin" class="bg-primary/10 text-primary px-2 py-1 rounded text-sm">管理员</span>
              <span v-else class="bg-gray-100 text-gray-600 px-2 py-1 rounded text-sm">成员</span>
            </td>
            <td class="py-3 px-4">{{ formatDate(invite.created_at) }}</td>
            <td class="py-3 px-4">{{ formatDate(invite.expires_at) }}</td>
            <td class="py-3 px-4">
              <!-- 投递状态：delivered=已投递 / queued=投递中 / failed=投递失败 / none=链接分享；
                   老数据无 delivery_status 时回退展示接受状态 -->
              <span
                v-if="deliveryStatusBadge(invite).text"
                :class="deliveryStatusBadge(invite).class"
                :title="deliveryStatusBadge(invite).title || undefined"
                class="px-2 py-1 rounded text-sm"
              >{{ deliveryStatusBadge(invite).text }}</span>
            </td>
            <td class="py-3 px-4">
              <!-- 投递时间：delivered 后由 delivery-callback 落库；无投递动作（link 渠道/老数据）显示 '-' -->
              <span v-if="invite.email_sent_at">{{ formatDate(invite.email_sent_at) }}</span>
              <span v-else class="text-text-light">-</span>
            </td>
            <td class="py-3 px-4 text-right">
              <button
                @click="resendInvitationLink(invite)"
                class="text-primary hover:text-primary/80 mr-3"
              >
                重新发送
              </button>
              <button 
                @click="revokeInvitation(invite.id)"
                class="text-danger hover:text-danger/80"
              >
                撤销
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    
    <div v-if="!invitationLoading && pendingInvitations.length === 0" class="text-center py-8 text-text-light">
      暂无待处理邀请
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import toastService from '../utils/toastService'
import modalService from '../utils/modalService.js'
import { toastRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { formatInviteUseLabel } from '../utils/peopleInviteLinkPayload.js'
import { DEFAULT_INVITE_EXPIRATION_DAYS } from '../utils/peopleInviteExpiration.js'

const props = defineProps({
  tenantId: {
    type: String,
    required: true
  }
})

const { tenantId } = props
const pendingInvitations = ref([])
const invitationLoading = ref(false)

const revokeInvitationGuard = createClickGuard()
const resendInvitationGuard = createClickGuard()

const fetchPendingInvitations = async () => {
  invitationLoading.value = true
  
  try {
    const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/pending-invitations/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw new Error('获取待处理邀请失败')
    }
    
    const data = await response.json()
    pendingInvitations.value = data
  } catch (error) {
    console.error('获取待处理邀请失败:', error)
    toastRequestError(error.message || '获取待处理邀请失败', error)
  } finally {
    invitationLoading.value = false
  }
}

const revokeInvitation = async (inviteId) => {
  const confirmed = await modalService.confirm('确定要撤销此邀请吗？', '确认')
  if (!confirmed) {
    return
  }
  
  // OPT-20260819-038: 撤销邀请是成员写操作，防连点/超时重试双发
  await revokeInvitationGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/${inviteId}/revoke-invitation/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', 'Accept': 'application/json' },
          idempotencyKey,
        ),
      })

      if (!response.ok) {
        throw new Error('撤销邀请失败')
      }

      // 刷新待处理邀请列表
      await fetchPendingInvitations()
    } catch (error) {
      console.error('撤销邀请失败:', error)
      toastRequestError(error.message || '撤销邀请失败', error)
    } finally {}
  })
}

const resendInvitationLink = async (invite) => {
  const targetText = invite.invite_method === 'email'
    ? `邮箱 ${invite.invite_target || ''}`
    : invite.invite_method === 'phone'
      ? `手机号 ${invite.invite_target || ''}`
      : '邀请链接'
  const confirmed = await modalService.confirm(`确定重新发送邀请到${targetText}吗？`, '确认')
  if (!confirmed) {
    return
  }

  // OPT-20260819-038: 重发邀请是成员写操作，防连点/超时重试双发
  await resendInvitationGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/${invite.id}/resend-invitation-link/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', 'Accept': 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ expiration_days: DEFAULT_INVITE_EXPIRATION_DAYS })
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || '重新发送邀请链接失败')
      }

      const result = await response.json()
      invite.token = result.invite_token || invite.token
      invite.expires_at = result.expires_at || invite.expires_at

      const newInviteLink = `${window.location.origin}/tenant/${tenantId}/people/join/?token=${invite.token}`
      await navigator.clipboard.writeText(newInviteLink)
      toastService.success('已重新发送邀请并复制最新链接到剪贴板', 1500)
    } catch (error) {
      console.error('重新发送邀请链接失败:', error)
      toastRequestError(error.message || '重新发送邀请链接失败', error, 1500)
    }
  })
}

const copyInviteLink = async (token) => {
  try {
    const inviteLink = `${window.location.origin}/tenant/${tenantId}/people/join/?token=${token}`
    await navigator.clipboard.writeText(inviteLink)
    toastService.success('邀请链接已复制到剪贴板！', 1000)
  } catch (error) {
    console.error('复制邀请链接失败:', error)
    toastService.error('复制失败，请手动复制链接', 1000)
  }
}

const formatDate = (dateString) => {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatInviteMethod = (invite) => {
  if (invite.invite_method === 'email') return '邮箱'
  if (invite.invite_method === 'phone') return '短信'
  if (invite.link_kind === 'open') return '开放链接'
  if (invite.invite_method === 'link') return '分享链接'
  return '分享链接'
}

// 投递状态徽章映射（后端 tenant_invitation.delivery_status）：
// delivered=SMTP/短信已送达；queued=事件已发布待异步投递；failed=投递失败；
// none=link 渠道无投递动作；缺失（迁移前老数据）→ 回退「待接受」。
const deliveryStatusBadge = (invite) => {
  const status = invite.delivery_status
  switch (status) {
    case 'delivered':
      return { text: '已投递', class: 'bg-success/10 text-success', title: '' }
    case 'queued':
      return { text: '投递中', class: 'bg-warning/10 text-warning', title: '' }
    case 'failed':
      return { text: '投递失败', class: 'bg-danger/10 text-danger', title: invite.delivery_error || '投递失败' }
    case 'skipped_unsubscribed':
      return { text: '已退订，请复制链接', class: 'bg-gray-100 text-gray-700', title: '该邮箱已退订邮件邀请' }
    case 'none':
      return { text: '链接分享', class: 'bg-gray-100 text-gray-600', title: '' }
    default:
      // 兼容老数据（无 delivery_status 列）与已接受记录
      return invite.is_accepted
        ? { text: '已接受', class: 'bg-success/10 text-success', title: '' }
        : { text: '待接受', class: 'bg-warning/10 text-warning', title: '' }
  }
}

// 组件挂载时获取待处理邀请
onMounted(() => {
  fetchPendingInvitations()
})
</script>

<style scoped>
/* 组件样式 */
</style>