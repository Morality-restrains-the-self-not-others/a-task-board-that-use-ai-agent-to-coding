// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] LayerGraphZtreeNode.pushError.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  const { default: LayerGraphZtreeNode } = await import('./LayerGraphZtreeNode.vue')

  describe('LayerGraphZtreeNode push error chip', () => {
    it('shows push 失败 next to submit-and-push and sets data-traceId', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-err',
            name: 'completed job',
            layerId: 'L-err',
            jobId: 'J-err',
            children: [],
            canSubmitAndPush: true,
            submitAndPushDisabled: false,
            pushErrorLabel: 'push 失败',
            pushErrorTitle: '该仓库未找到可用的 OAuth access_token',
            pushErrorTraceId: 'tid-push',
          },
        },
      })
      const chip = wrapper.get('[data-testid="layer-ztree-push-error-label"]')
      expect(chip.text()).toBe('push 失败')
      expect(chip.attributes('title')).toContain('OAuth access_token')
      expect(chip.attributes('data-traceid') || chip.attributes('data-traceId')).toBe('tid-push')
      expect(wrapper.get('[data-testid="layer-ztree-submit-push-btn"]').text()).toContain('提交并创建PR')
      expect(wrapper.get('[data-testid="layer-ztree-push-error-copy"]').text()).toBe('复制')
    })

    it('shows push 无权限 when node.pushErrorLabel is permission-denied copy', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-perm',
            name: 'completed job',
            layerId: 'L-perm',
            jobId: 'J-perm',
            children: [],
            canSubmitAndPush: true,
            submitAndPushDisabled: false,
            pushErrorLabel: 'push 无权限',
            pushErrorTitle: '当前授权账号对该仓库无写权限。请换有写权限的账号重新授权。\nremote: Permission to ruandao/helloworld.git denied',
            pushErrorTraceId: 'tid-perm',
          },
        },
      })
      const chip = wrapper.get('[data-testid="layer-ztree-push-error-label"]')
      expect(chip.text()).toBe('push 无权限')
      expect(chip.attributes('title')).toContain('无写权限')
      expect(chip.attributes('data-traceid') || chip.attributes('data-traceId')).toBe('tid-perm')
      expect(wrapper.get('[data-testid="layer-ztree-push-error-copy"]').text()).toBe('复制')
    })

    it('binding missing 时渲染真实 a[href] 绑定入口且保留 chip 与复制', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-bind',
            name: 'binding missing layer',
            layerId: 'L-bind',
            children: [],
            pushErrorLabel: '未绑定 Git 授权',
            pushErrorKind: 'binding',
            pushErrorTitle: '请先完成该仓库的 Git 授权绑定。（缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work）',
            pushErrorTraceId: 'tid-bind',
            pushErrorBindHref:
              '/api/git-oauth/gitlab-start-from-gateway/?repo_url=https%3A%2F%2Fgitlab-tencent-sh-1.daydaymoney.com%2Fexample-user%2Fram-work&grant_kind=pending',
          },
        },
      })
      const anchor = wrapper.get('[data-testid="layer-ztree-push-error-bind"]')
      expect(anchor.element.tagName).toBe('A')
      expect(anchor.attributes('href')).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(anchor.attributes('href')).toContain('grant_kind=pending')
      expect(anchor.text()).toBe('去绑定')
      expect(wrapper.get('[data-testid="layer-ztree-push-error-label"]').text()).toBe(
        '未绑定 Git 授权',
      )
      expect(wrapper.get('[data-testid="layer-ztree-push-error-copy"]').text()).toBe('复制')
    })

    it('does not render bind anchor when no pushErrorBindHref', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-bind-no',
            name: 'binding missing layer',
            layerId: 'L-bind-no',
            children: [],
            pushErrorLabel: '未绑定 Git 授权',
            pushErrorKind: 'binding',
            pushErrorTitle: '请先完成该仓库的 Git 授权绑定',
          },
        },
      })
      expect(wrapper.find('[data-testid="layer-ztree-push-error-bind"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="layer-ztree-push-error-label"]').text()).toBe(
        '未绑定 Git 授权',
      )
    })

    it('containerReleased=true 时提交并创建PR 按钮仍渲染但禁用并带明确文案', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-rel',
            name: 'released layer',
            layerId: 'L-rel',
            jobId: 'J-rel',
            children: [],
            canSubmitAndPush: true,
            submitAndPushDisabled: true,
            submitAndPushTitle: '服务器已释放，无法提交并创建 PR',
            containerReleased: true,
          },
        },
      })
      const btn = wrapper.get('[data-testid="layer-ztree-submit-push-btn"]')
      expect(btn.text()).toContain('提交并创建PR')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(btn.attributes('title')).toContain('服务器已释放')
    })

    it('does not render copy button when there is no push error', () => {
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-ok',
            name: 'clean layer',
            layerId: 'L-ok',
            children: [],
            canSubmitAndPush: true,
            submitAndPushDisabled: false,
          },
        },
      })
      expect(wrapper.find('[data-testid="layer-ztree-push-error-label"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="layer-ztree-push-error-copy"]').exists()).toBe(false)
    })

    it('copy button writes failure title and traceId then shows 已复制', async () => {
      const writeText = vi.fn(async () => {})
      vi.stubGlobal('navigator', { clipboard: { writeText } })
      const wrapper = mount(LayerGraphZtreeNode, {
        props: {
          node: {
            id: '__layer__:L-copy',
            name: 'failed job',
            layerId: 'L-copy',
            children: [],
            pushErrorLabel: 'push 失败',
            pushErrorTitle: '该仓库未找到可用的 OAuth access_token',
            pushErrorTraceId: 'tid-copy',
          },
        },
      })
      const btn = wrapper.get('[data-testid="layer-ztree-push-error-copy"]')
      expect(btn.text()).toBe('复制')
      expect(btn.attributes('title')).toBe('复制失败信息')
      await btn.trigger('click')
      await flushPromises()
      expect(writeText).toHaveBeenCalledTimes(1)
      expect(writeText.mock.calls[0][0]).toBe(
        '该仓库未找到可用的 OAuth access_token\ntraceId: tid-copy',
      )
      expect(btn.text()).toBe('已复制')
      vi.unstubAllGlobals()
    })
  })
}
