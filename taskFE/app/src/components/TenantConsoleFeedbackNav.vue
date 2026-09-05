<template>
  <div data-testid="tenant-console-feedback-nav">
    <div
      class="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group cursor-pointer"
      :class="[open ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/10', collapsed ? 'justify-center px-0' : '']"
      title="意见与建议"
      data-testid="tenant-feedback-toggle"
      @click="toggle"
    >
      <svg class="w-5 h-5 text-primary shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M21 12c0 4.418-4.03 8-9 8a9.86 9.86 0 01-4-.8L3 20l1.2-3.6A7.5 7.5 0 013 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
      </svg>
      <span v-show="!collapsed" class="text-sm leading-5">意见与建议</span>
      <svg v-show="!collapsed" class="w-4 h-4 ml-auto shrink-0 transition-transform" :class="{ 'rotate-90': open }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
      </svg>
    </div>
    <div v-if="open && !collapsed" class="ml-11 mt-1 space-y-2" data-testid="tenant-feedback-submenu">
      <p v-if="!groups.length" class="px-3 py-2 text-sm text-text-light" data-testid="tenant-feedback-empty">暂无链接</p>
      <div v-for="g in groups" :key="g.id" data-testid="tenant-feedback-group">
        <div class="px-3 py-1 text-xs font-semibold text-text-light" data-testid="tenant-feedback-group-name">{{ g.name }}</div>
        <!-- Anti-Replay-OK: 真实外链 href，无写请求 -->
        <a
          v-for="l in g.links || []"
          :key="l.id"
          :href="l.url"
          class="block px-3 py-2 rounded-lg text-sm leading-5 transition-colors hover:bg-primary/5 text-text-light hover:text-primary"
          target="_blank"
          rel="noopener noreferrer"
          data-testid="tenant-feedback-link"
        >{{ l.title }}</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'

const props = defineProps({
  tenantId: { type: String, default: '' },
  collapsed: { type: Boolean, default: false },
})

const emit = defineEmits(['expand-sidebar'])

const open = ref(false)
const groups = ref([])

async function load() {
  const tid = String(props.tenantId || '').trim()
  if (!tid) {
    groups.value = []
    return
  }
  const r = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/feedback-links/`, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!r.ok) {
    groups.value = []
    return
  }
  const data = await r.json()
  groups.value = Array.isArray(data?.groups) ? data.groups : []
}

function toggle() {
  if (props.collapsed) {
    emit('expand-sidebar')
    open.value = true
    return
  }
  open.value = !open.value
}

onMounted(load)
watch(() => props.tenantId, load)
</script>
