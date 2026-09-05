// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailGithubPrCredentialPanel.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: () => {},
  }))

  const baseProps = {
    tenantId: 't1',
    workspaceId: 'w1',
    taskId: '42',
  }

  beforeEach(() => {
    hoistedMocks.apiFetchMock.mockReset()
  })

  describe('TaskDetailGithubPrCredentialPanel', () => {
    it('fetches status with funcName-first path on mount (missing-slash regression)', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          approved_for_task: false,
          github_app_connected: true,
          github_login: 'ruandao',
          github_connections: [
            { connected: true, github_user_id: '1321779', github_login: 'ruandao' },
          ],
          repo_bindings: [
            {
              repo_url: 'https://github.com/ruandao/somanyad',
              repo_slug: 'ruandao/somanyad',
              selected_github_user_id: null,
              selected_github_login: null,
            },
          ],
          all_repo_bound: false,
        }),
      })

      const component = await import('./TaskDetailGithubPrCredentialPanel.vue')
      const wrapper = mount(component.default, {
        props: baseProps,
        global: {
          stubs: {
            'router-link': { template: '<a><slot /></a>' },
          },
        },
      })

      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledTimes(1)
      // 回归：原 bug 拼成 task_id/${taskId}github-credential-status/ 缺斜杠。
      expect(String(hoistedMocks.apiFetchMock.mock.calls[0][0])).toBe(
        '/api/cloud/compute/github-credential-status/tenant_id/t1/workspace_id/w1/task_id/42/',
      )

      await vi.waitFor(() => {
        expect(wrapper.text()).toContain('已连接')
      })
      expect(wrapper.text()).toContain('@ruandao')
    })
  })
}
