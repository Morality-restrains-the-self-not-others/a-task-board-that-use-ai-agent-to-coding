// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsGitlabConnection.test.js requires vitest runtime')
} else {
  const { computed, nextTick, ref } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: WorkspaceSettingsGitlabConnection } = await import('./WorkspaceSettingsGitlabConnection.vue')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ configured: false }),
      traceId: '',
    })),
  }))

  vi.mock('../composables/useGitlabResourcePurchase.js', () => ({
    useGitlabResourcePurchase: vi.fn(),
    DISK_MONTH_OPTIONS: [1, 3, 6],
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({
      params: { tenant: '123' },
      fullPath: '/tenant/123/settings/gitlab',
    }),
    useRouter: () => ({
      push: vi.fn(),
      replace: vi.fn(),
    }),
  }))

  const { useGitlabResourcePurchase } = await import('../composables/useGitlabResourcePurchase.js')

  const stubs = {
    GitlabOAuthAppStatus: { template: '<div class="gitlab-oauth-stub" />', props: ['tenantId', 'gitlabWebUrl'] },
    CreateGitlabOAuthApp: { template: '<div class="create-oauth-stub" />', props: ['tenantId', 'gitlabWebUrl'] },
    GitlabConnectionStatus: { template: '<div class="connection-status-stub" />', props: ['tenantId', 'gitlabWebUrl'] },
    GitlabProjectSyncStatus: { template: '<div class="project-sync-stub" />', props: ['tenantId'] },
    WorkspaceSettingsGitlabOauthAppHelp: { template: '<div class="oauth-help-stub" />', props: ['redirectUri'] },
    WorkspaceSettingsGitlabOidcSso: { template: '<div class="oidc-sso-stub" />', props: ['tenantId', 'pathABaseUrl'] },
    GitlabSelfHostedSameVpcHint: { template: '<div class="same-vpc-hint-stub" />', props: ['tenantId'] },
    GitlabSelfHostedReachability: {
      template: '<div class="reachability-stub" />',
      props: ['tenantId', 'configured', 'intranet'],
    },
  }

  function mockGitlabComposable(overrides = {}) {
    const defaults = {
      loading: ref(false),
      resLoading: ref(false),
      purchasing: ref(false),
      errorMessage: ref(''),
      resErrorMessage: ref(''),
      errorTraceId: ref(''),
      resErrorTraceId: ref(''),
      successMessage: ref(''),
      resSuccessMessage: ref(''),
      provisioningStatus: ref('not_purchased'),
      currentDiskGb: ref(0),
      currentDiskMonths: ref(0),
      currentTrafficGb: ref(0),
      currentDiskExpiresAt: ref(''),
      diskUsedGb: ref(0),
      trafficUsedGb: ref(0),
      diskUnitPrice: ref(0),
      trafficUnitPrice: ref(0),
      region: ref(''),
      regionName: ref(''),
      availableRegions: ref([]),
      gitlabWebUrl: ref(''),
      balanceYuanCents: ref(0),
      form: { disk_gb: 0, disk_months: 0, traffic_prepaid_gb: 0, custom_months: 0 },
      estimatedCost: computed(() => 0),
      diskMonthOptions: [1, 3, 6],
      customMonths: ref(0),
      diskMonthsCustomLabel: '自定义',
      load: vi.fn(),
      purchase: vi.fn(),
      ...overrides,
    }
    useGitlabResourcePurchase.mockReturnValue(defaults)
    return defaults
  }

  describe('WorkspaceSettingsGitlabConnection — GitLab resource visibility', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('renders available GitLab regions in the picker', () => {
      mockGitlabComposable({
        provisioningStatus: ref('not_purchased'),
        availableRegions: ref([{ slug: 'tencent-sh-1', name: '腾讯上海一区' }]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const select = wrapper.find('[data-testid="gitlab-region-select"]')
      expect(select.exists()).toBe(true)
      const values = select.findAll('option').map((o) => o.element.value)
      expect(values).toContain('tencent-sh-1')
    })

    it('OPT-20260902-013: groups region options by cloud provider with 腾讯云/阿里云 optgroup', () => {
      mockGitlabComposable({
        provisioningStatus: ref('not_purchased'),
        availableRegions: ref([
          { slug: 'aliyun-pending', name: '阿里华东（待开通）', cloud_provider: 'aliyun', infra_status: 'pending_node' },
          { slug: 'tencent-sh-1', name: '腾讯上海一区', cloud_provider: 'tencent' },
          { slug: 'tencent-sh-2', name: '腾讯上海二区', cloud_provider: 'tencent' },
        ]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const select = wrapper.find('[data-testid="gitlab-region-select"]')
      const groups = select.findAll('optgroup').map((g) => ({ label: g.attributes('label'), count: g.findAll('option').length }))
      expect(groups).toEqual([
        { label: '腾讯云', count: 2 },
        { label: '阿里云（人工开通节点）', count: 1 },
      ])
      expect(select.findAll('option').map((o) => o.element.value)).toContain('aliyun-pending')
    })

    it('OPT-20260902-013: selecting a pending_node region shows the manual-provisioning hint', async () => {
      const region = ref('aliyun-pending')
      mockGitlabComposable({
        provisioningStatus: ref('not_purchased'),
        region,
        availableRegions: ref([
          { slug: 'aliyun-pending', name: '阿里华东（待开通）', cloud_provider: 'aliyun', infra_status: 'pending_node' },
        ]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      await nextTick()
      const hint = wrapper.find('[data-testid="gitlab-region-pending-node-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('尚未部署')
    })

    it('hides built-in GitLab section when provisioningStatus is not_purchased', () => {
      mockGitlabComposable({ provisioningStatus: ref('not_purchased') })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-builtin-resources"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-region-picker"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('购买 GitLab 资源')
      const buy = wrapper.find('[data-testid="gitlab-purchase-link"]')
      expect(buy.exists()).toBe(true)
      expect(buy.attributes('href')).toContain('/tenant/123/billing/orders/create/')
    })

    it('keeps region picker when built-in GitLab is active and places details after the select', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentDiskGb: ref(10),
        region: ref('tencent-sh-1'),
        regionName: ref('腾讯上海一区'),
        availableRegions: ref([
          { slug: 'tencent-shanghai-5', name: '腾讯上海五区', gitlab_web_url: 'https://gl5.example' },
          { slug: 'tencent-sh-1', name: '腾讯上海一区', gitlab_web_url: 'https://gl1.example' },
        ]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const picker = wrapper.find('[data-testid="gitlab-region-picker"]')
      const details = wrapper.find('[data-testid="gitlab-builtin-resources"]')
      expect(picker.exists()).toBe(true)
      expect(details.exists()).toBe(true)
      expect(picker.find('[data-testid="gitlab-region-select"]').exists()).toBe(true)
      expect(picker.find('[data-testid="gitlab-builtin-resources"]').exists()).toBe(true)
      const html = picker.element.innerHTML
      expect(html.indexOf('data-testid="gitlab-region-select"')).toBeGreaterThanOrEqual(0)
      expect(html.indexOf('data-testid="gitlab-builtin-resources"')).toBeGreaterThan(
        html.indexOf('data-testid="gitlab-region-select"')
      )
      expect(wrapper.find('[data-testid="gitlab-region-name"]').text()).toBe('腾讯上海一区')
      expect(wrapper.text()).toContain('10 GB')
    })

    it('shows pending message under the picker when provisioningStatus is pending_admin', () => {
      mockGitlabComposable({
        provisioningStatus: ref('pending_admin'),
        region: ref('tencent-sh-1'),
        availableRegions: ref([{ slug: 'tencent-sh-1', name: '腾讯上海一区' }]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-region-picker"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-builtin-resources"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-provisioning-pending"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-provisioning-pending"]').text()).toContain('等待管理员开通实施')
    })

    it('updates GitLab details when switching region option without replacing the picker', async () => {
      const region = ref('tencent-shanghai-5')
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        region,
        regionName: ref('腾讯上海五区'),
        gitlabWebUrl: ref('https://gl5.example'),
        availableRegions: ref([
          { slug: 'tencent-shanghai-5', name: '腾讯上海五区', gitlab_web_url: 'https://gl5.example' },
          { slug: 'tencent-sh-1', name: '腾讯上海一区', gitlab_web_url: 'https://gl1.example' },
        ]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-region-name"]').text()).toBe('腾讯上海五区')
      await wrapper.find('[data-testid="gitlab-region-select"]').setValue('tencent-sh-1')
      await nextTick()
      expect(wrapper.find('[data-testid="gitlab-region-picker"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-builtin-resources"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-region-select"]').element.value).toBe('tencent-sh-1')
      expect(wrapper.find('[data-testid="gitlab-region-name"]').text()).toBe('腾讯上海一区')
      const web = wrapper.find('[data-testid="gitlab-web-url"]')
      expect(web.exists()).toBe(true)
      expect(web.attributes('href')).toBe('https://gl1.example')
    })

    it('shows details under picker for a selected region even when not purchased', () => {
      mockGitlabComposable({
        provisioningStatus: ref('not_purchased'),
        region: ref('tencent-sh-1'),
        availableRegions: ref([{ slug: 'tencent-sh-1', name: '腾讯上海一区' }]),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-region-picker"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-builtin-resources"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-region-name"]').text()).toBe('腾讯上海一区')
      expect(wrapper.find('[data-testid="gitlab-current-disk-gb"]').text()).toBe('未购买')
    })

    it('shows disk and traffic unit prices without 锁价 in resource quota labels', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        diskUnitPrice: ref(500),
        trafficUnitPrice: ref(100),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const section = wrapper.find('[data-testid="gitlab-builtin-resources"]')
      expect(section.exists()).toBe(true)
      const dtTexts = section.findAll('dt').map((dt) => dt.text())
    expect(dtTexts).toContain('磁盘单价')
    expect(dtTexts).toContain('流量单价')
    expect(dtTexts).not.toContain('账户余额')
    expect(dtTexts).not.toContain('资源配额')
    expect(section.find('[data-testid="gitlab-balance-points"]').exists()).toBe(false)
    expect(dtTexts.some((t) => t.includes('锁价'))).toBe(false)
    expect(section.text()).not.toMatch(/[（(]锁价[）)]/)
  })

  it('does not show spendable cash wallet on GitLab resource panel', () => {
    mockGitlabComposable({
      provisioningStatus: ref('active'),
      balanceYuanCents: ref(12345),
      currentDiskGb: ref(10),
      currentTrafficGb: ref(20),
    })
    const wrapper = mount(WorkspaceSettingsGitlabConnection, {
      global: { stubs },
    })
    const section = wrapper.find('[data-testid="gitlab-builtin-resources"]')
    const dtTexts = section.findAll('dt').map((dt) => dt.text())
    expect(dtTexts).toContain('当前磁盘配额')
    expect(dtTexts).toContain('当前流量预购')
    expect(dtTexts).not.toContain('账户余额')
    expect(section.find('[data-testid="gitlab-balance-points"]').exists()).toBe(false)
    expect(section.text()).not.toContain('123.45 元')
  })

    it('shows distinct disk used and traffic used when API values differ', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentDiskGb: ref(1),
        currentTrafficGb: ref(1),
        diskUsedGb: ref(0.01049),
        trafficUsedGb: ref(0),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const disk = wrapper.find('[data-testid="gitlab-disk-used-gb"]')
      const traffic = wrapper.find('[data-testid="gitlab-traffic-used-gb"]')
      expect(disk.text()).toContain('0.01049 GB')
      expect(traffic.text()).toContain('0 GB')
      expect(traffic.text()).not.toContain('0.01049')
    })

    it('shows unpurchased traffic quota when currentTrafficGb is 0', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentTrafficGb: ref(0),
        trafficUsedGb: ref(0.01049),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const used = wrapper.find('[data-testid="gitlab-traffic-used-gb"]')
      expect(used.exists()).toBe(true)
      expect(used.text()).toContain('0.01049 GB')
      expect(used.text()).toContain('未预购')
      expect(used.text()).not.toMatch(/\/ 0 GB/)
      const blocked = wrapper.find('[data-testid="gitlab-traffic-quota-blocked"]')
      expect(blocked.exists()).toBe(true)
      expect(blocked.text()).toContain('任务')
      expect(blocked.text()).toContain('同区域内网')
      expect(blocked.text()).toContain('CI')
      const buy = wrapper.find('[data-testid="gitlab-traffic-quota-purchase-link"]')
      expect(buy.exists()).toBe(true)
      expect(buy.element.tagName).toBe('A')
      expect(buy.attributes('href')).toContain('/tenant/123/billing/orders/create/')
    })

    it('shows 未预购 for traffic when currentTrafficGb is 0', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentTrafficGb: ref(0),
        trafficUsedGb: ref(0.01049),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const used = wrapper.find('[data-testid="gitlab-traffic-used-gb"]')
      expect(used.exists()).toBe(true)
      expect(used.text()).toContain('0.01049 GB')
      expect(used.text()).toContain('未预购')
      expect(used.text()).not.toMatch(/\/ 0 GB/)
    })

    it('hides traffic block banner when prepaid remains', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentTrafficGb: ref(5),
        trafficUsedGb: ref(0.01),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-traffic-quota-blocked"]').exists()).toBe(false)
    })

    it('shows 未购买 for disk when currentDiskGb is 0', () => {
      mockGitlabComposable({
        provisioningStatus: ref('active'),
        currentDiskGb: ref(0),
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      expect(wrapper.find('[data-testid="gitlab-current-disk-gb"]').text()).toBe('未购买')
    })

    it('loads connection via canonical key/value path tenant_id/{tid}/', async () => {
      mockGitlabComposable({ provisioningStatus: ref('not_purchased') })
      apiFetchMock.mockClear()
      mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      await vi.waitFor(() => {
        expect(apiFetchMock).toHaveBeenCalled()
      })
      const calledPath = String(apiFetchMock.mock.calls[0][0])
      expect(calledPath).toBe('/api/git-oauth/tenant-connection/tenant_id/123/')
      expect(calledPath).not.toContain('/gitlab-oauth-connection/')
      expect(calledPath).not.toMatch(/^\/api\/tenant\//)
    })

    it('displays tenant-scoped redirect_uri from connection API', async () => {
      mockGitlabComposable({ provisioningStatus: ref('not_purchased') })
      const tenantRedirect =
        'https://daydaymoney.com/api/accounts/tenant-123/oauth/callback/'
      apiFetchMock.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          configured: false,
          company_id: '123',
          redirect_uri: tenantRedirect,
          provider_key: 'gitlab:tenant-123',
        }),
        traceId: '',
      })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-redirect-uri"]').exists()).toBe(true)
      })
      const input = wrapper.find('[data-testid="gitlab-redirect-uri"]')
      expect(input.element.value).toBe(tenantRedirect)
      expect(input.element.value).toContain('tenant-123')
      expect(input.element.value).not.toContain('tenant-gitlab')
    })

    it('places the same-VPC hint inside the self-hosted GitLab section', () => {
      mockGitlabComposable({ provisioningStatus: ref('not_purchased') })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      const section = wrapper.find('[data-testid="gitlab-self-hosted-connection"]')
      expect(section.exists()).toBe(true)
      expect(section.find('.same-vpc-hint-stub').exists()).toBe(true)
    })

    it('places reachability controls inside the self-hosted GitLab section', async () => {
      mockGitlabComposable({ provisioningStatus: ref('not_purchased') })
      const wrapper = mount(WorkspaceSettingsGitlabConnection, {
        global: { stubs },
      })
      await vi.waitFor(() => {
        const section = wrapper.find('[data-testid="gitlab-self-hosted-connection"]')
        expect(section.find('.reachability-stub').exists()).toBe(true)
      })
    })
  })
}
