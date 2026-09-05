// @vitest-environment jsdom
// OPT-20260811-066：首页页脚占位链接真实化——首页→/、联系我们→页脚联系区锚点、
// 帮助中心/常见问题无真实目标页（待产品确认路径）故隐藏，避免 href="#" 点击无导航。
// OPT-20260824-083：页脚元素调整——移除「项目管理」链接与整个「支持」栏（含联系我们锚点链接）；
// 新增「常见问题」链接（/faq/ 路由，展示 src/faq/*.md 文档）。
if (!process.env.VITEST) {
  console.log('[skip] Home.footer-links.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Home = (await import('./Home.vue')).default

  const mountHome = () =>
    mount(Home, {
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

  describe('Home.vue 页脚占位链接（OPT-20260811-066）', () => {
    it('页脚品牌文案为「云端Coding 平台」', () => {
      const wrapper = mountHome()
      const brand = wrapper.find('[data-testid="home-footer-brand"]')
      expect(brand.exists()).toBe(true)
      expect(brand.text().trim()).toBe('云端Coding 平台')
      expect(brand.classes()).toContain('text-2xl')
      expect(brand.classes()).toContain('font-bold')
    })

    it('「首页」跳转根路径 /', () => {
      const wrapper = mountHome()
      const homeLink = wrapper.find('[data-testid="home-footer-home-link"]')
      expect(homeLink.exists()).toBe(true)
      expect(homeLink.attributes('href')).toBe('/')
    })

    it('页脚联系区 id=footer-contact 保留，支持栏内锚点链接已移除', () => {
      const wrapper = mountHome()
      const footer = wrapper.find('footer')
      // 联系区（第四栏）保留，URL 片段 #footer-contact 仍有效
      expect(footer.find('#footer-contact').exists()).toBe(true)
      // 「支持」栏已移除：不再存在指向 #footer-contact 的锚点链接
      const contactAnchors = footer
        .findAll('a')
        .filter((a) => a.attributes('href') === '#footer-contact')
      expect(contactAnchors.length).toBe(0)
    })

    it('「项目管理」页脚链接已移除（不再 href="#projects"）', () => {
      const wrapper = mountHome()
      const projectLinks = wrapper
        .find('footer')
        .findAll('a')
        .filter((a) => a.attributes('href') === '#projects')
      expect(projectLinks.length).toBe(0)
    })

    it('「支持」栏整体已移除（页脚不再渲染支持标题/内容）', () => {
      const wrapper = mountHome()
      const footerText = wrapper.find('footer').text()
      expect(footerText).not.toContain('支持')
    })

    it('页脚联系邮箱读取 VITE_CONTACT_EMAIL，源码无裸 author@example.com', async () => {
      const wrapper = mountHome()
      const email = String(import.meta.env.VITE_CONTACT_EMAIL || '').trim()
      expect(email).toMatch(/@/)
      const shown = wrapper.find('[data-testid="home-footer-contact-email"]')
      expect(shown.exists()).toBe(true)
      expect(shown.text().trim()).toBe(email)
      const src = (await import('./Home.vue?raw')).default
      expect(src).not.toContain('author@example.com')
    })

    it('「常见问题」链接真实化：联系我们栏新增 FAQ 路由链接（/faq/）', () => {
      const wrapper = mountHome()
      const footerText = wrapper.find('footer').text()
      expect(footerText).toContain('常见问题')
      const faqLink = wrapper.find('[data-testid="home-footer-faq-link"]')
      expect(faqLink.exists()).toBe(true)
      // router-link 指向 faq 命名路由（stub 渲染 data-to="faq"）
      expect(faqLink.attributes('data-to')).toBe('faq')
      // 页脚不再残留任何 href="#" 占位链接
      const placeholderLinks = wrapper.find('footer').findAll('a').filter((a) => a.attributes('href') === '#')
      expect(placeholderLinks.length).toBe(0)
    })

    it('页脚备案号读取 VITE_ICP_BEIAN，源码无硬编码闽ICP备', async () => {
      const wrapper = mountHome()
      const beian = String(import.meta.env.VITE_ICP_BEIAN || '').trim()
      expect(beian.length, 'conf/frontend/vue/config.yaml icpBeian').toBeGreaterThan(0)
      const link = wrapper.find('[data-testid="home-footer-icp-link"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('https://beian.miit.gov.cn/')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toMatch(/noopener/)
      expect(link.text().trim()).toBe(beian)
      const src = (await import('./Home.vue?raw')).default
      expect(src).not.toMatch(/闽ICP备\d+号/)
    })
  })
}
