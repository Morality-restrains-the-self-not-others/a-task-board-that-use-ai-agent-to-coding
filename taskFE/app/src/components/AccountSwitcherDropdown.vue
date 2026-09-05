<template>
  <div class="relative" data-testid="account-switcher" ref="rootEl">
    <div
      class="inline-flex items-center gap-2 text-gray-700 cursor-pointer"
      data-testid="account-switcher-trigger"
      @click="onTriggerClick"
      @dblclick.prevent="onTriggerDblClick"
    >
      <a
        :href="profileHref"
        class="shrink-0"
        :title="'个人资料: ' + displayName"
        data-testid="account-switcher-avatar-link"
        @click.stop
      >
        <img
          :src="avatarSrc"
          alt=""
          class="w-8 h-8 rounded-full object-cover border border-gray-200 shrink-0 hover:opacity-80 transition-opacity"
          width="32"
          height="32"
        >
      </a>
      <a
        :href="profileHref"
        class="flex flex-col items-start min-w-0 max-w-[9rem] leading-tight no-underline text-inherit hover:text-primary transition-colors"
        :title="'个人资料: ' + displayName"
        data-testid="account-switcher-name-link"
        @click.stop
      >
        <span class="max-w-full truncate text-sm">{{ displayName }}</span>
        <span
          v-if="publicIp"
          class="max-w-full truncate text-[11px] text-gray-500 font-mono"
          data-testid="account-switcher-public-ip"
          :title="publicIp"
        >{{ publicIp }}</span>
      </a>
      <button
        type="button"
        class="shrink-0 cursor-pointer hover:text-primary transition-colors"
        :aria-expanded="open ? 'true' : 'false'"
        aria-haspopup="menu"
        :aria-label="open ? '收起账号菜单' : '展开账号菜单'"
      >
        <svg
          class="w-3.5 h-3.5 opacity-70"
          :class="{ 'rotate-180': open }"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.17l3.71-3.94a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
        </svg>
      </button>
    </div>

    <div
      v-if="open"
      class="absolute right-0 mt-2 w-64 bg-white border border-gray-200 rounded-md shadow-lg py-1 z-[60]"
      role="menu"
      data-testid="account-switcher-menu"
    >
      <a
        :href="profileHref"
        class="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
        data-testid="account-switcher-profile"
        role="menuitem"
        @click="close"
      >
        个人资料
      </a>
      <button
        type="button"
        class="w-full px-3 py-2 text-sm text-left text-red-600 hover:bg-gray-50"
        data-testid="account-switcher-logout"
        role="menuitem"
        @click="$emit('logout')"
      >
        退出当前
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { fetchPublicClientIp } from '../utils/publicClientIp.js'

// 去掉多账号登录：不再接收 accounts/activeUserId/switching，也不渲染账号切换列表与「添加账号」入口
const props = defineProps({
  displayName: { type: String, default: '未设置昵称' },
  avatarSrc: { type: String, required: true },
  profilePath: { type: [String, Object], required: true },
})

defineEmits(['logout'])

/** 真实 <a href>（禁止 router-link 点击拦截） */
const profileHref = computed(() => {
  const to = props.profilePath
  if (typeof to === 'string') return to
  const path = String(to?.path || '/profile/').trim() || '/profile/'
  const query = to?.query && typeof to.query === 'object' ? to.query : null
  if (!query) return path
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(query)) {
    if (v == null || v === '') continue
    params.set(k, Array.isArray(v) ? String(v[0]) : String(v))
  }
  const s = params.toString()
  return s ? `${path}?${s}` : path
})

const open = ref(false)
const rootEl = ref(null)
const publicIp = ref('')
let clickOpenedAt = 0

function close() {
  open.value = false
}

function onTriggerClick() {
  // 双击会先触发两次 click：用短窗忽略第二次 click，由 dblclick 收起
  const now = Date.now()
  if (now - clickOpenedAt < 350 && open.value) {
    return
  }
  open.value = true
  clickOpenedAt = now
}

function onTriggerDblClick() {
  open.value = false
  clickOpenedAt = 0
}

function onDocClick(e) {
  if (!open.value) return
  const el = rootEl.value
  if (el && !el.contains(e.target)) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  fetchPublicClientIp()
    .then((ip) => {
      publicIp.value = ip
    })
    .catch((err) => {
      console.warn('[AccountSwitcherDropdown] public client ip unavailable', err)
    })
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
})

defineExpose({ open, close, publicIp })
</script>
