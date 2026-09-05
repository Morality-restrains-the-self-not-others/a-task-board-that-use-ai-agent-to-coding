<template>
  <div data-alias="view-system-admin-privacy-policy" id="privacy-policy" class="p-6">
    <div class="mb-6">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold text-gray-900">隐私条款管理</h2>
        <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="createModalVisible = true">
          创建条款
        </button>
      </div>
      <p class="text-sm text-gray-500 mt-2">
        若新版本勾选「含实质性变更」，老用户登录后需再次确认同意；同意记录会写入数据库供合规审计（亦可在下方按用户 ID 查询）。
      </p>
    </div>

    <SystemAdminPrivacyPolicyConsentQuery />

    <div v-if="createModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">创建隐私条款</h3>
          <button @click="createModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleCreate">
          <div class="space-y-4">
            <div>
              <label for="create-title" class="block text-sm font-medium text-gray-700 mb-2">条款标题</label>
              <input type="text" id="create-title" v-model="createForm.title"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="输入条款标题" required>
            </div>

            <div>
              <label for="create-version" class="block text-sm font-medium text-gray-700 mb-2">版本号</label>
              <input type="text" id="create-version" v-model="createForm.version"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="例如：1.0.0" required>
            </div>

            <div>
              <label for="create-content" class="block text-sm font-medium text-gray-700 mb-2">条款内容</label>
              <textarea id="create-content" v-model="createForm.content"
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                        placeholder="输入隐私条款内容" rows="10"></textarea>
            </div>

            <div class="flex items-center">
              <input type="checkbox" id="create-is-active" v-model="createForm.is_active"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <label for="create-is-active" class="ml-2 block text-sm text-gray-700">立即生效</label>
            </div>
            <div class="flex items-start gap-2">
              <input
                id="create-material"
                v-model="createForm.is_material_change"
                type="checkbox"
                class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded mt-0.5"
              >
              <label for="create-material" class="text-sm text-gray-700">
                含实质性变更（老用户须登录后再次确认）
              </label>
            </div>
          </div>

          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="createModalVisible = false"
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="creating"
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ creating ? '创建中...' : '创建条款' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="editModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">编辑隐私条款</h3>
          <button @click="editModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleEdit">
          <div class="space-y-4">
            <div>
              <label for="edit-title" class="block text-sm font-medium text-gray-700 mb-2">条款标题</label>
              <input type="text" id="edit-title" v-model="editForm.title"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="输入条款标题" required>
            </div>

            <div>
              <label for="edit-version" class="block text-sm font-medium text-gray-700 mb-2">版本号</label>
              <input type="text" id="edit-version" v-model="editForm.version"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="例如：1.0.0" required>
            </div>

            <div>
              <label for="edit-content" class="block text-sm font-medium text-gray-700 mb-2">条款内容</label>
              <textarea id="edit-content" v-model="editForm.content"
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                        placeholder="输入隐私条款内容" rows="10"></textarea>
            </div>

            <div class="flex items-center">
              <input type="checkbox" id="edit-is-active" v-model="editForm.is_active"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <label for="edit-is-active" class="ml-2 block text-sm text-gray-700">生效</label>
            </div>
            <div class="flex items-start gap-2">
              <input type="checkbox" id="edit-is-material" v-model="editForm.is_material_change"
                     class="mt-0.5 w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <div>
                <label for="edit-is-material" class="block text-sm text-gray-700">相对上一版为实质性变更</label>
                <p class="text-xs text-gray-500 mt-0.5">面向已注册用户的再次确认需求，详见创建条款说明。</p>
              </div>
            </div>
          </div>

          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="editModalVisible = false"
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="editing"
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ editing ? '保存中...' : '保存修改' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="deleteModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">删除隐私条款</h3>
          <button @click="deleteModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <p class="text-gray-600 mb-6">确定要删除条款"{{ deleteTarget.title }}"吗？此操作不可恢复。</p>

        <div class="flex justify-end space-x-3">
          <button type="button" @click="deleteModalVisible = false"
                  class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
            取消
          </button>
          <button type="button" :disabled="deleting" @click="handleDelete"
                  class="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-all duration-300">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
      <span class="ml-3 text-gray-600">加载中...</span>
    </div>

    <div v-else>
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">版本</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">实质性变更</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-if="policies.length === 0">
              <td colspan="7" class="px-6 py-12 text-center">
                <div class="flex flex-col items-center justify-center">
                  <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                  </svg>
                  <h3 class="text-lg font-medium text-gray-900 mb-1">暂无隐私条款</h3>
                  <p class="text-gray-500 mb-6">请点击"创建条款"按钮创建第一个隐私条款。</p>
                  <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="createModalVisible = true">
                    创建条款
                  </button>
                </div>
              </td>
            </tr>
            <tr v-for="policy in policies" :key="policy.id">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ policy.id }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ policy.title }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">v{{ policy.version }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span :class="policy.is_active ? 'text-green-600' : 'text-gray-400'">
                  {{ policy.is_active ? '生效中' : '未生效' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span :class="policy.is_material_change ? 'text-amber-700' : 'text-gray-500'">
                  {{ policy.is_material_change ? '是' : '否' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(policy.created_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button class="text-primary hover:text-primary/90 mr-3" @click="openEditModal(policy)">
                  编辑
                </button>
                <button class="text-red-600 hover:text-red-800" @click="openDeleteModal(policy)">
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import SystemAdminPrivacyPolicyConsentQuery from './SystemAdminPrivacyPolicyConsentQuery.vue'

const createModalVisible = ref(false)
const editModalVisible = ref(false)
const deleteModalVisible = ref(false)

const loading = ref(false)
const creating = ref(false)
const editing = ref(false)
const deleting = ref(false)

const createForm = ref({
  title: '',
  version: '',
  content: '',
  is_active: true,
  is_material_change: false
})

const editForm = ref({
  id: '',
  title: '',
  version: '',
  content: '',
  is_active: true,
  is_material_change: false
})

const deleteTarget = ref({
  id: '',
  title: ''
})

const policies = ref([])

// OPT-20260819-038: 隐私条款创建/编辑/删除均为写操作，防连点/超时重试双发
const createPolicyGuard = createClickGuard()
const editPolicyGuard = createClickGuard()
const deletePolicyGuard = createClickGuard()

const formatDate = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const refreshPolicies = async () => {
  loading.value = true
  try {
    const response = await apiFetch('/api/system-admin/privacy-policy/', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })

    if (response.ok) {
      policies.value = await response.json()
    } else {
      const errorData = await response.json().catch(() => ({}))
      const err = new Error(errorData.detail || '获取隐私条款列表失败')
      err.traceId = errorData._traceId || response.traceId
      throw err
    }
  } catch (error) {
    console.error('获取隐私条款列表失败:', error)
    showRequestError('获取隐私条款列表失败', error)
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  // OPT-20260819-038: 创建隐私条款是写操作，防连点双发 POST
  await createPolicyGuard.run(async ({ idempotencyKey }) => {
    creating.value = true
    try {
      const response = await apiFetch('/api/system-admin/privacy-policy/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify(createForm.value)
      })

      if (response.ok) {
        createModalVisible.value = false
        createForm.value = {
          title: '',
          version: '',
          content: '',
          is_active: true,
          is_material_change: false
        }
        await refreshPolicies()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.detail || '创建隐私条款失败')
        err.traceId = errorData._traceId || response.traceId
        throw err
      }
    } catch (error) {
      console.error('创建隐私条款失败:', error)
      showRequestError('创建隐私条款失败: ' + error.message, error)
    } finally {
      creating.value = false
    }
  })
}

const openEditModal = (policy) => {
  editForm.value = {
    id: policy.id,
    title: policy.title,
    version: policy.version,
    content: policy.content,
    is_active: policy.is_active,
    is_material_change: Boolean(policy.is_material_change)
  }
  editModalVisible.value = true
}

const handleEdit = async () => {
  // OPT-20260819-038: 编辑隐私条款是写操作，防连点双发 PUT
  await editPolicyGuard.run(async ({ idempotencyKey }) => {
    editing.value = true
    try {
      const { id, title, version, content, is_active, is_material_change } = editForm.value
      const response = await apiFetch(`/api/system-admin/privacy-policy/${id}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify({ title, version, content, is_active, is_material_change })
      })

      if (response.ok) {
        editModalVisible.value = false
        await refreshPolicies()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.detail || '更新隐私条款失败')
        err.traceId = errorData._traceId || response.traceId
        throw err
      }
    } catch (error) {
      console.error('更新隐私条款失败:', error)
      showRequestError('更新隐私条款失败: ' + error.message, error)
    } finally {
      editing.value = false
    }
  })
}

const openDeleteModal = (policy) => {
  deleteTarget.value = {
    id: policy.id,
    title: policy.title
  }
  deleteModalVisible.value = true
}

const handleDelete = async () => {
  // OPT-20260819-038: 删除隐私条款是写操作，防连点双发 DELETE
  await deletePolicyGuard.run(async ({ idempotencyKey }) => {
    deleting.value = true
    try {
      const response = await apiFetch(`/api/system-admin/privacy-policy/${deleteTarget.value.id}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders(
          {
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include'
      })

      if (response.ok || response.status === 204) {
        deleteModalVisible.value = false
        await refreshPolicies()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.detail || '删除隐私条款失败')
        err.traceId = errorData._traceId || response.traceId
        throw err
      }
    } catch (error) {
      console.error('删除隐私条款失败:', error)
      showRequestError('删除隐私条款失败: ' + error.message, error)
    } finally {
      deleting.value = false
    }
  })
}

onMounted(async () => {
  await refreshPolicies()
})
</script>
<style scoped>
</style>
