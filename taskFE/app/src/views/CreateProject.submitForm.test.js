// @vitest-environment jsdom
/**
 * CreateProject.submitForm（OPT-20260807-021）：
 * - 仅凭 response.ok 判定成功会静默跳转（后端路由吞错时可能假 200 无实体）。
 * - 校验响应体存在 id 才跳转；假 200 时展示错误，不跳转。
 */
if (!process.env.VITEST) {
  console.log('[skip] CreateProject.submitForm.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    push: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' }, query: {} }),
    useRouter: () => ({ push: hoisted.push }),
  }))

  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const CreateProject = (await import('./CreateProject.vue')).default

  function jsonResponse(body, { ok = true, status = 201 } = {}) {
    return { ok, status, json: async () => body }
  }

  function stubDefaultApis() {
    hoisted.apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/installed-images/')) return jsonResponse([])
      if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
      return jsonResponse({})
    })
  }

  const mountView = async () => {
    const wrapper = mount(CreateProject, {
      global: {
        stubs: {
          ProjectTagsInput: { template: '<div class="stub-tags" />' },
          ProjectRunTemplatePanel: { template: '<div class="stub-run-template" />' },
        },
      },
    })
    await flushPromises()
    return wrapper
  }

  const fillRequiredFields = async (wrapper) => {
    await wrapper.find('#projectName').setValue('测试项目')
    await wrapper.find('#projectDescription').setValue('测试描述')
    await wrapper.find('#workspace').setValue('ws1')
  }

  describe('CreateProject.submitForm 创建成功判定', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.push.mockReset()
      sessionStorage.clear()
      stubDefaultApis()
    })

    it('后端返回 201 且响应含 id → 跳转项目列表', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        return jsonResponse({ id: 'proj_new', name: '测试项目' }, { ok: true, status: 201 })
      })

      const wrapper = await mountView()
      await fillRequiredFields(wrapper)
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()

      expect(hoisted.push).toHaveBeenCalledWith('/tenant/t1/projects/')
      expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    })

    it('假 200 无响应体 id → 不跳转，展示错误提示', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        // 假 200：ok=true 但无实体 id（路由吞错场景）
        return { ...jsonResponse({}, { ok: true, status: 200 }), traceId: 'trace-fake-200' }
      })

      const wrapper = await mountView()
      await fillRequiredFields(wrapper)
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()

      expect(hoisted.push).not.toHaveBeenCalled()
      const alert = wrapper.find('[role="alert"]')
      expect(alert.exists()).toBe(true)
      expect(alert.text()).toContain('创建项目失败')
      expect(alert.attributes('data-traceid')).toBe('trace-fake-200')
    })

    it('T2 提交创建 POST 携带 session grant_ticket', async () => {
      const repo = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
      sessionStorage.setItem(
        'gitOauthGrantTickets',
        JSON.stringify({
          'gitlab-tencent-sh-1.daydaymoney.com': 'tkt-create-1',
          '*': 'tkt-create-1',
        }),
      )
      let createBody
      hoisted.apiFetch.mockImplementation(async (url, opts) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        if (u.includes('/api/projects/tenant_id/')) {
          createBody = JSON.parse(String(opts?.body || '{}'))
          return jsonResponse({ id: 'proj_new', name: '测试项目' }, { ok: true, status: 201 })
        }
        return jsonResponse({
          results: [{ url: repo, is_accessible: true, token_status: 'token_available' }],
        })
      })

      const wrapper = await mountView()
      await fillRequiredFields(wrapper)
      await wrapper.find('#gitRepo0').setValue(repo)
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()

      expect(createBody?.grant_ticket).toBe('tkt-create-1')
      expect(createBody?.grant_tickets).toEqual(['tkt-create-1'])
      expect(createBody?.git_repos).toContain(repo)
    })
  })
}
