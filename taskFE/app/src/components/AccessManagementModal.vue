<template>
  <div v-if="show" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
      <div class="flex justify-between items-center mb-6">
        <h4 class="text-lg font-bold">访问管理</h4>
        <button 
          class="text-gray-500 hover:text-gray-700"
          @click="$emit('close')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>
      
      <div class="space-y-6">
        <AccessMemberList
          :members="companyMembers"
          :access="memberAccess"
          :loading="loadingWorkspaceAccess"
          @add="openAddMemberModal"
          @update-access="updateMemberAccess"
          @remove="removeMemberAccess"
        />

        <AccessGroupList
          :groups="companyGroups"
          :access="groupAccess"
          :loading="loadingWorkspaceAccess"
          @add="openAddGroupModal"
          @update-access="updateGroupAccess"
          @remove="removeGroupAccess"
        />

        <!-- 底部按钮 -->
        <div class="flex justify-end space-x-3 pt-6">
          <button 
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
            @click="$emit('close')"
          >
            关闭
          </button>
        </div>
      </div>

      <AccessAddMemberModal
        :show="showAddMemberModal"
        :loading="loadingAddableMembers"
        :submitting="addMemberSubmitting"
        :members="membersWithoutAccess"
        :form="addMemberForm"
        :label-fn="getMemberIdentityLabel"
        @close="closeAddMemberModal"
        @submit="handleAddMember"
      />

      <AccessAddGroupModal
        :show="showAddGroupModal"
        :loading="loadingAddableGroups"
        :submitting="addGroupSubmitting"
        :groups="groupsWithoutAccess"
        :form="addGroupForm"
        @close="closeAddGroupModal"
        @submit="handleAddGroup"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import AccessAddMemberModal from './AccessAddMemberModal.vue'
import AccessAddGroupModal from './AccessAddGroupModal.vue'
import AccessMemberList from './AccessMemberList.vue'
import AccessGroupList from './AccessGroupList.vue'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  workspaceId: {
    type: String,
    required: true
  },
  tenantId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['close'])

// 访问管理相关数据：成员/小组列表仅从工作空间的关联列表（workspace-permissions）拉取
const companyMembers = ref([])
const companyGroups = ref([])
const loadingWorkspaceAccess = ref(false)
const memberAccess = ref({})
const groupAccess = ref({})

// 添加成员/小组子模态框：候选列表从公司范围拉取，过滤掉已在关联列表中的
const showAddMemberModal = ref(false)
const showAddGroupModal = ref(false)
const addMemberSubmitting = ref(false)
const addGroupSubmitting = ref(false)
// OPT-20260819-038: 访问管理各写操作，防连点/超时重试双发
const updateMemberAccessGuard = createClickGuard()
const updateGroupAccessGuard = createClickGuard()
const removeMemberAccessGuard = createClickGuard()
const removeGroupAccessGuard = createClickGuard()
const addMemberGuard = createClickGuard()
const addGroupGuard = createClickGuard()
const addMemberForm = ref({ userId: '', role: 'view' })
const addGroupForm = ref({ groupId: '', role: 'view' })
const addableMembers = ref([])  // 用于添加成员的可选列表（公司成员中尚未加入该空间的）
const addableGroups = ref([])   // 用于添加小组的可选列表（公司小组中尚未加入该空间的）
const loadingAddableMembers = ref(false)
const loadingAddableGroups = ref(false)

// 尚无访问权限的成员（用于添加下拉）：仅从 addableMembers 拉取
const membersWithoutAccess = computed(() => addableMembers.value)

// 尚无访问权限的小组（用于添加下拉）：仅从 addableGroups 拉取
const groupsWithoutAccess = computed(() => addableGroups.value)

const getMemberIdentityLabel = (member) => {
  if (member?.is_tenant) return '租户'

  const role = member?.role
  if (role === '创建者') return '创建者'
  if (role === 'admin' || role === '管理员') return '管理员'
  if (role === 'member' || role === '成员') return '成员'

  return role || '成员'
}

// 监听show属性变化，当显示时加载工作空间关联列表
watch(() => props.show, async (newValue) => {
  if (newValue && props.workspaceId) {
    await loadWorkspaceAccessList(props.workspaceId)
  } else if (!newValue) {
    companyMembers.value = []
    companyGroups.value = []
    memberAccess.value = {}
    groupAccess.value = {}
    showAddMemberModal.value = false
    showAddGroupModal.value = false
    addableMembers.value = []
    addableGroups.value = []
  }
})

// 加载工作空间关联列表：成员/小组权限仅从该关联列表拉取
const loadWorkspaceAccessList = async (workspaceId) => {
  loadingWorkspaceAccess.value = true
  companyMembers.value = []
  companyGroups.value = []
  memberAccess.value = {}
  groupAccess.value = {}
  try {
    const response = await apiFetch(`/api/projects/workspace-access/workspace-permissions/tenant_id/${props.tenantId}/?workspace_id=${workspaceId}`, {
      headers: { 'Accept': 'application/json' }
    })
    if (response.ok) {
      const data = await response.json()
      const members = []
      const groups = []
      const permMap = { member: {}, group: {} }
      ;(Array.isArray(data) ? data : []).forEach(perm => {
        if (perm.user_info) {
          const uid = String(perm.user_info.company_member_id || perm.user_info.id)
          members.push({
            company_member_id: uid,
            member_name: perm.user_info.member_name || '未设置',
            email: perm.user_info.email || '无邮箱',
            is_tenant: perm.user_info.is_tenant || false
          })
          permMap.member[uid] = perm.role
        } else if (perm.group_info) {
          const gid = String(perm.group_info.id)
          groups.push({
            id: gid,
            name: perm.group_info.name || ''
          })
          permMap.group[gid] = perm.role
        }
      })
      companyMembers.value = members
      companyGroups.value = groups
      memberAccess.value = permMap.member
      groupAccess.value = permMap.group
    }
  } catch (error) {
    console.error('Error loading workspace access list:', error)
  } finally {
    loadingWorkspaceAccess.value = false
  }
}

// 打开添加成员模态框时，拉取公司成员并过滤出尚未加入该空间的
// OPT-049: 移除 Django bare-path 回退（已下线），tenantId 为 required prop
const openAddMemberModal = async () => {
  showAddMemberModal.value = true
  addableMembers.value = []
  loadingAddableMembers.value = true
  try {
    const base = `/api/tenant/${props.tenantId}/accounts/members/company_members/`
    const response = await apiFetch(base, { headers: { 'Accept': 'application/json' } })
    if (response.ok) {
      const all = await response.json()
      const { members } = parseCompanyMembersResponse(all)
      const inWorkspace = new Set(companyMembers.value.map(m => String(m.company_member_id)))
      addableMembers.value = members.filter(m => !inWorkspace.has(String(m.company_member_id)))
    }
  } catch (e) {
    console.error('Error loading addable members:', e)
  } finally {
    loadingAddableMembers.value = false
  }
}

// 打开添加小组模态框时，拉取公司小组并过滤出尚未加入该空间的
// OPT-049: 移除 Django bare-path 回退（已下线），tenantId 为 required prop
const openAddGroupModal = async () => {
  showAddGroupModal.value = true
  addableGroups.value = []
  loadingAddableGroups.value = true
  try {
    const base = `/api/tenant/${props.tenantId}/accounts/groups/`
    const response = await apiFetch(base, { headers: { 'Accept': 'application/json' } })
    if (response.ok) {
      const all = await response.json()
      const inWorkspace = new Set(companyGroups.value.map(g => String(g.id)))
      addableGroups.value = (Array.isArray(all) ? all : []).filter(g => !inWorkspace.has(String(g.id)))
    }
  } catch (e) {
    console.error('Error loading addable groups:', e)
  } finally {
    loadingAddableGroups.value = false
  }
}

// 更新成员访问权限
const updateMemberAccess = async (memberId) => {
  const workspaceId = props.workspaceId
  const role = memberAccess.value[memberId]

  // OPT-20260819-038: 更新成员权限是写操作，防连点/超时重试双发
  await updateMemberAccessGuard.run(async ({ idempotencyKey }) => {
    try {
      const url = role
        ? `/api/projects/workspace-access/set-permission/tenant_id/${props.tenantId}/`
        : `/api/projects/workspace-access/remove-permission/tenant_id/${props.tenantId}/`

      const method = role ? 'POST' : 'DELETE'
      const body = {
        workspace_id: workspaceId,
        company_member_id: memberId
      }

      if (role) {
        body.role = role
      }

      const response = await apiFetch(url, {
        method,
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(body)
      })


      if (!response.ok) {
        const err = new Error('Failed to update member access')
        err.traceId = response.traceId || ''
        throw err
      }
      // 若移除权限，从关联列表中移除该成员
      if (!role) {
        await loadWorkspaceAccessList(workspaceId)
      }
    } catch (error) {
      console.error('Error updating member access:', error)
      showRequestError('更新成员权限失败，请重试', error)
    }
  })
}

// 更新小组访问权限
const updateGroupAccess = async (groupId) => {
  const workspaceId = props.workspaceId
  const role = groupAccess.value[groupId]

  // OPT-20260819-038: 更新小组权限是写操作，防连点/超时重试双发
  await updateGroupAccessGuard.run(async ({ idempotencyKey }) => {
    try {
      const url = role
        ? `/api/projects/workspace-access/set-permission/tenant_id/${props.tenantId}/`
        : `/api/projects/workspace-access/remove-permission/tenant_id/${props.tenantId}/`

      const method = role ? 'POST' : 'DELETE'
      const body = {
        workspace_id: workspaceId,
        group_id: groupId
      }

      if (role) {
        body.role = role
      }

      const response = await apiFetch(url, {
        method,
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(body)
      })


      if (!response.ok) {
        const err = new Error('Failed to update group access')
        err.traceId = response.traceId || ''
        throw err
      }
      // 若移除权限，从关联列表中移除该小组
      if (!role) {
        await loadWorkspaceAccessList(workspaceId)
      }
    } catch (error) {
      console.error('Error updating group access:', error)
      showRequestError('更新小组权限失败，请重试', error)
    }
  })
}

// 移除成员访问权限
const removeMemberAccess = async (memberId) => {
  if (confirm('确定要移除该成员的访问权限吗？')) {
    const workspaceId = props.workspaceId
    // OPT-20260819-038: 移除成员权限是写操作，防连点/超时重试双发 DELETE
    await removeMemberAccessGuard.run(async ({ idempotencyKey }) => {
      try {
        const response = await apiFetch(`/api/projects/workspace-access/remove-permission/tenant_id/${props.tenantId}/`, {
          method: 'DELETE',
          headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
          body: JSON.stringify({
            workspace_id: workspaceId,
            company_member_id: memberId
          })
        })

        if (response.ok) {
          await loadWorkspaceAccessList(workspaceId)
        } else {
          const err = new Error('移除成员权限失败')
          err.traceId = response.traceId || ''
          throw err
        }
      } catch (error) {
        console.error('Error removing member access:', error)
        showRequestError('移除成员权限失败，请重试', error)
      }
    })
  }
}

// 移除小组访问权限
const removeGroupAccess = async (groupId) => {
  if (confirm('确定要移除该小组的访问权限吗？')) {
    const workspaceId = props.workspaceId
    // OPT-20260819-038: 移除小组权限是写操作，防连点/超时重试双发 DELETE
    await removeGroupAccessGuard.run(async ({ idempotencyKey }) => {
      try {
        const response = await apiFetch(`/api/projects/workspace-access/remove-permission/tenant_id/${props.tenantId}/`, {
          method: 'DELETE',
          headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
          body: JSON.stringify({
            workspace_id: workspaceId,
            group_id: groupId
          })
        })

        if (response.ok) {
          await loadWorkspaceAccessList(workspaceId)
        } else {
          const err = new Error('移除小组权限失败')
          err.traceId = response.traceId || ''
          throw err
        }
      } catch (error) {
        console.error('Error removing group access:', error)
        showRequestError('移除小组权限失败，请重试', error)
      }
    })
  }
}

// 关闭添加成员模态框
const closeAddMemberModal = () => {
  showAddMemberModal.value = false
  addMemberForm.value = { userId: '', role: 'view' }
}

// 关闭添加小组模态框
const closeAddGroupModal = () => {
  showAddGroupModal.value = false
  addGroupForm.value = { groupId: '', role: 'view' }
}

// 添加成员
const handleAddMember = async () => {
  const { userId, role } = addMemberForm.value
  if (!userId || !role) return
  // OPT-20260819-038: 添加成员是写操作，防连点/超时重试双发 POST
  await addMemberGuard.run(async ({ idempotencyKey }) => {
    addMemberSubmitting.value = true
    try {
      const response = await apiFetch(`/api/projects/workspace-access/set-permission/tenant_id/${props.tenantId}/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          workspace_id: props.workspaceId,
          company_member_id: userId,
          role
        })
      })
      if (response.ok) {
        closeAddMemberModal()
        await loadWorkspaceAccessList(props.workspaceId)
      } else {
        const err = new Error('添加成员失败')
        err.traceId = response.traceId || ''
        throw err
      }
    } catch (error) {
      console.error('Error adding member:', error)
      showRequestError('添加成员失败，请重试', error)
    } finally {
      addMemberSubmitting.value = false
    }
  })
}

// 添加小组
const handleAddGroup = async () => {
  const { groupId, role } = addGroupForm.value
  if (!groupId || !role) return
  // OPT-20260819-038: 添加小组是写操作，防连点/超时重试双发 POST
  await addGroupGuard.run(async ({ idempotencyKey }) => {
    addGroupSubmitting.value = true
    try {
      const response = await apiFetch(`/api/projects/workspace-access/set-permission/tenant_id/${props.tenantId}/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          workspace_id: props.workspaceId,
          group_id: groupId,
          role
        })
      })
      if (response.ok) {
        closeAddGroupModal()
        await loadWorkspaceAccessList(props.workspaceId)
      } else {
        const err = new Error('添加小组失败')
        err.traceId = response.traceId || ''
        throw err
      }
    } catch (error) {
      console.error('Error adding group:', error)
      showRequestError('添加小组失败，请重试', error)
    } finally {
      addGroupSubmitting.value = false
    }
  })
}
</script>

<style scoped>
/* 组件内样式 */
</style>