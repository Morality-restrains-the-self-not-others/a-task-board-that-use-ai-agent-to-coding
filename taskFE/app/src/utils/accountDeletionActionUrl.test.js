// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] accountDeletionActionUrl.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { resolveAccountDeletionActionUrl } = await import('./accountDeletionActionUrl.js')

  describe('resolveAccountDeletionActionUrl', () => {
    it('把不存在的 gitlab-resources 页改写到 GitLab 设置页', () => {
      expect(resolveAccountDeletionActionUrl({
        action_url: '/tenant/877397588196749312/billing/gitlab-resources/',
        tenant_id: '877397588196749312',
        code: 'BILLING_GITLAB_RESOURCE_ACTIVE',
      })).toBe('/tenant/877397588196749312/settings/gitlab-connection/')
    })

    it('把 settings/members 改写到人员管理', () => {
      expect(resolveAccountDeletionActionUrl({
        action_url: '/tenant/t1/settings/members/',
        code: 'TENANT_SOLE_ADMIN',
      })).toBe('/tenant/t1/people/manage/')
    })

    it('把 /workspace/.../task/ 改写到 task-detail', () => {
      expect(resolveAccountDeletionActionUrl({
        action_url: '/tenant/t1/workspace/w1/task/task_1/',
        code: 'CLOUD_MACHINE_RUNNING',
      })).toBe('/tenant/t1/workspace/w1/task-detail/task_1/')
    })

    it('待支付且仅指向 /profile/ 时改到租户账单页', () => {
      expect(resolveAccountDeletionActionUrl({
        action_url: '/profile/',
        tenant_id: '877397588196749312',
        code: 'BILLING_PAYMENT_PENDING',
      })).toBe('/tenant/877397588196749312/billing/')
    })

    it('已正确的账单路径保持不变', () => {
      expect(resolveAccountDeletionActionUrl({
        action_url: '/tenant/t1/billing/orders/',
        code: 'BILLING_ORDER_PENDING',
      })).toBe('/tenant/t1/billing/orders/')
    })
  })
}
