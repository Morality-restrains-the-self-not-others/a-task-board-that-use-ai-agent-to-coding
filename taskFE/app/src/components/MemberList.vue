<template>
  <div class="bg-white rounded-xl shadow-md p-6 mb-6">
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-lg font-semibold">成员列表</h3>
      <div class="relative">
        <input 
          type="text" 
          v-model="searchQuery" 
          placeholder="搜索成员..."
          class="pl-10 pr-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
        >
        <svg class="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-text-light" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
        </svg>
      </div>
    </div>
    
    <div v-if="budgetSectionVisible" class="mb-4 rounded-lg border border-border bg-primary/5 px-4 py-3 text-sm text-text-light">
      <template v-if="llmBudgetEnabled && canManageBudgetPermissions">
        勾选后，该成员可在任务详情对<strong>已超额</strong>的模型执行「临时上调」预算（须二次确认）。创建者无需配置。
      </template>
      <template v-else-if="llmBudgetEnabled">
        当前账号无权限修改成员预算上调授权，请联系租户管理员。
      </template>
      <template v-else>
        当前租户尚未启用 LLM 预算：请先在
        <router-link :to="featureParamsLink" class="text-primary hover:underline">智能体资源配置</router-link>
        页为至少一个 API 端点勾选「启用 LLM 预算」后，再配置成员权限。
      </template>
    </div>

    <div v-if="isLoading" class="text-center py-8">
      加载中...
    </div>

    <div v-else-if="!hasPermission" class="text-center py-12">
      <p class="text-text-light text-lg">🔒 {{ permissionMessage }}</p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b border-border">
            <th class="text-left py-3 px-4 font-medium text-text-light">公司成员名称</th>
            <th class="text-left py-3 px-4 font-medium text-text-light w-24">成员头像</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">登陆方式</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">角色</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">加入时间</th>
            <th class="text-left py-3 px-4 font-medium text-text-light">状态</th>
            <th v-if="llmBudgetEnabled" class="text-left py-3 px-4 font-medium text-text-light">
              允许临时上调单任务 LLM 预算
            </th>
            <th class="text-right py-3 px-4 font-medium text-text-light">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="member in filteredMembers" :key="member.id" class="border-b border-border hover:bg-primary/5">
            <td class="py-3 px-4">{{ member.member_name }}</td>
            <td class="py-3 px-4 align-middle">
              <img
                :src="memberAvatarDisplaySrc(member)"
                alt=""
                class="w-10 h-10 rounded-full object-cover border border-border"
                width="40"
                height="40"
              >
            </td>
            <td class="py-3 px-4">{{ member.email }}</td>
            <td class="py-3 px-4">
                <span v-if="member.role === '创建者'" class="bg-primary/10 text-primary px-2 py-1 rounded text-sm">创建者</span>
                <span v-else-if="member.role === 'admin'" class="bg-primary/10 text-primary px-2 py-1 rounded text-sm">管理员</span>
                <span v-else class="bg-gray-100 text-gray-600 px-2 py-1 rounded text-sm">成员</span>
              </td>
            <td class="py-3 px-4">{{ member.joinDate }}</td>
            <td class="py-3 px-4">
              <span v-if="member.status === 'active'" class="text-success">活跃</span>
              <span v-else-if="member.status === 'pending'" class="text-warning">待激活</span>
              <span v-else class="text-danger">已禁用</span>
            </td>
            <td v-if="llmBudgetEnabled" class="py-3 px-4">
              <span v-if="member.is_creator" class="text-xs text-gray-400" title="创建者默认拥有全部预算管理权限">内置</span>
              <input
                v-else-if="canManageBudgetPermissions"
                type="checkbox"
                :checked="Boolean(budgetRaiseByMemberId[String(member.id)])"
                :title="'允许该成员在任务超额后临时上调预算'"
                @change="toggleBudgetRaise(member, $event.target.checked)"
              />
              <span v-else class="text-xs text-gray-400" title="仅租户管理员可修改">—</span>
            </td>
            <td class="py-3 px-4 text-right">
              <button @click="openGitIdentities(member)" class="text-primary hover:text-primary/80 mr-3">Git 提交身份</button>
              <button @click="editMember(member)" class="text-primary hover:text-primary/80 mr-3">编辑</button>
              <button 
                v-if="member.status === 'active' && member.role !== '创建者'" 
                @click="toggleMemberStatus(member, false)"
                class="text-danger hover:text-danger/80 mr-3"
              >
                禁用
              </button>
              <button 
                v-else-if="member.status !== 'active' && member.role !== '创建者'" 
                @click="toggleMemberStatus(member, true)"
                class="text-success hover:text-success/80 mr-3"
              >
                启用
              </button>
              <button 
                v-if="member.role !== '创建者'" 
                @click="removeMember(member)" 
                class="text-danger hover:text-danger/80"
              >
                移除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    
    <div v-if="!isLoading && filteredMembers.length === 0" class="text-center py-8 text-text-light">
      暂无成员
    </div>
    
    <!-- 编辑成员模态框 -->
    <div v-if="showEditModal" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold mb-4">编辑成员</h3>
        <form @submit.prevent="saveMemberEdit">
          <div class="mb-4">
            <label class="block text-sm font-medium text-text mb-1">公司成员名称</label>
            <input 
              type="text" 
              v-model="editingMember.member_name" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
            >
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-text mb-1">角色</label>
            <select 
              v-model="editingMember.role" 
              :disabled="editingMember.role === '创建者'"
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
            >
              <option value="member">成员</option>
              <option value="admin">管理员</option>
            </select>
            <p v-if="editingMember.role === '创建者'" class="text-text-light text-sm mt-1">创建者角色不可修改</p>
          </div>
          <div class="flex justify-end space-x-3">
            <button @click="showEditModal = false" type="button" class="px-4 py-2 border border-border rounded-lg hover:bg-gray-50 transition-colors">
              取消
            </button>
            <button type="submit" class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors">
              保存
            </button>
          </div>
        </form>
      </div>
    </div>

    <MemberGitIdentitiesModal
      v-if="gitIdentityMember"
      :tenant-id="tenantId"
      :member-id="String(gitIdentityMember.id)"
      :member-name="gitIdentityMember.member_name || ''"
      @close="gitIdentityMember = null"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'
import { showRequestError, humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import MemberGitIdentitiesModal from './MemberGitIdentitiesModal.vue'

const memberAvatarDisplaySrc = (member) => {
  const url = member.member_avatar_url
  if (url) {
    return url
  }
  const seed = member.member_name || member.email || 'member'
  return initialsAvatarDataUri(seed)
}

const props = defineProps({
  tenantId: {
    type: String,
    required: true
  }
})

const searchQuery = ref('')
const members = ref([])
const isLoading = ref(false)
const hasPermission = ref(true)
const permissionMessage = ref('')
const showEditModal = ref(false)
const editingMember = ref({})
const gitIdentityMember = ref(null)
const llmBudgetEnabled = ref(false)
const canManageBudgetPermissions = ref(false)
const budgetRaiseByMemberId = ref({})
const budgetFeatureLoaded = ref(false)

// OPT-20260819-038: 成员写操作（预算权限/改角色/切状态/移除）防连点/超时重试双发
const toggleBudgetRaiseGuard = createClickGuard()
const saveMemberEditGuard = createClickGuard()
const toggleMemberStatusGuard = createClickGuard()
const removeMemberGuard = createClickGuard()
const budgetToggleError = ref('')

const featureParamsLink = computed(() => `/tenant/${props.tenantId}/settings/feature-params/`)
const budgetSectionVisible = computed(() => budgetFeatureLoaded.value)

const filteredMembers = computed(() => {
  if (!searchQuery.value) {
    return members.value
  }
  
  const query = searchQuery.value.toLowerCase()
  return members.value.filter((member) => {
    const name = (member.member_name || '').toLowerCase()
    const mail = (member.email || '').toLowerCase()
    return name.includes(query) || mail.includes(query)
  })
})

const fetchMembers = async () => {
  isLoading.value = true
  
  try {
    const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/company_members/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (!response.ok) {
      console.error('获取成员列表失败: HTTP', response.status)
      hasPermission.value = false
      permissionMessage.value = '获取成员列表失败，请稍后重试'
      return
    }

    const data = await response.json()
    const { members: rows, meta } = parseCompanyMembersResponse(data)

    // 检查权限标记（优先于任何错误处理）
    if (meta.has_permission === false) {
      hasPermission.value = false
      permissionMessage.value = meta.permission_message || '没有访问权限'
      members.value = []
      return
    }

    hasPermission.value = true
    permissionMessage.value = ''
    members.value = rows
    llmBudgetEnabled.value = Boolean(meta.llm_budget_enabled)
    canManageBudgetPermissions.value = Boolean(meta.can_manage_budget_permissions)
    budgetRaiseByMemberId.value = { ...(meta.budget_raise_by_member_id || {}) }
    for (const row of rows) {
      if (row.can_raise_task_budget) {
        budgetRaiseByMemberId.value[String(row.id)] = true
      }
    }
    budgetFeatureLoaded.value = true
  } catch (error) {
    console.error('获取成员列表失败:', error)
    hasPermission.value = false
    permissionMessage.value = '获取成员列表失败，请稍后重试'
  } finally {
    isLoading.value = false
  }
}

const toggleBudgetRaise = async (member, checked) => {
  // OPT-20260819-038: 预算权限切换是写操作，防连点双发 PATCH
  await toggleBudgetRaiseGuard.run(async ({ idempotencyKey }) => {
    budgetToggleError.value = ''
    try {
      const resp = await apiFetch(`/api/cloud/budget-permissions/tenant_id/${props.tenantId}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          items: [
            {
              subject_type: 'member',
              subject_id: String(member.id),
              can_raise_task_budget: checked,
            },
          ],
        }),
      })
      if (!resp.ok) {
        const errData = await resp.json().catch(() => ({}))
        const err = new Error(errData.message || '保存权限失败')
        err.traceId = resp.traceId || errData._traceId || ''
        throw err
      }
      budgetRaiseByMemberId.value = {
        ...budgetRaiseByMemberId.value,
        [String(member.id)]: checked,
      }
    } catch (error) {
      budgetToggleError.value = humanizeRequestErrorMessage(error.message || '保存权限失败')
      showRequestError(budgetToggleError.value, error)
    }
  })
}

const editMember = (member) => {
  editingMember.value = { ...member }
  showEditModal.value = true
}

const openGitIdentities = (member) => {
  gitIdentityMember.value = member
}

const saveMemberEdit = async () => {
  // OPT-20260819-038: 更新成员角色是写操作，防连点双发 PATCH
  await saveMemberEditGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/${editingMember.value.id}/update_role/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          role: editingMember.value.role,
          member_name: editingMember.value.member_name
        })
      })

      if (!response.ok) {
        const err = new Error('更新成员失败')
        err.traceId = response.traceId || ''
        throw err
      }
      // 刷新成员列表
      await fetchMembers()
      showEditModal.value = false
    } catch (error) {
      console.error('更新成员失败:', error)
      showRequestError(error.message || '更新成员失败', error)
    }
  })
}

const toggleMemberStatus = async (member, activate) => {
  // OPT-20260819-038: 切换成员状态是写操作，防连点双发 PATCH
  await toggleMemberStatusGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/${member.id}/toggle_status/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
      })

      if (!response.ok) {
        const err = new Error('切换状态失败')
        err.traceId = response.traceId || ''
        throw err
      }
      // 刷新成员列表
      await fetchMembers()
    } catch (error) {
      console.error('切换状态失败:', error)
      showRequestError(error.message || '切换状态失败', error)
    }
  })
}

const removeMember = async (member) => {
  if (!confirm('确定要移除该成员吗？')) {
    return
  }

  // OPT-20260819-038: 移除成员是写操作，防连点双发 DELETE
  await removeMemberGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${props.tenantId}/accounts/members/${member.id}/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
      })

      if (!response.ok) {
        const err = new Error('移除成员失败')
        err.traceId = response.traceId || ''
        throw err
      }
      // 刷新成员列表
      await fetchMembers()
    } catch (error) {
      console.error('移除成员失败:', error)
      showRequestError(error.message || '移除成员失败', error)
    }
  })
}

// 组件挂载时获取成员列表
onMounted(() => {
  fetchMembers()
})
</script>

<style scoped>
/* 组件样式 */
</style>