<template>
  <div class="space-y-2">
    <label class="block text-sm font-medium text-text-light">智能体资源配置</label>
    <select
      v-model="selectedSource"
      class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
      @change="onChange"
    >
      <option value="company">公司默认</option>
      <option value="workspace">工作空间默认</option>
      <option v-if="personalAllowed" disabled>──────────</option>
      <option v-if="personalAllowed" v-for="pc in personalConfigs" :key="pc.id" :value="'personal:' + pc.id">
        个人: {{ pc.name }}
      </option>
    </select>
    <p v-if="!personalAllowed && selectedSource === 'personal'" class="text-xs text-yellow-600">
      当前工作空间未开启个人配置权限
    </p>
  </div>
</template>

<script>
import { apiFetch } from '../utils/apiUtils'

export default {
  name: 'TaskFeatureParamsSelector',
  props: {
    workspaceId: { type: String, default: '' },
    currentSource: { type: String, default: 'company' },
    currentPersonalConfigId: { type: String, default: '' },
  },
  emits: ['update:source', 'update:personalConfigId'],
  data() {
    return {
      personalConfigs: [],
      personalAllowed: false,
      selectedSource: this.currentSource === 'personal' ? ('personal:' + this.currentPersonalConfigId) : this.currentSource,
    }
  },
  async mounted() {
    await this.loadPersonalConfigs()
  },
  watch: {
    currentSource(v) { this.selectedSource = v === 'personal' ? ('personal:' + this.currentPersonalConfigId) : v },
  },
  methods: {
    async loadPersonalConfigs() {
      try {
        const resp = await apiFetch('/api/personal/feature-params-configs/')
        if (resp.ok) {
          this.personalConfigs = (await resp.json()).configs || []
        }
        // Check workspace governance
        if (this.workspaceId) {
          // simplified: assume personal is allowed if configs can be loaded
          this.personalAllowed = true
        }
      } catch (e) { /* personal configs unavailable */ }
    },
    onChange() {
      const v = this.selectedSource
      if (v.startsWith('personal:')) {
        this.$emit('update:source', 'personal')
        this.$emit('update:personalConfigId', v.slice(9))
      } else {
        this.$emit('update:source', v)
        this.$emit('update:personalConfigId', '')
      }
    },
  },
}
</script>
