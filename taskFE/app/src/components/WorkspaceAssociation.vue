<template>
  <div>
    <!-- 关联工作空间列表 -->
    <div class="bg-white rounded-lg shadow-md p-6">
      <h2 class="text-xl font-semibold mb-4">关联工作空间</h2>
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">工作空间名称</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="workspace in workspaces" :key="workspace.id">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{ workspace.name }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button @click="removeWorkspace(workspace.id)" class="text-red-600 hover:text-red-900">移除关联</button>
              </td>
            </tr>
            <tr v-if="!workspaces || workspaces.length === 0">
              <td colspan="2" class="px-6 py-4 text-center text-sm text-gray-500">未关联任何工作空间</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 添加工作空间关联 -->
      <div class="mt-4">
        <h3 class="text-lg font-medium text-gray-900 mb-2">添加工作空间关联</h3>
        <div class="flex space-x-3">
          <select v-model="selectedWorkspaceId" class="border border-gray-300 rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent">
            <option value="">选择工作空间</option>
            <option v-for="ws in availableWorkspaces" :key="ws.id" :value="ws.id">
              {{ ws.name }}
            </option>
          </select>
          <button @click="addWorkspace" class="bg-green-500 hover:bg-green-600 text-white px-4 py-2 rounded-md transition-colors">
            添加关联
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import modalService from '../utils/modalService.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const props = defineProps({
  project: {
    type: Object,
    required: true
  },
  workspaces: {
    type: Array,
    default: () => []
  },
  availableWorkspaces: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  },
  tenantId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:project', 'update:workspaces', 'update:availableWorkspaces'])

const selectedWorkspaceId = ref('')

// OPT-20260819-038: 项目工作空间关联为资源写路径，createClickGuard 防连点双发 POST。
const addGuard = createClickGuard()
const removeGuard = createClickGuard()

async function refreshProjectAfterAssociation() {
  const projectResponse = await apiFetch(`/api/projects/tenant_id/${props.tenantId}/${props.project.id}/`, {
    method: 'GET',
    credentials: 'include'
  })
  if (!projectResponse.ok) return
  const projectData = await projectResponse.json()
  emit('update:project', projectData)
  emit('update:workspaces', projectData.workspaces)
}

function attachErrorTrace(err, response, errorData) {
  const tid = extractTraceId(response) || extractTraceId(errorData)
  if (tid) err.traceId = tid
  return err
}

const removeWorkspace = async (workspaceId) => {
  await removeGuard.run(async ({ idempotencyKey }) => {
    try {
      modalService.alert('加载中...', '提示', { autoClose: 3000 })

      const response = await apiFetch(`/api/projects/tenant_id/${props.tenantId}/${props.project.id}/associateWorkspace/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
        }, idempotencyKey),
        body: JSON.stringify({
          workspace_id: workspaceId,
          action: 'remove'
        }),
        credentials: 'include'
      })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw attachErrorTrace(new Error(errorData.error || '移除工作空间关联失败'), response, errorData)
    }
    await response.json().catch(() => ({}))
    await refreshProjectAfterAssociation()
    modalService.alert('移除工作空间关联成功', '成功', { autoClose: 2000 })
      } catch (err) {
        console.error('Error removing workspace association:', err)
        showRequestError(err.message || '移除工作空间关联失败', err, { autoClose: 3000 })
      }
  })
}

const addWorkspace = async () => {
  if (!selectedWorkspaceId.value) return

  await addGuard.run(async ({ idempotencyKey }) => {
    try {
      modalService.alert('加载中...', '提示', { autoClose: 3000 })

      const response = await apiFetch(`/api/projects/tenant_id/${props.tenantId}/${props.project.id}/associateWorkspace/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
        }, idempotencyKey),
        body: JSON.stringify({
          workspace_id: selectedWorkspaceId.value
        }),
        credentials: 'include'
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw attachErrorTrace(new Error(errorData.error || '添加工作空间关联失败'), response, errorData)
      }
      await response.json().catch(() => ({}))
      await refreshProjectAfterAssociation()
      selectedWorkspaceId.value = ''
      modalService.alert('添加工作空间关联成功', '成功', { autoClose: 2000 })
    } catch (err) {
      console.error('Error adding workspace association:', err)
      showRequestError(err.message || '添加工作空间关联失败', err, { autoClose: 3000 })
    }
  })
}
</script>

<style scoped>
/* 组件内样式可以根据需要添加 */
</style>
