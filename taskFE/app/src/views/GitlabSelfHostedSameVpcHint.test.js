// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] GitlabSelfHostedSameVpcHint.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const { default: GitlabSelfHostedSameVpcHint } = await import('./GitlabSelfHostedSameVpcHint.vue')

  const stubs = {
    CreateVpcModal: { template: '<div class="create-vpc-stub" />', props: ['visible', 'initialData'] },
    CreateVswitchModal: { template: '<div class="create-vswitch-stub" />', props: ['visible', 'initialData'] },
  }

  describe('GitlabSelfHostedSameVpcHint', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('does not render the hint when default machine configs are empty', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'success', data: [] }),
        traceId: '',
      })
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint"]').exists()).toBe(false)
    })

    it('shows VPC and vswitch IDs when a default machine is configured', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          data: [{
            authorization_id: 'auth-1',
            platform_type: 'aliyun',
            region: 'cn-hangzhou',
            vpc_id: 'vpc-abc',
            vswitch_id: 'vsw-xyz',
          }],
        }),
        traceId: '',
      })
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('专有网络')
      expect(wrapper.text()).toContain('交换机')
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint-vpc"]').text()).toContain('vpc-abc')
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint-vswitch"]').text()).toContain('vsw-xyz')
      expect(wrapper.find('[data-testid="gitlab-same-vpc-machine-settings-link"]').attributes('href'))
        .toBe('/tenant/123/settings/task-panel/')
    })

    it('disables create-vswitch when vpc is missing', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          data: [{
            authorization_id: 'auth-1',
            platform_type: 'aliyun',
            region: 'cn-hangzhou',
          }],
        }),
        traceId: '',
      })
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-create-vpc"]').element.disabled).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-same-vpc-create-vswitch"]').element.disabled).toBe(true)
    })

    it('attaches data-traceId when default-config GET fails', async () => {
      apiFetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ message: 'boom' }),
        traceId: 'trace-same-vpc-1',
      })
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint"]').exists()).toBe(false)
      const err = wrapper.find('[data-testid="gitlab-same-vpc-hint-error"]')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('trace-same-vpc-1')
    })

    it('enables create-vswitch after VPC created event supplies vpc_id', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          data: [{
            authorization_id: 'auth-1',
            platform_type: 'aliyun',
            region: 'cn-hangzhou',
          }],
        }),
        traceId: '',
      })
      const vpcStub = {
        props: ['visible', 'initialData'],
        emits: ['created', 'close'],
        template: '<button class="emit-vpc" type="button" @click="$emit(\'created\', { vpc_id: \'vpc-new\', name: \'net\', cidr_block: \'10.0.0.0/16\' })" />',
      }
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs: { ...stubs, CreateVpcModal: vpcStub } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-create-vswitch"]').element.disabled).toBe(true)
      await wrapper.find('.emit-vpc').trigger('click')
      expect(wrapper.find('[data-testid="gitlab-same-vpc-create-vswitch"]').element.disabled).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-same-vpc-hint-vpc"]').text()).toContain('vpc-new')
    })

    it('shows write-back only after VPC created and POSTs vpc_id onto existing default config (OPT-20260826-006)', async () => {
      apiFetchMock
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            status: 'success',
            data: [{
              authorization_id: 'auth-1',
              platform_type: 'aliyun',
              region: 'cn-hangzhou',
              zone_id: 'cn-hangzhou-b',
            }],
          }),
          traceId: '',
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'success' }),
          traceId: '',
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            status: 'success',
            data: [{
              authorization_id: 'auth-1',
              platform_type: 'aliyun',
              region: 'cn-hangzhou',
              zone_id: 'cn-hangzhou-b',
              vpc_id: 'vpc-new',
            }],
          }),
          traceId: '',
        })
      const vpcStub = {
        props: ['visible', 'initialData'],
        emits: ['created', 'close'],
        template: '<button class="emit-vpc" type="button" @click="$emit(\'created\', { vpc_id: \'vpc-new\', name: \'net\', cidr_block: \'10.0.0.0/16\' })" />',
      }
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs: { ...stubs, CreateVpcModal: vpcStub } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-same-vpc-writeback"]').exists()).toBe(false)
      await wrapper.find('.emit-vpc').trigger('click')
      const btn = wrapper.find('[data-testid="gitlab-same-vpc-writeback"]')
      expect(btn.exists()).toBe(true)
      await btn.trigger('click')
      await flushPromises()
      const post = apiFetchMock.mock.calls.find(([, opts]) => opts && opts.method === 'POST')
      expect(post).toBeTruthy()
      expect(String(post[0])).toContain('/api/cloud/server-config-default/tenant_id/123/')
      const body = JSON.parse(post[1].body)
      expect(body.vpc_id).toBe('vpc-new')
      expect(body.authorization_id).toBe('auth-1')
      expect(body.platform_type).toBe('aliyun')
      expect(wrapper.find('[data-testid="gitlab-same-vpc-writeback-message"]').text()).toContain('已写入')
    })

    it('write-back failure surfaces data-traceId (OPT-20260826-006)', async () => {
      apiFetchMock
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            status: 'success',
            data: [{
              authorization_id: 'auth-1',
              platform_type: 'aliyun',
              region: 'cn-hangzhou',
            }],
          }),
          traceId: '',
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 502,
          json: async () => ({ message: '写入失败' }),
          traceId: 'trace-writeback-1',
        })
      const vpcStub = {
        props: ['visible', 'initialData'],
        emits: ['created', 'close'],
        template: '<button class="emit-vpc" type="button" @click="$emit(\'created\', { vpc_id: \'vpc-new\', name: \'net\', cidr_block: \'10.0.0.0/16\' })" />',
      }
      const wrapper = mount(GitlabSelfHostedSameVpcHint, {
        props: { tenantId: '123' },
        global: { stubs: { ...stubs, CreateVpcModal: vpcStub } },
      })
      await flushPromises()
      await wrapper.find('.emit-vpc').trigger('click')
      await wrapper.find('[data-testid="gitlab-same-vpc-writeback"]').trigger('click')
      await flushPromises()
      const err = wrapper.find('[data-testid="gitlab-same-vpc-hint-error"]')
      expect(err.exists()).toBe(true)
      expect(err.text()).toContain('写入失败')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('trace-writeback-1')
    })
  })
}
