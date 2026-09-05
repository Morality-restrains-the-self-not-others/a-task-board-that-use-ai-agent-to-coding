// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] workspaceCloudPlatformsApi.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const {
    fetchWorkspaceCloudPlatforms,
    resetWorkspaceCloudPlatformsCacheForTests,
  } = await import('./workspaceCloudPlatformsApi.js')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('./apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  const TENANT_ID = '850256677331562496'
  const WORKSPACE_ID = '857903329669984256'

  describe('fetchWorkspaceCloudPlatforms', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      resetWorkspaceCloudPlatformsCacheForTests()
    })

    it('deduplicates concurrent requests for the same workspace', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(
        () => new Promise((resolve) => {
          setTimeout(() => {
            resolve({
              ok: true,
              json: async () => ({
                status: 'success',
                platforms: [{ id: 1, platform_type: 'aliyun' }],
              }),
            })
          }, 20)
        }),
      )

      const [first, second] = await Promise.all([
        fetchWorkspaceCloudPlatforms(TENANT_ID, WORKSPACE_ID),
        fetchWorkspaceCloudPlatforms(TENANT_ID, WORKSPACE_ID),
      ])

      expect(first).toEqual([{ id: 1, platform_type: 'aliyun' }])
      expect(second).toEqual(first)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledTimes(1)
    })

    it('returns cached result without another HTTP call within TTL', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          platforms: [{ id: 2, platform_type: 'aws' }],
        }),
      })

      await fetchWorkspaceCloudPlatforms(TENANT_ID, WORKSPACE_ID)
      const cached = await fetchWorkspaceCloudPlatforms(TENANT_ID, WORKSPACE_ID)

      expect(cached).toEqual([{ id: 2, platform_type: 'aws' }])
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledTimes(1)
    })

    it('returns empty list when tenant or workspace id is missing', async () => {
      await expect(fetchWorkspaceCloudPlatforms('', WORKSPACE_ID)).resolves.toEqual([])
      await expect(fetchWorkspaceCloudPlatforms(TENANT_ID, '')).resolves.toEqual([])
      expect(hoistedMocks.apiFetchMock).not.toHaveBeenCalled()
    })

    it('请求 taskCloudService 的 /api/cloud/ kv-last 端点（e105bad 迁移回归）', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'success', platforms: [] }),
      })
      await fetchWorkspaceCloudPlatforms(TENANT_ID, WORKSPACE_ID)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        `/api/cloud/platforms/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/`,
        expect.anything(),
      )
    })
  })
}
