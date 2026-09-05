// @vitest-environment jsdom
// OPT-20260824-083：常见问题页——展示 src/faq/*.md 文档（import.meta.glob ?raw 自动发现）。
// 2026-08-30：去掉侧边栏导航、「知识产权处理方法」与「账号与登录」文档，剩余文档纵向铺开。
if (!process.env.VITEST) {
  console.log('[skip] Faq.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Faq = (await import('./Faq.vue')).default

  const mountFaq = () =>
    mount(Faq, {
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template:
              '<a data-testid="stub-router-link" :data-to="typeof to === \'string\' ? to : (to.name || to.path)"><slot /></a>',
          },
        },
      },
    })

  describe('Faq.vue 常见问题页（OPT-20260824-083）', () => {
    it('不渲染文档侧边栏导航', () => {
      const wrapper = mountFaq()
      expect(wrapper.find('[data-testid="faq-doc-nav"]').exists()).toBe(false)
      expect(wrapper.find('aside').exists()).toBe(false)
    })

    it('纵向铺开剩余文档内容，不含账号与登录、不含知识产权处理方法', () => {
      const wrapper = mountFaq()
      const content = wrapper.find('[data-testid="faq-doc-content"]')
      expect(content.exists()).toBe(true)
      const text = content.text()
      expect(text).not.toContain('如何注册账号？')
      expect(text).not.toContain('账号与登录')
      expect(text).toContain('如何创建项目？')
      expect(text).toContain('平台如何计费？')
      expect(text).not.toContain('知识产权处理方法')
      expect(text).not.toContain('我们便不会主动提起诉讼')
      expect(wrapper.find('[data-testid="faq-doc-panel-account"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="faq-doc-panel-usage"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="faq-doc-panel-billing"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="faq-doc-panel-知识产权处理方法"]').exists()).toBe(false)
    })

    it('将 markdown 中的 {{contactEmail}} 替换为 conf 注入的联系邮箱', () => {
      const wrapper = mountFaq()
      const email = import.meta.env.VITE_CONTACT_EMAIL
      expect(typeof email).toBe('string')
      expect(email.trim()).toMatch(/@/)
      const text = wrapper.find('[data-testid="faq-doc-content"]').text()
      expect(text).toContain(email.trim())
      expect(text).not.toContain('{{contactEmail}}')
    })

    it('faq markdown 源文件不硬编码邮箱，使用 {{contactEmail}} 占位符', () => {
      const rawDocs = import.meta.glob('../faq/*.md', {
        query: '?raw',
        import: 'default',
        eager: true,
      })
      const joined = Object.values(rawDocs).join('\n')
      expect(joined).toContain('{{contactEmail}}')
      expect(joined).not.toMatch(/\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b/)
    })

    it('返回首页链接指向 home 路由', () => {
      const wrapper = mountFaq()
      const backLink = wrapper.find('[data-testid="faq-back-home-link"]')
      expect(backLink.exists()).toBe(true)
      expect(backLink.attributes('data-to')).toBe('home')
    })
  })
}
