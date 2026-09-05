<template>
  <div class="border border-primary/30 rounded-lg p-4 space-y-3 bg-primary/5">
    <h4 class="font-semibold text-text text-sm">新增 GitLab 区域</h4>
    <div class="grid grid-cols-2 gap-3 text-sm">
      <div>
        <label class="block text-xs text-text-light mb-1">云服务商 <span class="text-danger">*</span></label>
        <input v-model="form.cloud_provider" type="text" placeholder="如：tencent / aws / azure / aliyun / huawei / gcp"
               class="w-full px-2 py-1.5 border border-border rounded text-sm"
               list="cloud-provider-options" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">区域名称 <span class="text-danger">*</span></label>
        <input v-model="form.name" type="text" placeholder="如：腾讯新加坡一区"
               class="w-full px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">标识符 (slug) <span class="text-danger">*</span></label>
        <input v-model="form.slug" type="text" placeholder="如：tencent-singapore-1"
               class="w-full px-2 py-1.5 border border-border rounded text-sm font-mono" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">描述</label>
        <input v-model="form.description" type="text" placeholder="区域描述"
               class="w-full px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">排序</label>
        <input v-model.number="form.sort_order" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">GitLab API 地址</label>
        <input v-model="form.gitlab_api_base" type="text" placeholder="http://127.0.0.1:8012"
               class="w-full px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">GitLab Web 地址</label>
        <input v-model="form.gitlab_web_url" type="text" placeholder="https://gitlab.daydaymoney.com"
               class="w-full px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">Admin Private Token</label>
        <input v-model="form.admin_private_token" type="password" placeholder="GitLab 管理员 Token"
               class="w-full px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">磁盘总量 (GB)</label>
        <input v-model.number="form.total_disk_gb" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">流量总量 (GB)</label>
        <input v-model.number="form.total_traffic_gb" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">总带宽 (Mbps)</label>
        <input v-model.number="form.total_bandwidth_mbps" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">剩余带宽 (Mbps)</label>
        <input v-model.number="form.remaining_bandwidth_mbps" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">区域模式</label>
        <select v-model="form.access_mode" aria-label="区域模式"
                class="w-full px-2 py-1.5 border border-border rounded text-sm">
          <option value="release">发布模式</option>
          <option value="development">开发模式（仅测试角色可用）</option>
        </select>
      </div>
      <div class="flex items-center gap-2 pt-5">
        <label class="text-xs text-text-light select-none cursor-pointer flex items-center gap-1">
          <input v-model="form.bandwidth_shared" type="checkbox" class="rounded" />
          带宽共享分区
        </label>
      </div>
    </div>
    <div class="flex gap-2 pt-2">
      <button class="px-4 py-2 bg-primary text-white rounded text-sm hover:bg-primary/90 disabled:opacity-60"
              :disabled="creating"
              @click="submit">
        {{ creating ? '创建中...' : '创建区域' }}
      </button>
      <button class="px-4 py-2 border border-border rounded text-sm hover:bg-gray-50"
              @click="$emit('cancel')">取消</button>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'

defineProps({
  creating: { type: Boolean, default: false },
})
const emit = defineEmits(['submit', 'cancel'])

const form = reactive({
  name: '',
  slug: '',
  description: '',
  sort_order: 0,
  gitlab_api_base: 'http://127.0.0.1:8012',
  gitlab_web_url: '',
  admin_private_token: '',
  cloud_provider: '',
  total_disk_gb: 100,
  total_traffic_gb: 1000,
  total_bandwidth_mbps: 0,
  remaining_bandwidth_mbps: 0,
  bandwidth_shared: false,
  access_mode: 'release',
})

function submit() {
  emit('submit', {
    name: form.name,
    slug: form.slug,
    description: form.description,
    sort_order: form.sort_order,
    gitlab_api_base: form.gitlab_api_base,
    gitlab_web_url: form.gitlab_web_url,
    admin_private_token: form.admin_private_token,
    cloud_provider: form.cloud_provider,
    total_disk_gb: form.total_disk_gb,
    total_traffic_gb: form.total_traffic_gb,
    total_bandwidth_mbps: form.total_bandwidth_mbps,
    remaining_bandwidth_mbps: form.remaining_bandwidth_mbps,
    bandwidth_shared: !!form.bandwidth_shared,
    access_mode: form.access_mode === 'development' ? 'development' : 'release',
  })
}
</script>
