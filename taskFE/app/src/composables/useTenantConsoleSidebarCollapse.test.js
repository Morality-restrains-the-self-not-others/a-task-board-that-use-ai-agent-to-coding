// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'
import {
  TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY,
  readSidebarCollapsed,
  writeSidebarCollapsed,
  useTenantConsoleSidebarCollapse,
} from './useTenantConsoleSidebarCollapse.js'

describe('useTenantConsoleSidebarCollapse', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('readSidebarCollapsed：无 key 时为 false，值为 1 时为 true', () => {
    expect(readSidebarCollapsed()).toBe(false)
    localStorage.setItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY, '1')
    expect(readSidebarCollapsed()).toBe(true)
    localStorage.setItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY, '0')
    expect(readSidebarCollapsed()).toBe(false)
  })

  it('writeSidebarCollapsed：写入 1/0', () => {
    writeSidebarCollapsed(true)
    expect(localStorage.getItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY)).toBe('1')
    writeSidebarCollapsed(false)
    expect(localStorage.getItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY)).toBe('0')
  })

  it('toggleCollapsed / expandSidebar 更新状态并持久化', async () => {
    localStorage.setItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY, '0')
    const { collapsed, toggleCollapsed, expandSidebar } = useTenantConsoleSidebarCollapse()
    expect(collapsed.value).toBe(false)

    toggleCollapsed()
    await nextTick()
    expect(collapsed.value).toBe(true)
    expect(localStorage.getItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY)).toBe('1')

    expandSidebar()
    await nextTick()
    expect(collapsed.value).toBe(false)
    expect(localStorage.getItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY)).toBe('0')
  })

  // 状态为模块级单例（OPT-20260809-024 联动改造）：localStorage 在模块求值（= 页面加载/
  // 首次组件挂载）时读取一次，Sidebar 与 Navbar 共享同一份状态。用 resetModules 模拟
  // 全新模块实例（页面刷新）验证持久化恢复路径。
  it('模块加载（页面刷新）时从 localStorage 恢复缩窄态', async () => {
    localStorage.setItem(TENANT_CONSOLE_SIDEBAR_COLLAPSED_KEY, '1')
    vi.resetModules()
    const fresh = await import('./useTenantConsoleSidebarCollapse.js')
    const { collapsed } = fresh.useTenantConsoleSidebarCollapse()
    expect(collapsed.value).toBe(true)
  })

  it('localStorage 抛错时读写不抛', () => {
    const getSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    const setSpy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(readSidebarCollapsed()).toBe(false)
    expect(() => writeSidebarCollapsed(true)).not.toThrow()
    getSpy.mockRestore()
    setSpy.mockRestore()
  })
})
