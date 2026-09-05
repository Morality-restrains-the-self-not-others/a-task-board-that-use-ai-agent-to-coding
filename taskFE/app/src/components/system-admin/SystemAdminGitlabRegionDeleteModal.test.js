// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabRegionDeleteModal.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { resolveGitlabRegionRepoAddress } = await import('./gitlabRegionRepoAddress.js')
  const Modal = (await import('./SystemAdminGitlabRegionDeleteModal.vue')).default

  describe('resolveGitlabRegionRepoAddress', () => {
    it('优先 gitlab_web_url', () => {
      expect(
        resolveGitlabRegionRepoAddress({
          gitlab_web_url: 'https://git.example.com',
          gitlab_api_base: 'http://127.0.0.1:8012',
        })
      ).toBe('https://git.example.com')
    })

    it('无 web 时回退 gitlab_api_base', () => {
      expect(
        resolveGitlabRegionRepoAddress({
          gitlab_web_url: '',
          gitlab_api_base: 'http://127.0.0.1:8012',
        })
      ).toBe('http://127.0.0.1:8012')
    })

    it('均未配置时返回占位文案', () => {
      expect(resolveGitlabRegionRepoAddress({})).toBe('（未配置仓库地址）')
    })
  })

  describe('SystemAdminGitlabRegionDeleteModal', () => {
    const region = {
      name: '腾讯上海一区',
      slug: 'tencent-sh-1',
      gitlab_web_url: 'https://git.daydaymoney.com',
      gitlab_api_base: 'https://git.daydaymoney.com/api/v4',
    }

    it('展示要删除的仓库地址，确认时 emit slug', async () => {
      const wrapper = mount(Modal, {
        props: { target: region, submitting: false },
      })
      expect(wrapper.find('[data-testid="gitlab-region-delete-modal"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-region-delete-repo-address"]').text()).toBe(
        'https://git.daydaymoney.com'
      )
      expect(wrapper.text()).toContain('请确认要删除（停用）的仓库地址')
      expect(wrapper.text()).toContain('腾讯上海一区')

      await wrapper.find('[data-testid="gitlab-region-delete-confirm"]').trigger('click')
      expect(wrapper.emitted('confirm')?.[0]).toEqual(['tencent-sh-1'])
    })

    it('取消 emit cancel，不触发 confirm', async () => {
      const wrapper = mount(Modal, { props: { target: region } })
      await wrapper.find('[data-testid="gitlab-region-delete-cancel"]').trigger('click')
      expect(wrapper.emitted('cancel')).toHaveLength(1)
      expect(wrapper.emitted('confirm')).toBeUndefined()
    })

    it('target 为空时不渲染弹层', () => {
      const wrapper = mount(Modal, { props: { target: null } })
      expect(wrapper.find('[data-testid="gitlab-region-delete-modal"]').exists()).toBe(false)
    })
  })
}
