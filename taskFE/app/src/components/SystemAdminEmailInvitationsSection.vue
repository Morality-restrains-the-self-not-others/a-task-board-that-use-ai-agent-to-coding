<template>
  <div>
    <!-- 邮箱邀请模态框 -->
    <div v-if="showEmailInviteModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">发送邮箱注册邀请</h3>
          <button @click="closeEmailInviteModal" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        <form @submit.prevent="handleSendEmailInvite">
          <div class="space-y-4">
            <div>
              <label for="invite-email" class="block text-sm font-medium text-gray-700 mb-2">邮箱地址 <span class="text-red-500">*</span></label>
              <input
                type="email" id="invite-email" v-model="emailInviteForm.email"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="输入要邀请的邮箱地址" required
              />
            </div>
            <div>
              <label for="invite-role" class="block text-sm font-medium text-gray-700 mb-2">用户身份</label>
              <select
                id="invite-role" v-model="emailInviteForm.assigned_role"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              >
                <option value="">普通用户 (member)</option>
                <option value="staff">员工 (staff)</option>
                <option value="tenant">租户 (tenant)</option>
                <option value="superuser">超级管理员 (superuser)</option>
              </select>
              <p class="text-xs text-gray-400 mt-1">注册后自动赋予对应系统角色</p>
            </div>
            <div>
              <label for="invite-expiry" class="block text-sm font-medium text-gray-700 mb-2">账号有效期</label>
              <div class="flex items-center space-x-3">
                <input
                  type="date" id="invite-expiry" v-model="emailInviteForm.account_expires_at"
                  class="flex-1 px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                  :disabled="emailInviteForm.account_permanent"
                />
                <label class="flex items-center text-sm text-gray-600 whitespace-nowrap">
                  <input type="checkbox" v-model="emailInviteForm.account_permanent" class="mr-1 w-4 h-4 text-primary rounded">
                  永久有效
                </label>
              </div>
              <p class="text-xs text-gray-400 mt-1">留空或勾选"永久有效"则账号不过期</p>
            </div>
            <div>
              <label for="invite-reason" class="block text-sm font-medium text-gray-700 mb-2">邀请原因</label>
              <textarea
                id="invite-reason" v-model="emailInviteForm.invite_reason" rows="2"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="如：新员工入职、合作伙伴接入等（可选）"
              ></textarea>
            </div>
            <p class="text-sm text-gray-500">邀请链接有效期为7天，用户点击链接即可使用邮箱注册。</p>
            <div
              v-if="inviteResult"
              class="p-3 rounded-lg text-sm"
              :class="inviteResult.success && !inviteResult.skipped ? 'bg-green-50 text-green-700 border border-green-200' : inviteResult.skipped ? 'bg-amber-50 text-amber-900 border border-amber-200' : 'bg-red-50 text-red-700 border border-red-200'"
              :data-traceId="inviteTraceId || undefined"
            >
              <p v-if="!inviteResult.skipped">{{ inviteResult.message }}</p>
              <InviteCopyLinkPrompt
                v-if="inviteResult.skipped"
                class="mt-1"
                :hint="inviteResult.message"
                :url="inviteResult.invitationUrl"
              />
              <button
                v-if="!inviteResult.success && inviteResult.canResend"
                type="button"
                class="mt-2 px-3 py-1.5 bg-red-600 text-white text-xs rounded hover:bg-red-700 transition-colors"
                :disabled="resendingInvite"
                @click="handleResendEmailInvite"
              >
                {{ resendingInvite ? '发送中...' : '📧 重新发送邀请邮件' }}
              </button>
            </div>
          </div>
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="closeEmailInviteModal" class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="sendingInvite" class="px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-all duration-300">
              {{ sendingInvite ? '发送中...' : '发送邀请' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- 邮箱邀请记录 -->
    <div v-if="showList" class="mt-8 bg-white rounded-lg shadow-sm border border-gray-200 p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">📧 邮箱邀请记录</h2>
          <p class="text-sm text-gray-500 mt-1">最近发送的邮箱注册邀请及送达状态</p>
        </div>
        <div class="flex items-center space-x-3">
          <button
            v-if="invitations.some(inv => inv.canResend)"
            type="button"
            class="px-3 py-1 bg-amber-500 text-white text-sm rounded hover:bg-amber-600 transition-colors disabled:opacity-50"
            :disabled="bulkResending"
            @click="handleBulkResend"
          >
            {{ bulkResending ? '批量重发中...' : '📧 全部重发' }}
          </button>
          <button
            type="button"
            class="text-sm text-primary hover:underline disabled:opacity-50"
            :disabled="loadingInvitations"
            @click="loadInvitations"
          >
            {{ loadingInvitations ? '刷新中...' : '刷新' }}
          </button>
        </div>
      </div>
      <div v-if="loadingInvitations" class="text-sm text-gray-500 py-4">加载中...</div>
      <div v-else-if="!invitations.length" class="text-sm text-gray-500 py-4">暂无邀请记录。</div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="py-2 pr-3 font-medium">邮箱</th>
              <th class="py-2 pr-3 font-medium">邀请人</th>
              <th class="py-2 pr-3 font-medium">预设角色</th>
              <th class="py-2 pr-3 font-medium">账号有效期</th>
              <th class="py-2 pr-2 font-medium max-w-[80px]">邀请原因</th>
              <th class="py-2 pr-3 font-medium">状态</th>
              <th class="py-2 pr-3 font-medium">投递状态</th>
              <th class="py-2 pr-3 font-medium">发送次数</th>
              <th class="py-2 pr-3 font-medium">最后发送</th>
              <th class="py-2 pr-3 font-medium">过期时间</th>
              <th class="py-2 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="inv in invitations" :key="inv.id" class="border-b border-gray-100">
              <td class="py-2 pr-3">{{ inv.email }}</td>
              <td class="py-2 pr-3 text-xs text-gray-600">{{ inv.inviterName || '-' }}</td>
              <td class="py-2 pr-3">
                <span
                  v-if="inv.assignedRole"
                  class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                  :class="{
                    'bg-red-100 text-red-800': inv.assignedRole === 'superuser',
                    'bg-blue-100 text-blue-800': inv.assignedRole === 'staff',
                    'bg-purple-100 text-purple-800': inv.assignedRole === 'tenant',
                    'bg-gray-100 text-gray-600': inv.assignedRole === 'member'
                  }"
                >{{ {superuser: '超管', staff: '员工', tenant: '租户', member: '普通用户'}[inv.assignedRole] || inv.assignedRole }}</span>
                <span v-else class="text-xs text-gray-400">普通用户</span>
              </td>
              <td class="py-2 pr-3 text-xs text-gray-500">{{ inv.accountExpiresAt ? formatInviteTime(inv.accountExpiresAt) : '永久' }}</td>
              <td class="py-2 pr-2 text-xs text-gray-500 max-w-[80px] truncate" :title="inv.inviteReason || ''">{{ inv.inviteReason || '-' }}</td>
              <td class="py-2 pr-3">
                <span
                  class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                  :class="{
                    'bg-yellow-100 text-yellow-800': inv.status === 'pending',
                    'bg-green-100 text-green-800': inv.status === 'accepted',
                    'bg-gray-100 text-gray-600': inv.status === 'expired' || inv.status === 'cancelled'
                  }"
                >
                  {{ {pending: '待接受', accepted: '已接受', expired: '已过期', cancelled: '已取消'}[inv.status] || inv.status }}
                </span>
              </td>
              <td class="py-2 pr-3">
                <span
                  class="inline-block px-2 py-0.5 rounded-full text-xs font-medium cursor-default"
                  :class="{
                    'bg-green-100 text-green-800': inv.deliveryStatus === 'delivered',
                    'bg-amber-100 text-amber-800': inv.deliveryStatus === 'queued',
                    'bg-red-100 text-red-800': inv.deliveryStatus === 'failed',
                    'bg-slate-100 text-slate-700': inv.deliveryStatus === 'skipped_unsubscribed',
                    'bg-gray-100 text-gray-600': inv.deliveryStatus === 'pending' || !inv.deliveryStatus
                  }"
                  @mouseenter="showDeliveryPopover($event, deliveryTooltip(inv))"
                  @mouseleave="hideDeliveryPopover"
                >
                  {{ {delivered: '已送达', queued: '已排队', failed: '投递失败', skipped_unsubscribed: '已退订跳过', pending: '未发送'}[inv.deliveryStatus] || '未发送' }}
                </span>
              </td>
              <td class="py-2 pr-3">
                <span
                  class="cursor-help border-b border-dotted border-gray-400"
                  @mouseenter="showDeliveryPopover($event, deliveryAttemptsTooltip(inv))"
                  @mouseleave="hideDeliveryPopover"
                >{{ inv.emailSendAttempts ?? 0 }}</span>
              </td>
              <td class="py-2 pr-3 text-xs text-gray-500">{{ inv.emailSentAt ? formatInviteTime(inv.emailSentAt) : '—' }}</td>
              <td class="py-2 pr-3 text-xs text-gray-500">{{ formatInviteTime(inv.expiresAt) }}</td>
              <td class="py-2">
                <button
                  v-if="inv.canResend"
                  type="button"
                  class="px-2 py-1 bg-green-600 text-white text-xs rounded hover:bg-green-700 transition-colors disabled:opacity-50"
                  :disabled="resendingInvitationId === inv.id"
                  @click="handleResendInvitation(inv)"
                >
                  {{ resendingInvitationId === inv.id ? '发送中...' : '重新发送' }}
                </button>
                <span v-else class="text-xs text-gray-400">—</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        v-if="invitationListError"
        class="mt-2 p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="invitationListTraceId || undefined"
      >
        {{ invitationListError }}
      </div>
    </div>

    <!-- Teleport-OK: overflow-clip — 投递错误气泡在表格滚动容器内会被裁剪 -->
    <DeliveryErrorPopover
      :show="deliveryPopover.show"
      :content="deliveryPopover.content"
      :x="deliveryPopover.x"
      :y="deliveryPopover.y"
    />
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { ref, reactive, onMounted } from 'vue'
import InviteCopyLinkPrompt from './InviteCopyLinkPrompt.vue'
import DeliveryErrorPopover from './DeliveryErrorPopover.vue'
import { deliveryAttemptsTooltip, deliveryTooltip, formatInviteTime } from './systemAdminEmailInviteFormat.js'

defineProps({
  showList: { type: Boolean, default: true },
})

const showEmailInviteModal = ref(false)
const sendingInvite = ref(false)
const resendingInvite = ref(false)
const emailInviteForm = reactive({ email: '', assigned_role: '', account_expires_at: '', account_permanent: false, invite_reason: '' })
const inviteResult = ref(null)
const inviteTraceId = ref('')

const invitations = ref([])
const loadingInvitations = ref(false)
const resendingInvitationId = ref(null)
const bulkResending = ref(false)
const invitationListError = ref('')
const invitationListTraceId = ref('')

const deliveryPopover = reactive({ show: false, content: '', x: 0, y: 0 })

// OPT-20260819-038: 邮箱邀请均为写操作，防连点/超时重试双发 POST
const sendInviteGuard = createClickGuard()
const resendInviteGuard = createClickGuard()
const resendInvitationGuard = createClickGuard()
const bulkResendGuard = createClickGuard()

function showDeliveryPopover(event, content) {
  if (!content) return
  const rect = event.target.getBoundingClientRect()
  deliveryPopover.content = content
  deliveryPopover.x = rect.left + window.scrollX
  deliveryPopover.y = rect.bottom + window.scrollY + 4
  deliveryPopover.show = true
}

function hideDeliveryPopover() {
  deliveryPopover.show = false
}

function openEmailInviteModal() {
  emailInviteForm.email = ''
  emailInviteForm.assigned_role = ''
  emailInviteForm.account_expires_at = ''
  emailInviteForm.account_permanent = false
  emailInviteForm.invite_reason = ''
  inviteResult.value = null
  inviteTraceId.value = ''
  resendingInvite.value = false
  showEmailInviteModal.value = true
}

function closeEmailInviteModal() {
  showEmailInviteModal.value = false
}

async function handleSendEmailInvite() {
  if (!emailInviteForm.email.trim()) return
  // OPT-20260819-038: 发送邀请是写操作，防连点/超时重试双发 POST
  await sendInviteGuard.run(async ({ idempotencyKey }) => {
    sendingInvite.value = true
    inviteResult.value = null
    inviteTraceId.value = ''
    try {
      const body = { email: emailInviteForm.email.trim() }
      if (emailInviteForm.assigned_role) body.assigned_role = emailInviteForm.assigned_role
      if (!emailInviteForm.account_permanent && emailInviteForm.account_expires_at) {
        body.account_expires_at = emailInviteForm.account_expires_at + 'T00:00:00Z'
      }
      if (emailInviteForm.invite_reason.trim()) body.invite_reason = emailInviteForm.invite_reason.trim()
      const r = await apiFetch('/api/system-admin/email-invitations/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(body),
      })
      const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
      inviteTraceId.value = traceId
      if (r.ok) {
        if (d.email_skipped) {
          inviteResult.value = {
            success: true,
            skipped: true,
            message: d.message || '该邮箱已退订邮件邀请，请手动复制邀请链接给对方',
            invitationUrl: d.invitation_url || '',
          }
        } else {
          inviteResult.value = { success: true, message: `邀请已发送至 ${emailInviteForm.email}，有效期7天` }
          emailInviteForm.email = ''
          emailInviteForm.assigned_role = ''
          emailInviteForm.account_expires_at = ''
          emailInviteForm.account_permanent = false
          emailInviteForm.invite_reason = ''
        }
        loadInvitations()
      } else {
        const errMsg = (d && (d.error || d.detail || d.message)) || '发送失败，请稍后重试'
        const canResend = !!(d && d.can_resend)
        inviteResult.value = { success: false, message: errMsg, canResend }
      }
    } catch {
      inviteResult.value = { success: false, message: '网络错误，请稍后重试' }
    } finally {
      sendingInvite.value = false
    }
  })
}

async function handleResendEmailInvite() {
  if (!emailInviteForm.email.trim()) return
  // OPT-20260819-038: 重发邀请是写操作，防连点双发 POST
  await resendInviteGuard.run(async ({ idempotencyKey }) => {
    resendingInvite.value = true
    inviteTraceId.value = ''
    try {
      const r = await apiFetch('/api/system-admin/email-invitations/resend/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ email: emailInviteForm.email.trim() }),
      })
      const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
      inviteTraceId.value = traceId
      if (r.ok) {
        if (d.email_skipped) {
          inviteResult.value = {
            success: true,
            skipped: true,
            message: d.message || '该邮箱已退订邮件邀请，请手动复制邀请链接给对方',
            invitationUrl: d.invitation_url || d.invite_url || '',
          }
        } else {
          inviteResult.value = { success: true, message: `邀请邮件已重新发送至 ${emailInviteForm.email}` }
          emailInviteForm.email = ''
        }
        loadInvitations()
      } else {
        const errMsg = (d && (d.error || d.detail || d.message)) || '重发失败，请稍后重试'
        const canResend = !!(d && d.can_resend)
        inviteResult.value = { success: false, message: errMsg, canResend }
      }
    } catch {
      inviteResult.value = { success: false, message: '网络错误，请稍后重试' }
    } finally {
      resendingInvite.value = false
    }
  })
}

async function loadInvitations() {
  loadingInvitations.value = true
  invitationListError.value = ''
  invitationListTraceId.value = ''
  try {
    const r = await apiFetch('/api/system-admin/email-invitations/', {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
    invitationListTraceId.value = traceId || ''
    if (r.ok) {
      invitations.value = Array.isArray(d?.invitations) ? d.invitations : []
    } else {
      invitationListError.value = (d && (d.error || d.detail || d.message)) || '加载失败'
      invitations.value = []
    }
  } catch (err) {
    invitationListError.value = err?.message || '加载失败'
    invitationListTraceId.value = err?.traceId || ''
    invitations.value = []
  } finally {
    loadingInvitations.value = false
  }
}

async function handleResendInvitation(inv) {
  if (!inv?.email) return
  // OPT-20260819-038: 列表行重发是写操作，防连点双发 POST
  await resendInvitationGuard.run(async ({ idempotencyKey }) => {
    resendingInvitationId.value = inv.id
    invitationListTraceId.value = ''
    try {
      const r = await apiFetch('/api/system-admin/email-invitations/resend/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ email: inv.email }),
      })
      const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
      invitationListTraceId.value = traceId || ''
      if (r.ok) {
        await loadInvitations()
      } else {
        invitationListError.value = (d && (d.error || d.detail || d.message)) || '重发失败'
      }
    } catch (err) {
      invitationListError.value = err?.message || '重发失败'
      invitationListTraceId.value = err?.traceId || ''
    } finally {
      resendingInvitationId.value = null
    }
  })
}

async function handleBulkResend() {
  // OPT-20260819-038: 批量重发是写操作，防连点双发 POST
  await bulkResendGuard.run(async ({ idempotencyKey }) => {
    bulkResending.value = true
    invitationListError.value = ''
    invitationListTraceId.value = ''
    try {
      const r = await apiFetch('/api/system-admin/email-invitations/bulk-resend/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
      })
      const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
      invitationListTraceId.value = traceId || ''
      if (r.ok) {
        invitationListError.value = d?.message || '批量重发完成'
        await loadInvitations()
      } else {
        invitationListError.value = (d && (d.error || d.detail || d.message)) || '批量重发失败'
      }
    } catch (err) {
      invitationListError.value = err?.message || '批量重发失败'
      invitationListTraceId.value = err?.traceId || ''
    } finally {
      bulkResending.value = false
    }
  })
}

onMounted(() => {
  loadInvitations()
})

defineExpose({ openEmailInviteModal })
</script>
