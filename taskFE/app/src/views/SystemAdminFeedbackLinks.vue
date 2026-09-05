<template>
  <div class="p-6 max-w-4xl" data-alias="view-system-admin-feedback-links">
    <h2 class="text-xl font-semibold text-gray-900 mb-2">意见与建议链接</h2>
    <p class="text-sm text-gray-500 mb-6">配置分组外链。租户累计消耗达到全部阈值后，成员在侧栏「意见与建议」中看到同一套链接。</p>
    <p v-if="errorMsg" class="text-sm text-danger mb-3" data-testid="feedback-admin-error" :data-traceId="errorTraceId || undefined">{{ errorMsg }}</p>
    <p v-if="successMsg" class="text-sm text-success mb-3" data-testid="feedback-admin-success">{{ successMsg }}</p>

    <section class="bg-white border rounded-lg p-4 mb-6 space-y-3">
      <h3 class="font-medium">{{ editingId ? '编辑组' : '新建组' }}</h3>
      <label class="block text-sm">组名
        <input v-model="form.name" class="mt-1 w-full border rounded px-3 py-2" data-testid="feedback-group-name" />
      </label>
      <label class="block text-sm">排序
        <input v-model.number="form.sort_order" type="number" class="mt-1 w-32 border rounded px-3 py-2" data-testid="feedback-group-sort" />
      </label>
      <label class="inline-flex items-center gap-2 text-sm">
        <input v-model="form.enabled" type="checkbox" data-testid="feedback-group-enabled" /> 启用
      </label>
      <div>
        <p class="text-sm font-medium mb-1">阈值（AND，达到即可见）</p>
        <div v-for="(t, i) in form.thresholds" :key="i" class="flex gap-2 mb-2">
          <select v-model="t.resource_kind" class="border rounded px-2 py-1" :data-testid="`feedback-th-kind-${i}`">
            <option v-for="k in kinds" :key="k.kind" :value="k.kind">{{ k.display_name }}（{{ k.unit }}）</option>
          </select>
          <input v-model.number="t.min_quantity" type="number" class="border rounded px-2 py-1 w-28" :data-testid="`feedback-th-qty-${i}`" />
          <button type="button" class="text-sm text-gray-600" @click="form.thresholds.splice(i, 1)">移除</button>
        </div>
        <button type="button" class="text-sm text-primary" data-testid="feedback-add-threshold" @click="addThreshold">添加阈值</button>
      </div>
      <div>
        <p class="text-sm font-medium mb-1">链接</p>
        <div v-for="(l, i) in form.links" :key="i" class="grid grid-cols-1 gap-2 mb-2">
          <input v-model="l.title" class="border rounded px-2 py-1" placeholder="标题" :data-testid="`feedback-link-title-${i}`" />
          <input v-model="l.url" class="border rounded px-2 py-1" placeholder="https://" :data-testid="`feedback-link-url-${i}`" />
          <label class="inline-flex items-center gap-2 text-sm">
            <input v-model="l.enabled" type="checkbox" /> 启用
          </label>
        </div>
        <button type="button" class="text-sm text-primary" data-testid="feedback-add-link" @click="addLink">添加链接</button>
      </div>
      <button
        type="button"
        class="btn btn-primary"
        data-testid="feedback-save"
        :disabled="saving"
        :aria-busy="saving ? 'true' : 'false'"
        @click="save"
      >{{ saving ? '保存中…' : '保存' }}</button>
    </section>

    <ul class="space-y-2" data-testid="feedback-group-list">
      <li v-for="g in groups" :key="g.id" class="border rounded p-3 flex justify-between items-center">
        <span>{{ g.name }} <span class="text-xs text-gray-500">{{ g.enabled ? '启用' : '禁用' }}</span></span>
        <span class="flex gap-2">
          <button type="button" class="text-sm text-primary" @click="edit(g)">编辑</button>
          <button
            type="button"
            class="text-sm text-danger"
            :disabled="deleting"
            @click="remove(g)"
          >删除</button>
        </span>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const groups = ref([])
const kinds = ref([])
const editingId = ref('')
const saving = ref(false)
const deleting = ref(false)
const errorMsg = ref('')
const errorTraceId = ref('')
const successMsg = ref('')
const saveGuard = createClickGuard()
const deleteGuard = createClickGuard()

const form = reactive({
  name: '',
  sort_order: 0,
  enabled: true,
  thresholds: [],
  links: [{ title: '', url: '', enabled: true, sort_order: 0 }],
})

function resetForm() {
  editingId.value = ''
  form.name = ''
  form.sort_order = 0
  form.enabled = true
  form.thresholds = []
  form.links = [{ title: '', url: '', enabled: true, sort_order: 0 }]
}

function addThreshold() {
  const kind = kinds.value[0]?.kind || 'task_post'
  form.thresholds.push({ resource_kind: kind, min_quantity: 0 })
}

function addLink() {
  form.links.push({ title: '', url: '', enabled: true, sort_order: form.links.length })
}

function edit(g) {
  editingId.value = g.id
  form.name = g.name
  form.sort_order = g.sort_order || 0
  form.enabled = g.enabled !== false
  form.thresholds = (g.thresholds || []).map((t) => ({
    resource_kind: t.resource_kind,
    min_quantity: t.min_quantity,
  }))
  form.links = (g.links || []).map((l) => ({
    title: l.title,
    url: l.url,
    enabled: l.enabled !== false,
    sort_order: l.sort_order || 0,
  }))
  if (!form.links.length) addLink()
}

async function load() {
  const [gRes, kRes] = await Promise.all([
    apiFetch('/api/system-admin/feedback-link-groups/', { credentials: 'include' }),
    apiFetch('/api/system-admin/feedback-resource-kinds/', { credentials: 'include' }),
  ])
  if (gRes.ok) {
    const data = await gRes.json()
    groups.value = data.results || []
  }
  if (kRes.ok) {
    const data = await kRes.json()
    kinds.value = data.results || []
  }
}

async function save() {
  const result = await saveGuard.run(async ({ headers }) => {
    saving.value = true
    errorMsg.value = ''
    errorTraceId.value = ''
    try {
      const path = editingId.value
        ? `/api/system-admin/feedback-link-groups/${encodeURIComponent(editingId.value)}/`
        : '/api/system-admin/feedback-link-groups/'
      const method = editingId.value ? 'PUT' : 'POST'
      const r = await apiFetch(path, {
        method,
        credentials: 'include',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, headers['Idempotency-Key']),
        body: JSON.stringify({
          name: form.name,
          sort_order: form.sort_order,
          enabled: form.enabled,
          thresholds: form.thresholds,
          links: form.links.filter((l) => l.title && l.url),
        }),
      })
      if (!r.ok) {
        const err = await r.json().catch(() => ({}))
        errorMsg.value = err.error || err.detail || '保存失败'
        errorTraceId.value = extractTraceId(r, err)
        return
      }
      successMsg.value = '已保存'
      resetForm()
      await load()
    } finally {
      saving.value = false
    }
  })
  return result
}

async function remove(g) {
  await deleteGuard.run(async ({ headers }) => {
    deleting.value = true
    try {
      await apiFetch(`/api/system-admin/feedback-link-groups/${encodeURIComponent(g.id)}/`, {
        method: 'DELETE',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({}, headers['Idempotency-Key']),
      })
      await load()
    } finally {
      deleting.value = false
    }
  })
}

onMounted(load)
</script>
