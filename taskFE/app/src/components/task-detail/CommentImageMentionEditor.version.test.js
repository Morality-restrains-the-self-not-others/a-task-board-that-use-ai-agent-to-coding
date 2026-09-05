// @vitest-environment jsdom
// 评论 $镜像 下拉菜单项需展示 name:version（无版本则仅名称），正文 mention 仍只写名称。
if (!process.env.VITEST) {
  console.log('[skip] CommentImageMentionEditor.version.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: CommentImageMentionEditor } = await import('./CommentImageMentionEditor.vue')

  async function mountEditor(installedImages) {
    const wrapper = mount(CommentImageMentionEditor, {
      props: {
        modelValue: '',
        'onUpdate:modelValue': () => {},
        installedImages,
      },
      attachTo: document.body,
    })
    await nextTick()
    return wrapper
  }

  async function openMentionMenu(wrapper) {
    const editor = wrapper.find('[data-testid="comment-content-editor"]')
    editor.element.innerHTML = '$'
    await editor.trigger('input')
    await nextTick()
  }

  describe('CommentImageMentionEditor 下拉版本展示', () => {
    it('菜单项展示 name:version；无版本仅名称；选中仍以名称为 mention 正文', async () => {
      const wrapper = await mountEditor([
        { id: 'img-a', name: 'trae-agent', version: 'x86_64-latest' },
        { id: 'img-b', name: 'other', version: '1' },
        { id: 'img-c', name: 'plain' },
      ])

      await openMentionMenu(wrapper)

      const picker = wrapper.find('[data-testid="comment-image-mention-picker"]')
      expect(picker.exists()).toBe(true)
      const items = picker.findAll('li')
      expect(items.map((li) => li.text())).toEqual([
        'trae-agent:x86_64-latest',
        'other:1',
        'plain',
      ])

      // 点击带版本项：正文 mention 仍只写名称、不含版本（不改 mentions 契约）
      await items[0].trigger('mousedown')
      await nextTick()
      const body = wrapper.find('[data-testid="comment-content-editor"]').text()
      expect(body).toContain('$trae-agent')
      expect(body).not.toContain('x86_64-latest')
      wrapper.unmount()
    })

    it('空查询打开时列出全部镜像（含版本）', async () => {
      const wrapper = await mountEditor([{ id: 'img-d', name: 'demo', version: 'v2' }])
      await openMentionMenu(wrapper)
      const picker = wrapper.find('[data-testid="comment-image-mention-picker"]')
      expect(picker.exists()).toBe(true)
      expect(picker.findAll('li')[0].text()).toBe('demo:v2')
      wrapper.unmount()
    })
  })
}
