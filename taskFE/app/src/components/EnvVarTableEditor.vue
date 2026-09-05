<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between mb-2">
      <label class="text-sm font-medium text-text-light">用户自定义变量</label>
      <div class="flex gap-2">
        <button type="button" class="px-3 py-1 text-xs border border-gray-300 rounded-md text-gray-600 hover:bg-gray-50" @click="showImport = true">📥 导入 .env</button>
        <button type="button" class="px-3 py-1 text-xs border border-gray-300 rounded-md text-gray-600 hover:bg-gray-50" @click="exportEnv">📤 导出 .env</button>
      </div>
    </div>

    <!-- Table Header -->
    <div class="grid grid-cols-12 gap-2 px-3 py-2 bg-gray-50 rounded-t-md border border-gray-200 text-xs font-medium text-gray-600">
      <div class="col-span-3">变量名 <span class="text-red-400">*</span></div>
      <div class="col-span-6">变量值</div>
      <div class="col-span-2">说明</div>
      <div class="col-span-1"></div>
    </div>

    <!-- Table Body -->
    <div class="border border-t-0 border-gray-200 rounded-b-md divide-y divide-gray-100">
      <div v-for="(row, idx) in items" :key="idx" class="grid grid-cols-12 gap-2 px-3 py-2 items-start">
        <!-- Key -->
        <div class="col-span-3 relative">
          <input
            v-model.trim="row.key"
            type="text"
            placeholder="MY_VAR"
            :class="keyInputClass(row)"
            class="w-full px-2 py-1.5 border rounded-md text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @input="onKeyInput(row, idx)"
            @focus="row._showSuggestions = true"
            @blur="row._showSuggestions = false"
          />
          <!-- Validation tooltip -->
          <div v-if="row._keyError" class="absolute left-0 top-full mt-1 z-10 bg-red-50 border border-red-200 text-red-700 text-xs rounded px-2 py-1 shadow-lg whitespace-nowrap">
            {{ row._keyError }}
          </div>
          <!-- Reserved key warning -->
          <div v-else-if="row._keyWarning" class="absolute left-0 top-full mt-1 z-10 bg-yellow-50 border border-yellow-200 text-yellow-700 text-xs rounded px-2 py-1 shadow-lg whitespace-nowrap">
            ⚠ {{ row._keyWarning }}
          </div>
          <!-- Suggestions dropdown -->
          <div v-if="row._showSuggestions && !row.key" class="absolute left-0 top-full mt-1 z-10 bg-white border border-gray-200 rounded-md shadow-lg text-xs w-48">
            <div class="px-2 py-1 text-gray-400 border-b">常用变量</div>
            <div v-for="tmpl in templateKeys" :key="tmpl" class="px-2 py-1 hover:bg-primary/10 cursor-pointer text-gray-700" @mousedown.prevent="row.key = tmpl; row._showSuggestions = false">{{ tmpl }}</div>
          </div>
        </div>
        <!-- Value -->
        <div class="col-span-6 flex items-center gap-1">
          <input
            :type="isSensitive(row) && !row._revealed ? 'password' : 'text'"
            v-model="row.value"
            placeholder="变量值"
            class="flex-1 px-2 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @input="emitChange"
          />
          <button v-if="isSensitive(row)" type="button" class="px-1.5 py-1 text-xs text-gray-400 hover:text-gray-600" @click="row._revealed = !row._revealed" :title="row._revealed ? '隐藏' : '显示'">
            {{ row._revealed ? '🙈' : '👁' }}
          </button>
        </div>
        <!-- Description -->
        <div class="col-span-2">
          <input
            v-model.trim="row.description"
            type="text"
            placeholder="可选说明"
            class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @input="emitChange"
          />
        </div>
        <!-- Remove -->
        <div class="col-span-1 pt-1">
          <button type="button" class="text-red-400 hover:text-red-600 hover:bg-red-50 rounded p-1" @click="removeRow(idx)">✕</button>
        </div>
      </div>
    </div>

    <button type="button" class="px-3 py-1 text-sm border border-dashed border-gray-300 rounded-md text-gray-500 hover:border-primary hover:text-primary w-full" @click="addRow">+ 添加变量</button>

    <!-- Import Modal -->
    <div v-if="showImport" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl shadow-xl p-6 w-full max-w-lg">
        <h4 class="text-lg font-bold mb-3">导入 .env 文件</h4>
        <p class="text-sm text-gray-500 mb-3">粘贴 .env 格式内容，每行格式：KEY=VALUE</p>
        <textarea v-model="importText" rows="8" class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary" placeholder="DEBUG=true&#10;LOG_LEVEL=info&#10;MY_ENDPOINT=https://api.daydaymoney.com"></textarea>
        <div class="flex justify-end gap-2 mt-4">
          <button class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50" @click="showImport = false">取消</button>
          <button class="btn-primary px-4 py-2" @click="doImport">导入</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const RESERVED_PREFIXES = ['TASK_', 'ACCESS_TOKEN', 'BUSINESS_API_']
const KEY_PATTERN = /^[A-Z_][A-Z0-9_]*$/
const SENSITIVE_PATTERN = /token|key|secret|password|api_key/i
const TEMPLATE_KEYS = ['DEBUG_AGENT', 'LOG_LEVEL', 'CUSTOM_ENDPOINT', 'TIMEOUT_SEC', 'MAX_RETRIES', 'API_BASE_URL', 'WS_ENDPOINT', 'SSE_ENDPOINT']

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue'])

const items = ref([])
const showImport = ref(false)
const importText = ref('')
const templateKeys = ref(TEMPLATE_KEYS)

function createEmptyRow() {
  return {
    key: '',
    value: '',
    description: '',
    _revealed: false,
    _keyError: '',
    _keyWarning: '',
    _showSuggestions: false,
  }
}

function mapIncomingRows(v) {
  return (v || []).map((e) => ({
    key: e.key || '',
    value: e.value || '',
    description: e.description || '',
    _revealed: false,
    _keyError: '',
    _keyWarning: '',
    _showSuggestions: false,
  }))
}

/** 仅含已填写变量名的行，供 v-model / 保存 / 预览；空草稿行只留在本地 items */
function committedPayload(rows) {
  return (rows || [])
    .filter((r) => (r.key || '').trim() !== '')
    .map((r) => ({
      key: r.key.trim(),
      value: r.value || '',
      description: r.description || '',
    }))
}

function payloadsEqual(a, b) {
  return JSON.stringify(a) === JSON.stringify(b)
}

watch(() => props.modelValue, (v) => {
  const incoming = mapIncomingRows(v)
  const incomingCommitted = committedPayload(incoming)
  const localCommitted = committedPayload(items.value)
  // 父组件回写与本地已提交内容一致时，保留空草稿行与输入态（_revealed 等）
  if (payloadsEqual(localCommitted, incomingCommitted)) {
    return
  }
  items.value = incoming
}, { immediate: true, deep: true })

function isSensitive(row) {
  return SENSITIVE_PATTERN.test(row.value || '') || SENSITIVE_PATTERN.test(row.key || '')
}

function keyInputClass(row) {
  if (row._keyError) return 'border-red-400 bg-red-50'
  if (row._keyWarning) return 'border-yellow-400 bg-yellow-50'
  return 'border-gray-300'
}

function onKeyInput(row, idx) {
  const key = (row.key || '').trim()
  row._keyError = ''
  row._keyWarning = ''
  if (key) {
    if (!KEY_PATTERN.test(key)) {
      row._keyError = '仅允许大写字母、数字和下划线，以字母或下划线开头'
    }
    const upper = key.toUpperCase()
    for (const prefix of RESERVED_PREFIXES) {
      if (upper.startsWith(prefix)) {
        row._keyWarning = '系统保留变量前缀，自定义值可能被运行时覆盖'
        break
      }
    }
  }
  emitChange()
}

function addRow() {
  // 空草稿不立刻 emit：否则会被 filter 掉，watch 回写后行立刻消失（表现为「添加变量无反应」）
  items.value.push(createEmptyRow())
}

function removeRow(idx) {
  items.value.splice(idx, 1)
  emitChange()
}

function emitChange() {
  emit('update:modelValue', committedPayload(items.value))
}

function doImport() {
  const text = importText.value.trim()
  if (!text) return
  const lines = text.split('\n').filter(l => l.trim() && !l.trim().startsWith('#'))
  for (const line of lines) {
    const eqIdx = line.indexOf('=')
    if (eqIdx > 0) {
      const key = line.substring(0, eqIdx).trim()
      const value = line.substring(eqIdx + 1).trim()
      const cleanValue = (value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))
        ? value.slice(1, -1)
        : value
      items.value.push({ key, value: cleanValue, description: '', _revealed: false, _keyError: '', _keyWarning: '', _showSuggestions: false })
    }
  }
  importText.value = ''
  showImport.value = false
  emitChange()
}

function exportEnv() {
  const lines = items.value
    .filter(r => (r.key || '').trim() !== '')
    .map(r => `${r.key.trim()}=${r.value || ''}`)
  const blob = new Blob([lines.join('\n') + '\n'], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'env-vars.env'
  a.click()
  URL.revokeObjectURL(url)
}
</script>
