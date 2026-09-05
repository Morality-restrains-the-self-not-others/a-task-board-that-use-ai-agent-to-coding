<template>
  <!-- OPT-20260809-025: 非管理员友好提示 — 先探权限，无权时展示「无权限」空态而非原始 403 -->
  <div v-if="!accessChecked" class="people-groups-container p-6 text-text-light">加载中…</div>
  <div v-else-if="!canManage" class="people-groups-container p-6">
    <div class="bg-white rounded-xl shadow-md p-6 text-center py-16">
      <p class="text-xl font-bold">无权限访问</p>
      <p class="text-text-light mt-3">管理分组仅对拥有分组管理权限的公司成员开放，如需访问请联系公司管理员。</p>
    </div>
  </div>
  <div v-else class="people-groups-container p-6">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold">管理分组</h2>
      <div v-if="companies.length >= 1" class="relative">
        <select
          v-model="selectedCompanyId"
          @change="switchCompany"
          class="px-4 py-2 border border-border rounded-lg bg-white focus:ring-2 focus:ring-primary focus:border-primary transition-colors text-sm"
        >
          <option
            v-for="c in companies"
            :key="c.company_id"
            :value="c.company_id"
          >
            🏢 {{ c.company_name }} {{ c.is_admin || c.is_creator ? '(管理员)' : '(成员)' }}
          </option>
        </select>
      </div>
    </div>
    
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- 左侧：分组列表 -->
      <div class="lg:col-span-1">
        <div class="bg-white rounded-xl shadow-md p-6 h-full">
          <div class="flex justify-between items-center mb-4">
            <h3 class="text-lg font-semibold">分组列表</h3>
            <button @click="showCreateGroupModal = true" class="bg-primary text-white px-3 py-1 rounded-lg hover:bg-primary/90 transition-colors text-sm">
              创建分组
            </button>
          </div>
          
          <div v-if="isLoadingGroups" class="text-center py-8">
            加载中...
          </div>
          
          <div v-else class="space-y-2">
            <div 
              v-for="group in groups" 
              :key="group.id"
              @click="selectGroup(group)"
              class="p-3 rounded-lg border cursor-pointer transition-colors"
              :class="selectedGroup?.id === group.id ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50'"
            >
              <div class="flex justify-between items-center">
                <span class="font-medium">{{ group.name }}</span>
                <span class="text-sm text-text-light">{{ group.memberCount }}人</span>
              </div>
              <div class="text-xs text-text-light mt-1">{{ group.description }}</div>
            </div>
          </div>
          
          <div v-if="!isLoadingGroups && groups.length === 0" class="text-center py-8 text-text-light">
            暂无分组
          </div>
        </div>
      </div>
      
      <!-- 右侧：分组详情 -->
      <div class="lg:col-span-2">
        <div class="bg-white rounded-xl shadow-md p-6">
          <div v-if="selectedGroup">
            <div class="flex justify-between items-center mb-4">
              <h3 class="text-lg font-semibold">{{ selectedGroup.name }}</h3>
              <div>
                <button @click="showEditGroupModal = true" class="text-primary hover:text-primary/80 mr-3">编辑</button>
                <button @click="deleteGroup(selectedGroup.id)" class="text-danger hover:text-danger/80">删除</button>
              </div>
            </div>
            
            <div class="mb-4">
              <label class="block text-sm font-medium text-text mb-1">分组描述</label>
              <p class="text-text-light">{{ selectedGroup.description }}</p>
            </div>
            
            <div class="mb-4">
              <div class="flex justify-between items-center mb-2">
                <label class="block text-sm font-medium text-text">成员列表</label>
                <button @click="showAddMemberModal = true" class="text-primary hover:text-primary/80 text-sm">
                  添加成员
                </button>
              </div>
              
              <div v-if="isLoadingMembers" class="text-center py-4">
                加载中...
              </div>
              
              <div v-else class="space-y-2">
                <div 
                  v-for="member in selectedGroupMembers" 
                  :key="member.id"
                  class="flex justify-between items-center p-2 border border-border rounded-lg"
                >
                  <div>
                    <div class="font-medium">{{ member.username }}</div>
                    <div class="text-xs text-text-light">{{ member.email }}</div>
                  </div>
                  <button @click="removeMember(member.id)" class="text-danger hover:text-danger/80 text-sm">移除</button>
                </div>
              </div>
              
              <div v-if="!isLoadingMembers && selectedGroupMembers.length === 0" class="text-center py-4 text-text-light">
                暂无成员
              </div>
            </div>
          </div>
          
          <div v-else class="text-center py-12 text-text-light">
            请选择一个分组查看详情
          </div>
        </div>
      </div>
    </div>
    
    <!-- 创建分组模态框 -->
    <div v-if="showCreateGroupModal" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold mb-4">创建分组</h3>
        <form @submit.prevent="createGroup">
          <div class="mb-4">
            <label for="groupName" class="block text-sm font-medium text-text mb-1">分组名称</label>
            <input 
              type="text" 
              id="groupName" 
              v-model="newGroup.name" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
          </div>
          <div class="mb-4">
            <label for="groupDescription" class="block text-sm font-medium text-text mb-1">分组描述</label>
            <textarea 
              id="groupDescription" 
              v-model="newGroup.description" 
              rows="3"
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
            ></textarea>
          </div>
          <div class="flex justify-end space-x-3">
            <button @click="showCreateGroupModal = false" type="button" class="px-4 py-2 border border-border rounded-lg hover:bg-gray-50 transition-colors">
              取消
            </button>
            <button type="submit" class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors" :disabled="isSubmitting">
              {{ isSubmitting ? '创建中...' : '创建' }}
            </button>
          </div>
        </form>
      </div>
    </div>
    
    <!-- 添加成员模态框 -->
    <div v-if="showAddMemberModal" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 max-w-md w-full">
        <h3 class="text-lg font-semibold mb-4">添加成员</h3>
        <form @submit.prevent="addMember">
          <div class="mb-4">
            <label for="userId" class="block text-sm font-medium text-text mb-1">选择成员</label>
            <select 
              id="userId" 
              v-model="selectedUserId" 
              class="w-full px-4 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-primary transition-colors"
              required
            >
              <option value="">请选择成员</option>
              <option v-for="member in companyMembers" :key="member.user_id" :value="member.user_id">
                {{ member.username }} ({{ member.email }})
              </option>
            </select>
          </div>
          <div class="flex justify-end space-x-3">
            <button @click="showAddMemberModal = false" type="button" class="px-4 py-2 border border-border rounded-lg hover:bg-gray-50 transition-colors">
              取消
            </button>
            <button type="submit" class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors" :disabled="isSubmitting">
              {{ isSubmitting ? '添加中...' : '添加' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils.js'
import { usePermissions } from '../composables/usePermissions.js'
import modalService from '../utils/modalService.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { errorFromFailedResponse } from '../utils/httpError.js'

const route = useRoute()
const router = useRouter()
const tenantId = computed(() => route.params.tenant)
const perms = usePermissions()

// OPT-20260809-025: 权限已探测完成才渲染内容；canManage 按 v63 RBAC 权限码判定
//（member:manage 任意组穿透 / group:manage 分组管理 / group-members:manage 组管理员，
//  与 shareLib/authz 及后端 group_handlers.go 一致；OPT-20260811-076 修正 colon 错码）
const accessChecked = ref(false)
const canManage = computed(() =>
  perms.hasPerm(tenantId.value, 'member:manage') ||
  perms.hasPerm(tenantId.value, 'group:manage') ||
  perms.hasPerm(tenantId.value, 'group-members:manage'),
)

const selectedCompanyId = ref(route.params.tenant)
const companies = ref([])

const switchCompany = () => {
  if (selectedCompanyId.value && selectedCompanyId.value !== tenantId.value) {
    router.push(`/tenant/${selectedCompanyId.value}/people/groups/`)
  }
}

watch(() => route.params.tenant, (newTenant) => {
  selectedCompanyId.value = newTenant
})

const fetchCompanies = async () => {
  try {
    const userId = getCookie('userId')
    if (!userId) return
    await perms.load(apiFetch)
    const resp = await apiFetch(`/api/accounts/users/me/`, {
      credentials: 'include',
      headers: { 'Accept': 'application/json' }
    })
    if (!resp.ok) return
    const data = await resp.json()
    const rawCompanies = data.companies || []
    if (rawCompanies.length > 0) {
      companies.value = rawCompanies.map(c => ({
        company_id: c.id,
        company_name: c.name,
        // v63 RBAC: 管理员标签改用权限码判定
        is_admin: perms.hasPerm(c.id, 'member:manage'),
        is_creator: data.current_company?.id === c.id && !!c.is_admin,
      }))
      const current = companies.value.find(c => c.company_id === tenantId.value)
      if (!current) {
        companies.value.push({
          company_id: tenantId.value,
          company_name: '当前公司',
          is_admin: false,
          is_creator: false
        })
      }
    }
  } catch (e) {
    console.error('获取公司列表失败:', e)
  }
}

const groups = ref([])
const selectedGroup = ref(null)
const selectedGroupMembers = ref([])
const companyMembers = ref([])
const isLoadingGroups = ref(false)
const isLoadingMembers = ref(false)
const isSubmitting = ref(false)
// OPT-20260819-038: 创建分组/加人/移除成员/删除分组均为写操作，防连点双发
const createGroupGuard = createClickGuard()
const addMemberGuard = createClickGuard()
const removeMemberGuard = createClickGuard()
const deleteGroupGuard = createClickGuard()
const showCreateGroupModal = ref(false)
const showEditGroupModal = ref(false)
const showAddMemberModal = ref(false)
const selectedUserId = ref('')

const newGroup = ref({
  name: '',
  description: ''
})

const fetchGroups = async () => {
  isLoadingGroups.value = true
  
  try {
    const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw errorFromFailedResponse(response, '获取分组列表失败')
    }
    
    const data = await response.json()
    groups.value = data
  } catch (error) {
    console.error('获取分组列表失败:', error)
    showRequestError(error.message || '获取分组列表失败', error)
  } finally {
    isLoadingGroups.value = false
  }
}

const fetchCompanyMembers = async () => {
  try {
    const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/members/company_members/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw errorFromFailedResponse(response, '获取公司成员列表失败')
    }
    
    const data = await response.json()
    const { members } = parseCompanyMembersResponse(data)
    companyMembers.value = members
  } catch (error) {
    console.error('获取公司成员列表失败:', error)
    showRequestError(error.message || '获取公司成员列表失败', error)
  }
}

const fetchGroupMembers = async (groupId) => {
  isLoadingMembers.value = true
  
  try {
    const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/${groupId}/members/`, {
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    })
    
    if (!response.ok) {
      throw errorFromFailedResponse(response, '获取分组成员列表失败')
    }
    
    const data = await response.json()
    selectedGroupMembers.value = data
  } catch (error) {
    console.error('获取分组成员列表失败:', error)
    showRequestError(error.message || '获取分组成员列表失败', error)
  } finally {
    isLoadingMembers.value = false
  }
}

const selectGroup = (group) => {
  selectedGroup.value = group
  fetchGroupMembers(group.id)
}

const createGroup = async () => {
  // OPT-20260819-038: 创建分组是写操作，防连点/超时重试双发 POST
  await createGroupGuard.run(async ({ idempotencyKey }) => {
    isSubmitting.value = true

    try {
      const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(newGroup.value)
      })

      if (!response.ok) {
        throw errorFromFailedResponse(response, '创建分组失败')
      }

      // 刷新分组列表
      await fetchGroups()

      // 重置表单并关闭模态框
      newGroup.value = {
        name: '',
        description: ''
      }
      showCreateGroupModal.value = false
    } catch (error) {
      console.error('创建分组失败:', error)
      showRequestError(error.message || '创建分组失败', error)
    } finally {
      isSubmitting.value = false
    }
  })
}

const addMember = async () => {
  if (!selectedGroup.value || !selectedUserId.value) return

  // OPT-20260819-038: 添加成员是写操作，防连点/超时重试双发 POST
  await addMemberGuard.run(async ({ idempotencyKey }) => {
    isSubmitting.value = true

    try {
      const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/${selectedGroup.value.id}/add_member/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({ user_id: selectedUserId.value })
      })

      if (!response.ok) {
        throw errorFromFailedResponse(response, '添加成员失败')
      }

      // 刷新分组成员列表
      await fetchGroupMembers(selectedGroup.value.id)

      // 重置表单并关闭模态框
      selectedUserId.value = ''
      showAddMemberModal.value = false
    } catch (error) {
      console.error('添加成员失败:', error)
      showRequestError(error.message || '添加成员失败', error)
    } finally {
      isSubmitting.value = false
    }
  })
}

const removeMember = async (memberId) => {
  if (!selectedGroup.value) return

  // OPT-20260819-038: 移除成员是写操作，防连点/超时重试双发 DELETE
  await removeMemberGuard.run(async ({ idempotencyKey }) => {
    try {
      const member = selectedGroupMembers.value.find(m => m.id === memberId)
      if (!member) return

      const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/${selectedGroup.value.id}/remove_member/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        // CompanyGroupMemberSerializer.user 为 user_id 字符串；兼容旧形态 { id }
        body: JSON.stringify({
          user_id: member.user?.id ?? member.user ?? member.user_id,
        })
      })

      if (!response.ok) {
        throw errorFromFailedResponse(response, '移除成员失败')
      }

      // 刷新分组成员列表
      await fetchGroupMembers(selectedGroup.value.id)
    } catch (error) {
      console.error('移除成员失败:', error)
      showRequestError(error.message || '移除成员失败', error)
    }
  })
}

const deleteGroup = async (groupId) => {
  // 前端规范禁止浏览器原生 confirm()；统一走 modalService.confirm（OPT-20260811-076）
  try {
    await modalService.confirm('确定要删除该分组吗？')
  } catch {
    return // 用户取消删除
  }

  // OPT-20260819-038: 删除分组是写操作，防连点/超时重试双发 DELETE
  await deleteGroupGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/tenant/${tenantId.value}/accounts/groups/${groupId}/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Accept': 'application/json' }, idempotencyKey),
      })

      if (!response.ok) {
        throw errorFromFailedResponse(response, '删除分组失败')
      }

      // 刷新分组列表
      await fetchGroups()

      // 重置选中状态
      if (selectedGroup.value?.id === groupId) {
        selectedGroup.value = null
        selectedGroupMembers.value = []
      }
    } catch (error) {
      console.error('删除分组失败:', error)
      showRequestError(error.message || '删除分组失败', error)
    }
  })
}

// 组件挂载时获取数据
// OPT-20260809-025: 无权限成员跳过会 403 的数据拉取，避免原始错误态
onMounted(async () => {
  await fetchCompanies()
  accessChecked.value = true
  if (!canManage.value) return
  await fetchGroups()
  await fetchCompanyMembers()
})
</script>

<style scoped>
.people-groups-container {
  min-height: 80vh;
}
</style>