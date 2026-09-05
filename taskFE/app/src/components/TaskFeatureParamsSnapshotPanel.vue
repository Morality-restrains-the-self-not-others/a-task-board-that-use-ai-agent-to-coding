<template>
  <div class="bg-white rounded-xl shadow p-6">
    <h3 class="text-lg font-bold text-text mb-4">智能体资源配置运行记录</h3>

    <div v-if="loading" class="text-sm text-text-light">加载中...</div>
    <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
    <div v-else-if="snapshots.length === 0" class="text-sm text-text-light">暂无运行记录</div>

    <div v-else class="space-y-3">
      <div v-for="snap in snapshots" :key="snap.id" class="border border-gray-200 rounded-lg p-4">
        <div class="flex items-center justify-between">
          <div>
            <span class="font-semibold text-sm text-text">{{ snap.source_display_name }}</span>
            <span class="text-xs text-text-light ml-3">{{ formatDate(snap.created_at) }}</span>
          </div>
          <button @click="toggleDetail(snap.id)" class="text-xs text-primary hover:underline">
            {{ expandedId === snap.id ? '收起' : '详情' }}
          </button>
        </div>
        <div class="text-sm text-text-light mt-1">
          智能体: {{ snap.agent_model }} ({{ snap.agent_model_provider }}) · 最大步数: {{ snap.agent_max_steps }}
        </div>

        <!-- Expanded Detail -->
        <div v-if="expandedId === snap.id" class="mt-3 pt-3 border-t border-gray-100 space-y-2">
          <div v-for="(p, idx) in (snap.providers_summary || [])" :key="idx" class="bg-gray-50 p-2 rounded text-xs">
            <div class="font-mono">{{ p.provider }}</div>
            <div class="text-text-light">{{ p.base_url }}</div>
            <div class="text-text-light">API Key Hash: {{ p.api_key_hash }}</div>
            <div class="text-text-light">Models: {{ (p.supported_models || []).join(', ') }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { apiFetch } from '../utils/apiUtils'

export default {
  name: 'TaskFeatureParamsSnapshotPanel',
  props: {
    taskId: { type: String, required: true },
  },
  data() {
    return {
      snapshots: [],
      loading: false,
      error: '',
      expandedId: null,
    }
  },
  async mounted() {
    await this.load()
  },
  methods: {
    async load() {
      this.loading = true
      this.error = ''
      try {
        const resp = await apiFetch(`/api/tasks/${this.taskId}/feature-params-snapshots/`)
        if (!resp.ok) throw new Error('加载失败')
        this.snapshots = (await resp.json()).snapshots || []
      } catch (e) {
        this.error = '加载运行记录失败: ' + e.message
      } finally {
        this.loading = false
      }
    },
    toggleDetail(id) {
      this.expandedId = this.expandedId === id ? null : id
    },
    formatDate(iso) {
      if (!iso) return ''
      return new Date(iso).toLocaleString()
    },
  },
}
</script>
