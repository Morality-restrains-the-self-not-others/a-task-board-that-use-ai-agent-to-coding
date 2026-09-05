// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] GitlabSyncProjectsModal.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: GitlabSyncProjectsModal } = await import('./GitlabSyncProjectsModal.vue')

  describe('GitlabSyncProjectsModal', () => {
    it('错误 alert 挂载 data-traceId', () => {
      const wrapper = mount(GitlabSyncProjectsModal, {
        props: {
          show: true,
          loading: false,
          creating: false,
          error: '网络错误，无法启动 GitLab 授权',
          errorTraceId: 'tid-gitlab-sync-oauth-1',
          oauthBound: false,
          gitlabWebsite: '',
          gitlabLogin: '',
          repos: [],
          selectedCount: 0,
          batchEligibleCount: 0,
          allSelectableChecked: false,
          selectedRepoKeys: {},
          workspaces: [],
          selectedWorkspaceId: '',
          createResult: null,
          combinedProjectName: '',
        },
      })
      const alert = wrapper.get('[role="alert"]')
      expect(alert.text()).toContain('网络错误，无法启动 GitLab 授权')
      expect(alert.attributes('data-traceid') || alert.attributes('data-traceId')).toBe(
        'tid-gitlab-sync-oauth-1',
      )
    })

    it('无 errorTraceId 时不挂 data-traceId', () => {
      const wrapper = mount(GitlabSyncProjectsModal, {
        props: {
          show: true,
          error: '请选择工作空间',
          errorTraceId: '',
          oauthBound: true,
          repos: [],
          selectedCount: 0,
          batchEligibleCount: 0,
          allSelectableChecked: false,
          selectedRepoKeys: {},
          workspaces: [],
          selectedWorkspaceId: '',
          createResult: null,
          combinedProjectName: '',
        },
      })
      const alert = wrapper.get('[role="alert"]')
      expect(alert.attributes('data-traceid') || alert.attributes('data-traceId')).toBeUndefined()
    })
    it('无已购区域时展示开通引导', () => {
      const wrapper = mount(GitlabSyncProjectsModal, {
        props: {
          show: true,
          loading: false,
          regionsLoading: false,
          purchasedRegions: [],
          selectedRegionSlug: '',
          oauthBound: false,
          repos: [],
          selectedCount: 0,
          batchEligibleCount: 0,
          allSelectableChecked: false,
          selectedRepoKeys: {},
          workspaces: [],
          selectedWorkspaceId: '',
          createResult: null,
          combinedProjectName: '',
        },
      })
      expect(wrapper.get('[data-testid="gitlab-sync-no-purchased-region"]').text()).toContain(
        '尚未开通可用的 GitLab 区域',
      )
      expect(wrapper.text()).not.toContain('gitlab.daydaymoney.com')
    })

    it('多区域时渲染来源下拉并 emit select-region', async () => {
      const wrapper = mount(GitlabSyncProjectsModal, {
        props: {
          show: true,
          purchasedRegions: [
            {
              region: 'a',
              region_name: 'A区',
              gitlab_web_url: 'https://a.example',
              provisioning_status: 'active',
            },
            {
              region: 'b',
              region_name: 'B区',
              gitlab_web_url: 'https://b.example',
              provisioning_status: 'active',
            },
          ],
          selectedRegionSlug: 'a',
          gitlabWebsite: 'https://a.example',
          oauthBound: true,
          repos: [],
          selectedCount: 0,
          batchEligibleCount: 0,
          allSelectableChecked: false,
          selectedRepoKeys: {},
          workspaces: [],
          selectedWorkspaceId: '',
          createResult: null,
          combinedProjectName: '',
        },
      })
      const select = wrapper.get('[data-testid="gitlab-sync-region-select"]')
      await select.setValue('b')
      expect(wrapper.emitted('select-region')?.[0]).toEqual(['b'])
    })
  })
}
