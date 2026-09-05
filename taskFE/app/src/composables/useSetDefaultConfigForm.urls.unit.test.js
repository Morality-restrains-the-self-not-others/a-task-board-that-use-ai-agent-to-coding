// @vitest-environment jsdom
/**
 * OPT-20260807-031 回归测试：设置默认配置表单的云资源请求全部使用 kv 形式
 * /api/cloud/{family}/{sub}/tenant_id/{tid}/，不再调用旧式 /api/tenant/{tid}/cloud/*。
 * 旧路径指向已停用的 django-cloud-tenant-config（gateway 已注释）→ 404。
 *
 * 运行时：通过公开 handleRegionChange / handleVpcChange / loadInstances / handleSubmit
 * 触发 server-images（vpcs/vswitches/security-groups）与 cloud-platform available-instances、
 * server-config-default 写入的 URL 断言；
 * 静态：对两个源文件做字面量扫描，确保不再残留 /api/tenant/ 旧路径。
 */
if (!process.env.VITEST) {
  console.log('[skip] useSetDefaultConfigForm.urls.unit.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { join } = await import('node:path')
  const { apiFetch } = await import('../utils/apiUtils.js')
  const { default: modalService } = await import('../utils/modalService.js')

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/modalService.js', () => ({
    __esModule: true,
    default: { alert: vi.fn() },
  }))

  const { useSetDefaultConfigForm } = await import('./useSetDefaultConfigForm.js')

  const TID = '850256677331562496'
  const originalLocation = window.location

  function setupForm(overrides = {}) {
    return useSetDefaultConfigForm(
      {
        visible: true,
        initialData: { authorization_id: 'auth-123', platform_type: 'aliyun' },
        ...overrides,
      },
      vi.fn()
    )
  }

  function calledUrls() {
    return apiFetch.mock.calls.map((c) => String(c[0]))
  }

  function hasUrlContaining(urls, prefix) {
    return urls.some((u) => u.startsWith(prefix))
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'location', {
      value: { pathname: `/tenant/${TID}/settings/task-panel/` },
      writable: true,
      configurable: true,
    })
    apiFetch.mockResolvedValue({ ok: true, json: async () => [] })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
  })

  describe('server-images 子资源 URL（kv 形式 /api/cloud/server-images/{sub}/tenant_id/{tid}/）', () => {
    it('handleRegionChange 触发 loadVpcs → vpcs 子路由', async () => {
      const f = setupForm()
      f.formData.value.platform_type = 'aliyun'
      f.formData.value.authorization_id = 'auth-123'
      f.formData.value.region = 'cn-hangzhou'
      await f.handleRegionChange()

      expect(calledUrls()).toContain(
        `/api/cloud/server-images/vpcs/tenant_id/${TID}/?region_id=cn-hangzhou&authorization_id=auth-123`
      )
    })

    it('handleVpcChange 触发 vswitches / security-groups / available-instances 子路由', async () => {
      const f = setupForm()
      f.formData.value.platform_type = 'aliyun'
      f.formData.value.authorization_id = 'auth-123'
      f.formData.value.region = 'cn-hangzhou'
      f.formData.value.vpc_id = 'vpc-1'
      await f.handleVpcChange()

      const urls = calledUrls()
      expect(urls).toContain(
        `/api/cloud/server-images/vswitches/tenant_id/${TID}/?region_id=cn-hangzhou&vpc_id=vpc-1&authorization_id=auth-123`
      )
      expect(urls).toContain(
        `/api/cloud/server-images/security-groups/tenant_id/${TID}/?region_id=cn-hangzhou&vpc_id=vpc-1&authorization_id=auth-123`
      )
      // available-instances 会追加默认筛选参数，仅断言 kv 前缀
      expect(hasUrlContaining(urls, `/api/cloud/cloud-platform/auth-123/available-instances/tenant_id/${TID}/?platform_type=aliyun&region_id=cn-hangzhou`)).toBe(true)
    })

    it('loadInstances 使用 available-instances 子路由', async () => {
      const f = setupForm()
      f.formData.value.platform_type = 'aliyun'
      f.formData.value.authorization_id = 'auth-123'
      f.formData.value.region = 'cn-hangzhou'
      await f.loadInstances()

      // available-instances 会追加默认筛选参数，仅断言 kv 前缀
      expect(hasUrlContaining(calledUrls(), `/api/cloud/cloud-platform/auth-123/available-instances/tenant_id/${TID}/?platform_type=aliyun&region_id=cn-hangzhou`)).toBe(true)
    })

    it('handleSubmit 使用列表写入端点 POST /server-config-default/tenant_id/{tid}/', async () => {
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) })
      const f = setupForm()
      await f.handleSubmit()

      const postCall = apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(String(postCall[0])).toBe(`/api/cloud/server-config-default/tenant_id/${TID}/`)
    })
  })

  describe('静态源扫描：两文件不再残留 /api/tenant/ 旧路径', () => {
    const srcRoot = join(process.cwd(), 'src')
    const composableSrc = readFileSync(join(srcRoot, 'composables/useSetDefaultConfigForm.js'), 'utf8')
    const modalSrc = readFileSync(join(srcRoot, 'components/cloud/CreateVswitchModal.vue'), 'utf8')

    it('useSetDefaultConfigForm.js 不含 /api/tenant/ 字面量，且含 kv 形式关键端点', () => {
      expect(composableSrc).not.toMatch(/\/api\/tenant\//)
      expect(composableSrc).toContain('/api/cloud/server-images/vpcs/tenant_id/')
      expect(composableSrc).toContain('/api/cloud/server-images/vswitches/tenant_id/')
      expect(composableSrc).toContain('/api/cloud/server-images/security-groups/tenant_id/')
      expect(composableSrc).toContain('/api/cloud/cloud-platform/')
      expect(composableSrc).toContain('/api/cloud/server-config-default/tenant_id/')
      expect(composableSrc).toContain('/api/cloud/installed-images/')
    })

    it('CreateVswitchModal.vue 不含 /api/tenant/ 字面量，且含 kv 形式关键端点', () => {
      expect(modalSrc).not.toMatch(/\/api\/tenant\//)
      expect(modalSrc).toContain('/api/cloud/server-images/vswitches/tenant_id/')
      expect(modalSrc).toContain('/api/cloud/cloud-platform/')
      expect(modalSrc).toContain('/api/cloud/server-images/create-vswitch/tenant_id/')
      expect(modalSrc).toContain('/api/cloud/server-images/update-vswitch/tenant_id/')
    })
  })
}
