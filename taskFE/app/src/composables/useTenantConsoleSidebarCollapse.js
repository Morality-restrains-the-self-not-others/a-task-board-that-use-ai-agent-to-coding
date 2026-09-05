import { ref, watch } from 'vue'

/** localStorage key：租户控制台侧栏是否缩窄（仅图标） */
export const TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY = 'tenant-console-sidebar-collapsed'

/**
 * @returns {boolean}
 */
export function readSidebarCollapsed() {
  try {
    return localStorage.getItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

/**
 * @param {boolean} collapsed
 */
export function writeSidebarCollapsed(collapsed) {
  try {
    localStorage.setItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // private mode / quota — ignore
  }
}

// ── 模块级单例状态 ──────────────────────────────────────────────
// 侧栏收起状态由 Sidebar（点击「控制台导航」）与 Navbar（顶部导航联动收起）
// 共享同一份响应式 ref，保证任意一侧切换另一侧即时联动。
// localStorage 读写只在此处执行一次，各调用方只读该单例。
const collapsed = ref(readSidebarCollapsed())

watch(collapsed, (value) => {
  writeSidebarCollapsed(value)
})

/**
 * 租户控制台侧栏缩窄/展开状态（模块级单例），并持久化到 localStorage。
 * Sidebar 与 Navbar 调用返回同一份状态，实现「点击控制台导航 ↔ 顶部导航栏
 * 收起/展开」联动。
 */
export function useTenantConsoleSidebarCollapse() {
  const toggleCollapsed = () => {
    collapsed.value = !collapsed.value
  }

  /** 缩窄态点击带子菜单的入口时先展开侧栏 */
  const expandSidebar = () => {
    collapsed.value = false
  }

  return { collapsed, toggleCollapsed, expandSidebar }
}
