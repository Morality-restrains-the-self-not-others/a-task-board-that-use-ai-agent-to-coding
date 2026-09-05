/**
 * useTenantPageAccess — 租户范围页面的 page:* 访问判定（v72）。
 *
 * 侧栏隐藏 page 不是安全边界（后端 PDP 仍会 403）；直链进入无权限页面时
 * 应展示 TenantPageAccessEmpty 空态而非继续渲染，避免误导（OPT-20260811-034）。
 *
 * fail-open 语义：权限快照尚未加载（perms.loaded=false，如直链首帧/单测环境）时
 * 视为可访问，避免首帧误闪空态与破坏既有页面单测；快照加载后按真实 page:* 判定。
 * perms.load 由应用壳（Sidebar）在挂载时触发。
 *
 * 用法（settings/billing 等租户范围视图）：
 *   const { accessAllowed } = useTenantPageAccess('settings.cloud')
 *   模板: <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.cloud" />
 *         <div v-else>…</div>
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { usePermissions } from './usePermissions.js'

export function useTenantPageAccess(pageKey) {
  const route = useRoute()
  const perms = usePermissions()
  const companyId = computed(() => String(route.params.tenant || ''))

  const accessAllowed = computed(() => {
    const cid = companyId.value
    if (!cid) return false
    if (!perms.loaded.value) return true // fail-open until permission snapshot loads
    return perms.hasPage(cid, pageKey)
  })

  return { companyId, accessAllowed }
}
