<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-900">派生Token供应商列表</h2>
        <p class="text-sm text-gray-500 mt-1">配置支持主Key派生子Key的AI模型供应商（设置baseUrl和派生方案）</p>
      </div>
      <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90" @click="startCreate">
        新增供应商
      </button>
    </div>

    <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">供应商名称</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">baseUrl</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">派生端点</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">说明</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          <tr v-for="item in providers" :key="item.id">
            <td class="px-4 py-3 text-sm text-gray-900">{{ item.provider_name }}</td>
            <td class="px-4 py-3 text-sm text-gray-700">{{ item.base_url }}</td>
            <td class="px-4 py-3 text-sm text-gray-700">{{ item.derive_endpoint }}</td>
            <td class="px-4 py-3 text-sm text-gray-700">{{ item.description || '-' }}</td>
            <td class="px-4 py-3 text-sm">
              <button class="text-primary hover:underline mr-3" @click="startEdit(item)">编辑</button>
              <button class="text-red-600 hover:underline" @click="removeItem(item.id)">删除</button>
            </td>
          </tr>
          <tr v-if="providers.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-sm text-gray-500">暂无派生Token供应商</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showForm" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg w-full max-w-2xl p-6 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-semibold mb-4">{{ editingId ? '编辑供应商' : '新增供应商' }}</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-gray-700 mb-1">供应商名称 <span class="text-red-500">*</span></label>
            <input v-model.trim="form.provider_name" type="text" class="w-full border rounded-md px-3 py-2" placeholder="例如：openai">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">API基础地址(base_url) <span class="text-red-500">*</span></label>
            <input v-model.trim="form.base_url" type="text" class="w-full border rounded-md px-3 py-2" placeholder="例如：https://api.openai.com">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">派生端点路径</label>
            <input v-model.trim="form.derive_endpoint" type="text" class="w-full border rounded-md px-3 py-2" placeholder="/api/token/derive">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">请求方法</label>
            <select v-model="form.derive_method" class="w-full border rounded-md px-3 py-2">
              <option value="POST">POST</option>
              <option value="GET">GET</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">派生请求参数模板（JSON）</label>
            <textarea v-model="form.derive_params_text" class="w-full border rounded-md px-3 py-2" rows="4" placeholder='{"ttl": 3600}' />
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">响应中token字段路径</label>
            <input v-model.trim="form.response_token_field" type="text" class="w-full border rounded-md px-3 py-2" placeholder="token">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">响应中过期时间字段路径</label>
            <input v-model.trim="form.expires_in_field" type="text" class="w-full border rounded-md px-3 py-2" placeholder="expires_in">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">说明</label>
            <textarea v-model.trim="form.description" class="w-full border rounded-md px-3 py-2" rows="2" placeholder="例如：支持主Key派生子Key" />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="px-4 py-2 border rounded-md" @click="closeForm">取消</button>
          <button class="px-4 py-2 bg-primary text-white rounded-md" :disabled="saving" @click="submitForm">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'

const providers = ref([])
const showForm = ref(false)
const editingId = ref('')
const saving = ref(false)
const form = ref({
  provider_name: '',
  base_url: '',
  derive_endpoint: '/api/token/derive',
  derive_method: 'POST',
  derive_params_text: '{}',
  response_token_field: 'token',
  expires_in_field: 'expires_in',
  description: ''
})

const loadProviders = async () => {
  const response = await apiFetch('/api/system-admin/sub-token-providers/')
  if (!response.ok) {
    const err = new Error('加载派生Token供应商失败')
    if (response.traceId) err.traceId = response.traceId
    throw err
  }
  const data = await response.json()
  providers.value = data.items || []
}

const startCreate = () => {
  editingId.value = ''
  form.value = {
    provider_name: '',
    base_url: '',
    derive_endpoint: '/api/token/derive',
    derive_method: 'POST',
    derive_params_text: '{}',
    response_token_field: 'token',
    expires_in_field: 'expires_in',
    description: ''
  }
  showForm.value = true
}

const startEdit = (item) => {
  editingId.value = item.id
  form.value = {
    provider_name: item.provider_name,
    base_url: item.base_url,
    derive_endpoint: item.derive_endpoint || '/api/token/derive',
    derive_method: item.derive_method || 'POST',
    derive_params_text: typeof item.derive_params === 'object' ? JSON.stringify(item.derive_params, null, 2) : '{}',
    response_token_field: item.response_token_field || 'token',
    expires_in_field: item.expires_in_field || 'expires_in',
    description: item.description || ''
  }
  showForm.value = true
}

const closeForm = () => {
  showForm.value = false
}

// OPT-20260819-038: sub-token 供应商派生端点属资源/安全路径，防连点双发。
const submitFormGuard = createClickGuard()
const removeItemGuard = createClickGuard()

const submitForm = async () => {
  await submitFormGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    try {
      let derive_params = {}
      try {
        derive_params = JSON.parse(form.value.derive_params_text)
      } catch (e) {
        alert('派生请求参数必须是有效的JSON格式')
        saving.value = false
        return
      }

      const method = editingId.value ? 'PUT' : 'POST'
      const body = editingId.value
        ? {
            id: editingId.value,
            provider_name: form.value.provider_name,
            base_url: form.value.base_url,
            derive_endpoint: form.value.derive_endpoint,
            derive_method: form.value.derive_method,
            derive_params,
            response_token_field: form.value.response_token_field,
            expires_in_field: form.value.expires_in_field,
            description: form.value.description
          }
        : {
            provider_name: form.value.provider_name,
            base_url: form.value.base_url,
            derive_endpoint: form.value.derive_endpoint,
            derive_method: form.value.derive_method,
            derive_params,
            response_token_field: form.value.response_token_field,
            expires_in_field: form.value.expires_in_field,
            description: form.value.description
          }

      const response = await apiFetch('/api/system-admin/sub-token-providers/', {
        method,
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: JSON.stringify(body)
      })
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || errorData.detail || '保存失败')
        if (response.traceId) err.traceId = response.traceId
        throw err
      }
      showForm.value = false
      await loadProviders()
    } catch (error) {
      showRequestError(error.message || '保存失败', error)
    } finally {
      saving.value = false
    }
  })
}

const removeItem = async (id) => {
  await removeItemGuard.run(async ({ idempotencyKey }) => {
    const response = await apiFetch('/api/system-admin/sub-token-providers/', {
      method: 'DELETE',
      headers: mergeIdempotencyHeaders({}, idempotencyKey),
      body: JSON.stringify({ id })
    })
    if (!response.ok) {
      showRequestError('删除失败', response)
      return
    }
    await loadProviders()
  })
}

onMounted(() => {
  loadProviders().catch((error) => {
    showRequestError('加载派生Token供应商失败', error)
  })
})
</script>