<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-900">推荐供应商列表</h2>
        <p class="text-sm text-gray-500 mt-1">维护租户侧展示的推荐大模型供应商（支持排序与说明）</p>
      </div>
      <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90" @click="startCreate">
        新增供应商
      </button>
    </div>

    <div class="bg-yellow-50 border border-yellow-200 text-yellow-800 text-sm rounded-lg p-3 mb-4">
      免责说明：以下推荐仅供参考，不构成任何采购或质量保证承诺。请结合合规、稳定性、安全性和服务协议自行评估供应商可靠性。
    </div>

    <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">排序</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">文档链接</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">说明</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          <tr v-for="(item, index) in providers" :key="item.id">
            <td class="px-4 py-3 text-sm text-gray-700">
              <div class="flex items-center gap-2">
                <span>{{ index + 1 }}</span>
                <button class="text-xs px-2 py-1 border rounded disabled:opacity-40" :disabled="index === 0" @click="moveUp(index)">上移</button>
                <button class="text-xs px-2 py-1 border rounded disabled:opacity-40" :disabled="index === providers.length - 1" @click="moveDown(index)">下移</button>
              </div>
            </td>
            <td class="px-4 py-3 text-sm text-gray-900">{{ item.name }}</td>
            <td class="px-4 py-3 text-sm">
              <a :href="item.docs_url" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">
                {{ item.docs_url }}
              </a>
            </td>
            <td class="px-4 py-3 text-sm text-gray-700">{{ item.description || '-' }}</td>
            <td class="px-4 py-3 text-sm">
              <button class="text-primary hover:underline mr-3" @click="startEdit(item)">编辑</button>
              <button class="text-red-600 hover:underline" @click="removeItem(item.id)">删除</button>
            </td>
          </tr>
          <tr v-if="providers.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-sm text-gray-500">暂无推荐供应商</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showForm" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg w-full max-w-lg p-6">
        <h3 class="text-lg font-semibold mb-4">{{ editingId ? '编辑供应商' : '新增供应商' }}</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-gray-700 mb-1">供应商名称</label>
            <input v-model.trim="form.name" type="text" class="w-full border rounded-md px-3 py-2" placeholder="例如：DeepSeek">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">文档链接</label>
            <input v-model.trim="form.docs_url" type="text" class="w-full border rounded-md px-3 py-2" placeholder="https://...">
          </div>
          <div>
            <label class="block text-sm text-gray-700 mb-1">说明</label>
            <textarea v-model.trim="form.description" class="w-full border rounded-md px-3 py-2" rows="3" placeholder="例如：适合通用对话与代码场景" />
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
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const providers = ref([])
const showForm = ref(false)
const editingId = ref('')
const saving = ref(false)
const form = ref({ name: '', docs_url: '', description: '' })

const loadProviders = async () => {
  const response = await apiFetch('/api/system-admin/recommended-llm-providers/')
  if (!response.ok) {
    const error = new Error('加载推荐供应商失败')
    error._errorData = response._errorData
    if (response.traceId) error.traceId = response.traceId
    throw error
  }
  const data = await response.json()
  providers.value = data.items || []
}

const startCreate = () => {
  editingId.value = ''
  form.value = { name: '', docs_url: '', description: '' }
  showForm.value = true
}

const startEdit = (item) => {
  editingId.value = item.id
  form.value = { name: item.name, docs_url: item.docs_url, description: item.description || '' }
  showForm.value = true
}

const closeForm = () => {
  showForm.value = false
}

const submitFormGuard = createClickGuard()
const removeItemGuard = createClickGuard()
const saveOrderGuard = createClickGuard()

const submitForm = async () => {
  // OPT-20260819-038: 保存/新增供应商是资源写操作，防连点双发 POST/PUT
  await submitFormGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    try {
      const method = editingId.value ? 'PUT' : 'POST'
      const body = editingId.value ? { id: editingId.value, ...form.value } : form.value
      const response = await apiFetch('/api/system-admin/recommended-llm-providers/', {
        method,
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: JSON.stringify(body)
      })
      if (!response.ok) {
        const err = new Error('保存失败')
        err.traceId = response.traceId || ''
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
  // OPT-20260819-038: 删除是破坏性写操作，防连点双发 DELETE
  await removeItemGuard.run(async ({ idempotencyKey }) => {
    const response = await apiFetch('/api/system-admin/recommended-llm-providers/', {
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

const saveOrder = async () => {
  // OPT-20260819-038: 上移/下移连点会连续 PUT 排序，防重入
  await saveOrderGuard.run(async ({ idempotencyKey }) => {
    for (let i = 0; i < providers.value.length; i += 1) {
      const item = providers.value[i]
      await apiFetch('/api/system-admin/recommended-llm-providers/', {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: JSON.stringify({ id: item.id, sort_order: i })
      })
    }
    await loadProviders()
  })
}

const moveUp = async (index) => {
  if (index === 0) return
  const list = [...providers.value]
  const temp = list[index - 1]
  list[index - 1] = list[index]
  list[index] = temp
  providers.value = list
  await saveOrder()
}

const moveDown = async (index) => {
  if (index >= providers.value.length - 1) return
  const list = [...providers.value]
  const temp = list[index + 1]
  list[index + 1] = list[index]
  list[index] = temp
  providers.value = list
  await saveOrder()
}

onMounted(() => {
  loadProviders().catch((error) => {
    showRequestError('加载推荐供应商失败', error)
  })
})
</script>
