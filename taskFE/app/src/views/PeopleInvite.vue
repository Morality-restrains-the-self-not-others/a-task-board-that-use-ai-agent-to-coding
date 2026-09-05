<template>
  <!-- OPT-20260809-025: 非管理员友好提示 — 先探权限，无权时展示「无权限」空态而非原始 403 -->
  <div v-if="!accessChecked" class="people-invite-container p-6 text-text-light">加载中…</div>
  <div v-else-if="!canManage" class="people-invite-container p-6">
    <div class="bg-white p-6 rounded-xl shadow text-center py-16">
      <p class="text-xl font-bold">无权限访问</p>
      <p class="text-text-light mt-3">邀请仅对拥有成员管理权限的公司成员开放，如需访问请联系公司管理员。</p>
    </div>
  </div>
  <div v-else class="people-invite-container p-6" data-rg-key="people.invite.main">
    <h2 class="text-2xl font-bold mb-6">邀请人</h2>
    
    <div class="bg-white rounded-xl shadow-md p-6">
      <!-- 工作空间选择 -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-text mb-2">工作空间</label>
        <select 
          v-model="formData.workspace_id" 
          class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
          :disabled="workspacesLoading"
        >
          <option value="">请选择工作空间</option>
          <option v-for="workspace in workspaces" :key="workspace.id" :value="workspace.id">
            {{ workspace.name }}
          </option>
        </select>
      </div>
      
      <!-- 邀请方式选择 -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-text mb-2">邀请方式</label>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <label class="flex items-center p-3 border border-border rounded-lg cursor-pointer transition-colors" :class="{'border-primary bg-primary/5': inviteMethod === 'email'}">
            <input 
              type="radio" 
              v-model="inviteMethod" 
              value="email" 
              class="mr-2"
            >
            <span>邮箱</span>
          </label>
          <label class="flex items-center p-3 border border-border rounded-lg cursor-pointer transition-colors" :class="{'border-primary bg-primary/5': inviteMethod === 'phone'}">
            <input 
              type="radio" 
              v-model="inviteMethod" 
              value="phone" 
              class="mr-2"
            >
            <span>电话号码</span>
          </label>
          <label class="flex items-center p-3 border border-border rounded-lg cursor-pointer transition-colors" :class="{'border-primary bg-primary/5': inviteMethod === 'link'}">
            <input 
              type="radio" 
              v-model="inviteMethod" 
              value="link" 
              class="mr-2"
            >
            <span>复制邀请链接</span>
          </label>
        </div>
      </div>

      <!-- 页面/区域预授（三种邀请方式共用） -->
      <div class="mb-6 space-y-4">
        <div>
          <label for="role-shared" class="block text-sm font-medium text-text mb-1">角色</label>
          <select
            id="role-shared"
            v-model="formData.role"
            class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
            required
          >
            <option value="member">成员</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <!-- OPT-20260811-082: v75 角色优先 — 多选可复用角色，join 直接按角色绑定 -->
        <div v-if="formData.role !== 'admin' && availableRoles.length" data-testid="people-invite-role-options">
          <label class="block text-sm font-medium text-text mb-1">可复用角色（可选）</label>
          <p class="text-xs text-text-light mb-2">邀请加入后按所选角色绑定权限；不选则按下方「页面/区域预授」。</p>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
            <label
              v-for="role in availableRoles"
              :key="role.id"
              class="flex items-center p-2 border border-border rounded-lg cursor-pointer transition-colors"
              :class="{ 'border-primary bg-primary/5': selectedRoleNames.includes(role.name) }"
            >
              <input
                type="checkbox"
                :value="role.name"
                v-model="selectedRoleNames"
                class="mr-2"
                :data-testid="`people-invite-role-${role.name}`"
              >
              <span>{{ role.display_name || role.name }}</span>
            </label>
          </div>
        </div>
        <InviteAccessGrants
          :company-id="String(tenantId)"
          :disabled="formData.role === 'admin'"
          @update:grants="onGrantsUpdate"
        />
      </div>
      
      <!-- 邮箱邀请表单 -->
      <form v-if="inviteMethod === 'email'" @submit.prevent="inviteUser" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="email" class="block text-sm font-medium text-text mb-1">邮箱</label>
            <input 
              type="email" 
              id="email" 
              v-model="formData.email" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
          </div>
          <div>
            <label for="username" class="block text-sm font-medium text-text mb-1">成员名称</label>
            <input 
              type="text" 
              id="company_member_name" 
              v-model="formData.company_member_name" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
          </div>
        </div>
        
        <div>
          <label for="message" class="block text-sm font-medium text-text mb-1">邀请消息（可选）</label>
          <textarea
            id="message"
            v-model="formData.message"
            rows="3"
            class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
          ></textarea>
        </div>

        <div>
          <label class="block text-sm font-medium text-text mb-1">有效期</label>
          <InviteExpirationSelect v-model="formData.expiration_days" />
        </div>

        <div class="flex justify-end">
          <button
            type="submit"
            class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors flex items-center"
            :disabled="isLoading"
          >
            <span v-if="isLoading">邀请中...</span>
            <span v-else>发送邀请</span>
          </button>
        </div>
      </form>
      
      <!-- 电话号码邀请表单 -->
      <form v-else-if="inviteMethod === 'phone'" @submit.prevent="inviteUser" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="phone" class="block text-sm font-medium text-text mb-1">电话号码</label>
            <input 
              type="tel" 
              id="phone" 
              v-model="formData.phone" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
          </div>
          <div>
            <label for="username" class="block text-sm font-medium text-text mb-1">成员名称</label>
            <input 
              type="text" 
              id="company_member_name" 
              v-model="formData.company_member_name" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
          </div>
        </div>
        
        <div>
          <label for="message-phone" class="block text-sm font-medium text-text mb-1">邀请消息（可选）</label>
          <textarea
            id="message-phone"
            v-model="formData.message"
            rows="3"
            class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
          ></textarea>
        </div>

        <div>
          <label class="block text-sm font-medium text-text mb-1">有效期</label>
          <InviteExpirationSelect v-model="formData.expiration_days" />
        </div>

        <div class="flex justify-end">
          <button
            type="submit"
            class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors flex items-center"
            :disabled="isLoading"
          >
            <span v-if="isLoading">邀请中...</span>
            <span v-else>发送邀请</span>
          </button>
        </div>
      </form>
      
      <InviteLinkMethodPanel
        v-else-if="inviteMethod === 'link'"
        v-model:company-member-name="formData.company_member_name"
        v-model:expiration-days="formData.expiration_days"
        v-model:link-kind="linkKind"
        v-model:max-uses="maxUses"
        :workspace-id="formData.workspace_id"
        :is-loading="isLoading"
        :invite-link="inviteLink"
        :invite-token="inviteToken"
        :copy-success="copySuccess"
        @generate="generateInviteLink"
        @copy="copyInviteLink"
      />

      <InviteCopyLinkPrompt
        v-if="skipInvite"
        class="mt-4"
        :hint="skipInvite.hint"
        :url="skipInvite.url"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import toastService from '../utils/toastService'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { usePermissions } from '../composables/usePermissions.js'
import InviteAccessGrants from '../components/InviteAccessGrants.vue'
import InviteCopyLinkPrompt from '../components/InviteCopyLinkPrompt.vue'
import InviteExpirationSelect from '../components/InviteExpirationSelect.vue'
import InviteLinkMethodPanel from '../components/InviteLinkMethodPanel.vue'
import { buildLinkInviteBody, memberNameRequiredForLink } from '../utils/peopleInviteLinkPayload.js'
import { DEFAULT_INVITE_EXPIRATION_DAYS } from '../utils/peopleInviteExpiration.js'

const route = useRoute()
const tenantId = route.params.tenant

const perms = usePermissions()

// OPT-20260809-025: 权限已探测完成才渲染内容；canManage 按 v63 RBAC 权限码判定
//（member:manage 与后端邀请接口强制一致）
const accessChecked = ref(false)
const canManage = computed(() => perms.hasPerm(tenantId, 'member:manage'))

// 邀请方式
const inviteMethod = ref('email')

const formData = ref({
  email: '',
  phone: '',
  company_member_name: '',
  role: 'member',
  message: '',
  expiration_days: DEFAULT_INVITE_EXPIRATION_DAYS,
  workspace_id: ''
})

/** @type {import('vue').Ref<Array<{group_key:string,effect:string}>>} */
const pendingGrants = ref([])

function onGrantsUpdate(list) {
  pendingGrants.value = Array.isArray(list) ? list : []
}

// OPT-20260811-082: 可复用角色多选（v75 角色优先；角色真源在 taskAuth /api/auth/roles/）
/** @type {import('vue').Ref<Array<{id:string,name:string,display_name?:string,is_system?:boolean}>>} */
const availableRoles = ref([])
/** @type {import('vue').Ref<string[]>} */
const selectedRoleNames = ref([])
const rolesLoading = ref(false)

async function fetchRoles() {
  rolesLoading.value = true
  try {
    const response = await apiFetch(`/api/auth/roles/company_id/${tenantId}`, {
      method: 'GET',
      credentials: 'include',
      headers: { 'Accept': 'application/json' },
    })
    if (response.ok) {
      const body = await response.json().catch(() => null)
      availableRoles.value = Array.isArray(body) ? body : []
    }
  } catch (error) {
    console.error('获取可复用角色失败:', error)
  } finally {
    rolesLoading.value = false
  }
}

const isLoading = ref(false)
// OPT-20260819-038: 邀请用户/生成邀请链接均为写操作，防连点双发
const inviteUserGuard = createClickGuard()
const generateInviteLinkGuard = createClickGuard()
const copySuccess = ref(false)
const inviteToken = ref('')
const linkKind = ref('single')
const maxUses = ref('')
const skipInvite = ref(null)
const workspaces = ref([])
const workspacesLoading = ref(false)

// 计算邀请链接
const inviteLink = computed(() => {
  if (!inviteToken.value) {
    return '请先生成邀请链接'
  }
  return `${window.location.origin}/tenant/${tenantId}/people/join/?token=${inviteToken.value}`
})

// 获取工作空间列表
const fetchWorkspaces = async () => {
  workspacesLoading.value = true
  try {
    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}`, {
      method: 'GET',
      credentials: 'include',
      headers: {
        'Accept': 'application/json',
      }
    })
    
    if (response.ok) {
      const { data } = await safeResponseJson(response, { fallback: [] })
      workspaces.value = data

      // 默认选择第一个工作空间
      if (data && data.length > 0 && !formData.value.workspace_id) {
        formData.value.workspace_id = data[0].id
      }
    }
  } catch (error) {
    console.error('获取工作空间失败:', error)
  } finally {
    workspacesLoading.value = false
  }
}

// 组件加载时获取工作空间列表
// OPT-20260809-025: 无权限成员跳过会 403 的数据拉取，避免原始错误态
onMounted(async () => {
  await perms.load(apiFetch)
  accessChecked.value = true
  if (canManage.value) {
    fetchWorkspaces()
    fetchRoles()
  }
})

const inviteUser = async () => {
  // OPT-20260819-038: 邀请用户是写操作，防连点/超时重试双发 POST
  await inviteUserGuard.run(async ({ idempotencyKey }) => {
    isLoading.value = true

    try {
      // 调用后端API邀请用户
      const response = await apiFetch(`/api/tenant/${tenantId}/accounts/members/invite/`, {
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
          ...formData.value,
          invite_method: inviteMethod.value,
          ...(formData.value.role !== 'admin' && pendingGrants.value.length
            ? { grants: pendingGrants.value }
            : {}),
          ...(formData.value.role !== 'admin' && selectedRoleNames.value.length
            ? { role_names: selectedRoleNames.value }
            : {}),
        })
      })
    
      if (!response.ok) {
        const { data: errorData, traceId } = await safeResponseJson(response, { fallback: {} })
        const err = new Error(errorData?.error || '邀请失败')
        if (traceId) err.traceId = traceId
        throw err
      }

      const { data: result } = await safeResponseJson(response, { fallback: {} })

      if (result?.email_skipped) {
        skipInvite.value = {
          hint: result.message || '该邮箱已退订邮件邀请，请手动复制邀请链接给对方',
          url: result.invitation_url || '',
        }
        if (result.invite_token) {
          inviteToken.value = result.invite_token
        }
        return
      }

      skipInvite.value = null
      toastService.success('邀请已成功发送！', 2000)

      formData.value.email = ''
      formData.value.phone = ''
      formData.value.company_member_name = ''
      formData.value.role = 'member'
      formData.value.message = ''
      formData.value.expiration_days = DEFAULT_INVITE_EXPIRATION_DAYS
      selectedRoleNames.value = []
    } catch (error) {
      console.error('邀请失败:', error)
      toastService.error(error.message, error.traceId ? { traceId: error.traceId } : undefined)
    } finally {
      isLoading.value = false
    }
  })
}

const generateInviteLink = async () => {
  if (!formData.value.workspace_id) {
    toastService.warning('请先选择工作空间')
    return
  }
  if (memberNameRequiredForLink(linkKind.value) && !formData.value.company_member_name) {
    toastService.warning('请填写成员名称')
    return
  }
  await generateInviteLinkGuard.run(async ({ idempotencyKey }) => {
    isLoading.value = true
    try {
      const response = await apiFetch(`/api/tenant/${tenantId}/accounts/members/invite/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(buildLinkInviteBody({
          formData: formData.value,
          pendingGrants: pendingGrants.value,
          selectedRoleNames: selectedRoleNames.value,
          linkKind: linkKind.value,
          maxUses: maxUses.value,
        })),
      })

      if (!response.ok) {
        const { data: errorData, traceId } = await safeResponseJson(response, { fallback: {} })
        const err = new Error(errorData?.error || '生成邀请链接失败')
        if (traceId) err.traceId = traceId
        throw err
      }

      const { data: result } = await safeResponseJson(response, { fallback: {} })
      inviteToken.value = result?.invite_token || ''
    } catch (error) {
      console.error('生成邀请链接失败:', error)
      toastService.error(error.message, error.traceId ? { traceId: error.traceId } : undefined)
    } finally {
      isLoading.value = false
    }
  })
}

// 复制邀请链接
const copyInviteLink = async () => {
  if (!inviteToken.value) {
    toastService.warning('请先生成邀请链接')
    return
  }
  
  try {
    await navigator.clipboard.writeText(inviteLink.value)
    toastService.success('邀请链接已复制到剪贴板！', 2000)
  } catch (error) {
    console.error('复制失败:', error)
    toastService.error('复制失败，请手动复制链接')
  }
}
</script>

<style scoped>
.people-invite-container {
  min-height: 80vh;
}
</style>