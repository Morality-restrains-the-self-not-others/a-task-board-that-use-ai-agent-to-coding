<template>
  <!-- 无仓库：ready 时真实 <a> 去价格页；unknown/error 留当前页（禁止假导航） -->
  <!-- Anti-Replay-OK: 真实 <a href> 只读导航 -->
  <a
    v-if="isUserAuthenticated && jumpable.length === 0"
    :href="emptyNavHref"
    class="nav-link hover:text-primary transition-colors shrink-0"
    data-testid="nav-git-service"
  >
    代码仓库
    <span
      v-if="showVipBadge"
      class="ml-1 px-1.5 py-0.5 rounded text-[10px] font-semibold leading-none bg-amber-100 text-amber-700 border border-amber-300 align-middle"
      data-testid="nav-git-service-vip-badge"
    >VIP1</span>
  </a>

  <!-- ≥1 已购/获赠区域：悬停或单击展开下拉（双击收起） -->
  <div
    v-else-if="isUserAuthenticated"
    class="relative shrink-0"
    data-testid="nav-git-service-wrap"
    @mouseenter="openMenu"
    @mouseleave="closeMenu"
  >
    <button
      type="button"
      class="nav-link hover:text-primary transition-colors inline-flex items-center gap-1"
      data-testid="nav-git-service"
      :aria-expanded="menuOpen ? 'true' : 'false'"
      aria-haspopup="menu"
      @click="toggleMenu"
      @dblclick.prevent="closeMenu"
      @keydown.esc="closeMenu"
    >
      <span>代码仓库</span>
      <span
        v-if="showVipBadge"
        class="ml-0.5 px-1.5 py-0.5 rounded text-[10px] font-semibold leading-none bg-amber-100 text-amber-700 border border-amber-300 align-middle"
        data-testid="nav-git-service-vip-badge"
      >VIP1</span>
      <svg
        class="w-3.5 h-3.5 shrink-0 transition-transform"
        :class="{ 'rotate-180': menuOpen }"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    <div
      v-if="menuOpen"
      class="absolute left-0 mt-1 w-56 rounded-lg bg-white shadow-lg ring-1 ring-black/5 border border-gray-100 py-1 z-50"
      role="menu"
      data-testid="nav-git-service-menu"
    >
      <a
        v-for="item in jumpable"
        :key="item.region"
        :href="item.gitlab_web_url"
        target="_blank"
        rel="noopener noreferrer"
        role="menuitem"
        class="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 hover:bg-gray-50 hover:text-primary"
        data-testid="nav-git-service-region"
        @click="closeMenu"
      >
        <span class="truncate">{{ item.region_name || item.region }}</span>
      </a>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  isUserAuthenticated: { type: Boolean, default: false },
  gitResources: { type: Array, default: () => [] },
  membershipTier: { type: String, default: '' },
  currentPageHref: { type: String, default: '/' },
  pricingHref: { type: String, default: '/pricing/' },
  /** ready=列表已成功拉取；unknown=未返回；error=拉取失败（fail-open 留当前页） */
  gitResourcesStatus: { type: String, default: 'ready' },
})

const menuOpen = ref(false)

const jumpable = computed(() =>
  (Array.isArray(props.gitResources) ? props.gitResources : []).filter((r) =>
    String(r?.gitlab_web_url || '').trim(),
  ),
)

const showVipBadge = computed(() => props.membershipTier === 'vip1')

const emptyNavHref = computed(() => {
  if (props.gitResourcesStatus === 'ready') {
    return props.pricingHref || '/pricing/'
  }
  return props.currentPageHref || '/'
})

function openMenu() {
  menuOpen.value = true
}

function closeMenu() {
  menuOpen.value = false
}

function toggleMenu() {
  menuOpen.value = !menuOpen.value
}
</script>
