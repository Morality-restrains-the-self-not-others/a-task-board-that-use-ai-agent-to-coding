// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] LayerGraphZtree.events.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: LayerGraphZtree } = await import('./LayerGraphZtree.vue')

  describe('LayerGraphZtree event forwarding', () => {
    const baseNode = {
      id: 'layer:1',
      name: 'layer-1',
      nodeKind: 'layer',
      layerId: 'layer-1',
      open: true,
      canSubmitAndPush: true,
      submitAndPushDisabled: false,
      canSubmit: true,
      canMerge: true,
      mergeDisabled: false,
      submitAndMergeDisabled: false,
    }

    it('forwards layer-submit-and-push from root node', async () => {
      const wrapper = mount(LayerGraphZtree, {
        props: { flatNodes: [{ ...baseNode, pId: 0 }] },
      })
      await wrapper.find('[data-testid="layer-ztree-submit-push-btn"]').trigger('click')
      expect(wrapper.emitted('layer-submit-and-push')?.length).toBe(1)
      expect(wrapper.emitted('layer-submit-and-push')[0][0].layerId).toBe('layer-1')
    })

    it('forwards layer-submit-and-merge from root node', async () => {
      const wrapper = mount(LayerGraphZtree, {
        props: {
          flatNodes: [
            {
              ...baseNode,
              pId: 0,
              canSubmitAndPush: false,
              submitAndPushDisabled: true,
            },
          ],
        },
      })
      await wrapper.find('[data-testid="layer-ztree-submit-merge-btn"]').trigger('click')
      expect(wrapper.emitted('layer-submit-and-merge')?.length).toBe(1)
      expect(wrapper.emitted('layer-submit-and-merge')[0][0].layerId).toBe('layer-1')
    })

    it('nested child forwards layer-submit-and-push to root', async () => {
      const wrapper = mount(LayerGraphZtree, {
        props: {
          flatNodes: [
            {
              id: 'layer:parent',
              pId: 0,
              name: 'parent',
              nodeKind: 'layer',
              layerId: 'parent',
              open: true,
            },
            {
              ...baseNode,
              id: 'layer:child',
              pId: 'layer:parent',
              layerId: 'child',
              name: 'child',
            },
          ],
        },
      })
      const btns = wrapper.findAll('[data-testid="layer-ztree-submit-push-btn"]')
      expect(btns.length).toBeGreaterThanOrEqual(1)
      await btns[btns.length - 1].trigger('click')
      expect(wrapper.emitted('layer-submit-and-push')?.length).toBe(1)
      expect(wrapper.emitted('layer-submit-and-push')[0][0].layerId).toBe('child')
    })

    it('shows clickable PR anchor when canOpenPr', () => {
      const wrapper = mount(LayerGraphZtree, {
        props: {
          flatNodes: [
            {
              id: 'layer:pr',
              pId: 0,
              name: 'with-pr',
              nodeKind: 'layer',
              layerId: 'layer-pr',
              open: true,
              canOpenPr: true,
              prHtmlUrl: 'https://github.com/acme/repo/pull/9',
              prTitle: '打开 PR 审查页',
            },
          ],
        },
      })
      const link = wrapper.find('[data-testid="layer-ztree-pr-btn"]')
      expect(link.exists()).toBe(true)
      expect(link.element.tagName.toLowerCase()).toBe('a')
      expect(link.attributes('href')).toBe('https://github.com/acme/repo/pull/9')
      expect(link.attributes('target')).toBe('_blank')
    })
  })
}
