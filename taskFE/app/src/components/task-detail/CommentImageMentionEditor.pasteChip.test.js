// @vitest-environment jsdom
// OPT-20260824-077: 手动粘贴/纯文本输入路径不渲染 mention chip，$镜像后 / 技能菜单
// 依赖 chip 才开启。粘贴（onPaste execCommand insertText → 仅 onInput）后应渲染 chip，
// 继续输入 / 即能开技能菜单。
if (!process.env.VITEST) {
  console.log('[skip] CommentImageMentionEditor.pasteChip.test.js requires vitest runtime')
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

  function setCaretToEnd(el) {
    const range = document.createRange()
    range.selectNodeContents(el)
    range.collapse(false)
    const sel = window.getSelection()
    sel.removeAllRanges()
    sel.addRange(range)
  }

  describe('CommentImageMentionEditor 粘贴路径 chip 渲染', () => {
    it('粘贴 $trae-agent / 后渲染 chip，继续输入 / 可开技能菜单', async () => {
      const wrapper = await mountEditor([{
        id: 'img-a',
        name: 'trae-agent',
        image_skills: {
          skills: [{ name: 'general-coding' }, { name: 'k8s-debug' }],
        },
      }])
      const editor = wrapper.find('[data-testid="comment-content-editor"]')

      // 模拟粘贴结果：正文为纯文本 $trae-agent /general-coding
      // （onPaste execCommand insertText → 仅 onInput）
      editor.element.innerHTML = '$trae-agent /general-coding'
      setCaretToEnd(editor.element)
      await editor.trigger('input')
      await nextTick()

      // 修复前：chip 不渲染；修复后：chip 出现，正文序列化仍为 $trae-agent ...
      const chip = editor.element.querySelector('.comment-mention-chip')
      expect(chip).not.toBeNull()
      expect(chip.dataset.mentionName).toBe('trae-agent')

      // renderPlainWithMention 重建 DOM 会把 caret 重置到 0；真实用户继续输入时
      // caret 落在 chip 之后的文本，触发下一次 onInput → 技能菜单可开。
      setCaretToEnd(editor.element)
      await editor.trigger('input')
      await nextTick()

      const skillPicker = wrapper.find('[data-testid="comment-image-skill-picker"]')
      expect(skillPicker.exists()).toBe(true)
      wrapper.unmount()
    })

    it('未知 $镜像 纯文本不渲染 chip（无匹配 installedImages）', async () => {
      const wrapper = await mountEditor([{ id: 'img-a', name: 'trae-agent' }])
      const editor = wrapper.find('[data-testid="comment-content-editor"]')
      editor.element.innerHTML = '$not-an-image'
      setCaretToEnd(editor.element)
      await editor.trigger('input')
      await nextTick()
      expect(editor.element.querySelector('.comment-mention-chip')).toBeNull()
      wrapper.unmount()
    })
  })
}
