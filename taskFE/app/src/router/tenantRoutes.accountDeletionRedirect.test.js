// @vitest-environment jsdom
// 注销阻断曾链到未注册路径，catch-all 会踢回首页；别名须落到真实页。
if (!process.env.VITEST) {
  console.log('[skip] tenantRoutes.accountDeletionRedirect.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { publicRoutes } = await import('./publicRoutes.js')
  const { tenantRoutes } = await import('./tenantRoutes.js')
  const { adminRoutes } = await import('./adminRoutes.js')

  function makeRouter() {
    return createRouter({
      history: createMemoryHistory(),
      routes: [...publicRoutes, ...tenantRoutes, ...adminRoutes],
    })
  }

  describe('注销阻断 action_url 路由别名', () => {
    it('/billing/gitlab-resources/ 重定向到 GitLab 设置页，而不是 home', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/billing/gitlab-resources/')
      expect(router.currentRoute.value.name).toBe('tenant_settings_gitlab_connection')
      expect(router.currentRoute.value.name).not.toBe('home')
    })

    it('/settings/members/ 重定向到人员管理', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/settings/members/')
      expect(router.currentRoute.value.name).toBe('people_manage')
    })

    it('/workspace/:ws/task/:task/ 重定向到 task-detail', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/workspace/w1/task/task_1/')
      expect(router.currentRoute.value.name).toBe('task_detail')
      expect(router.currentRoute.value.params.taskId).toBe('task_1')
    })
  })
}
