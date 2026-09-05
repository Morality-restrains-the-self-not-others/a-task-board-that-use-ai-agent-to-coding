// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] layerZtreePushError.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    collectLatestLayerPushError,
    formatPushErrorClipboardText,
    isGitOauthBindingMissing,
    isGitPushPermissionDenied,
    layerPushErrorFields,
    missingBindingRepoMatchKeys,
  } = await import('./layerZtreePushError.js')

  const permissionDetail =
    '推送失败（1/1） 失败：ruandao/helloworld（路径 helloworld） — remote: Permission to ruandao/helloworld.git denied to alice.'

  describe('isGitPushPermissionDenied', () => {
    it('detects GitHub Permission to repo denied', () => {
      expect(isGitPushPermissionDenied(permissionDetail)).toBe(true)
      expect(isGitPushPermissionDenied('remote: Permission to acme/demo.git denied to bob')).toBe(true)
    })

    it('detects GitLab not-allowed-to-push and GitHub 403', () => {
      expect(isGitPushPermissionDenied('You are not allowed to push code to this project')).toBe(true)
      expect(isGitPushPermissionDenied("fatal: unable to access 'https://github.com/a/b.git/': The requested URL returned error: 403")).toBe(true)
      expect(isGitPushPermissionDenied('Write access to repository not granted')).toBe(true)
    })

    it('does not treat missing token or empty as permission denied', () => {
      expect(isGitPushPermissionDenied('')).toBe(false)
      expect(isGitPushPermissionDenied('该仓库未找到可用的 OAuth access_token')).toBe(false)
      expect(isGitPushPermissionDenied('network unreachable')).toBe(false)
    })
  })

  describe('isGitOauthBindingMissing', () => {
    it('detects BINDING_MISSING payload and 缺少绑定 copy', () => {
      expect(isGitOauthBindingMissing('error_code":"BINDING_MISSING"')).toBe(true)
      expect(isGitOauthBindingMissing('缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/org/repo')).toBe(true)
      expect(isGitOauthBindingMissing(permissionDetail)).toBe(false)
      expect(isGitOauthBindingMissing('')).toBe(false)
    })
  })

  describe('missingBindingRepoMatchKeys', () => {
    it('extracts GitLab match key from 缺少绑定 copy', () => {
      expect(
        missingBindingRepoMatchKeys(
          '请先在创建或编辑任务、或评论「提交并运行」时为每个仓库完成 Git 授权绑定。（缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work）',
        ),
      ).toEqual(['gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work'])
    })

    it('extracts multiple match keys (gitlab + github) and builds first resolvable bind href', () => {
      const keys = missingBindingRepoMatchKeys(
        '请先完成该仓库的 Git 授权绑定。（缺少绑定: gitlab.daydaymoney.com/group/a, github.com/acme/demo）',
      )
      expect(keys).toEqual(['gitlab.daydaymoney.com/group/a', 'github.com/acme/demo'])
      const fields = layerPushErrorFields({
        git_remote: {
          last_push_error:
            '从 task2app 拉取 GitLab AccessToken 失败：HTTP 409 https://api.example/layer-git-oauth-access-tokens/: {"detail":"请先完成该仓库的 Git 授权绑定。（缺少绑定: gitlab.daydaymoney.com/group/a, github.com/acme/demo）","error_code":"BINDING_MISSING","failed_stage":"binding_check","ok":false}',
        },
      })
      expect(fields.pushErrorKind).toBe('binding')
      expect(fields.pushErrorBindHref).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(decodeURIComponent(fields.pushErrorBindHref)).toContain(
        'https://gitlab.daydaymoney.com/group/a',
      )
    })

    it('returns empty when no 缺少绑定 list present', () => {
      expect(missingBindingRepoMatchKeys('')).toEqual([])
      expect(missingBindingRepoMatchKeys('请先完成该仓库的 Git 授权绑定。')).toEqual([])
    })
  })

  describe('layerPushErrorFields', () => {
    it('maps git_remote.last_push_error to ztree chip fields', () => {
      const fields = layerPushErrorFields({
        git_remote: {
          last_push_error: '该仓库未找到可用的 OAuth access_token',
          last_push_error_trace_id: 'trace-abc',
        },
      })
      expect(fields.pushErrorLabel).toBe('push 失败')
      expect(fields.pushErrorKind).toBe('generic')
      expect(fields.pushErrorTitle).toBe('该仓库未找到可用的 OAuth access_token')
      expect(fields.pushErrorTraceId).toBe('trace-abc')
    })

    it('labels GitHub permission-denied as push 无权限 and hints re-authorize', () => {
      const fields = layerPushErrorFields({
        git_remote: {
          last_push_error: permissionDetail,
          last_push_error_trace_id: 'tid-perm',
        },
      })
      expect(fields.pushErrorLabel).toBe('push 无权限')
      expect(fields.pushErrorKind).toBe('permission')
      expect(fields.pushErrorTitle).toContain('无写权限')
      expect(fields.pushErrorTitle).toContain('换有写权限的账号重新授权')
      expect(fields.pushErrorTitle).toContain('ruandao/helloworld')
      expect(fields.pushErrorTraceId).toBe('tid-perm')
    })

    it('labels BINDING_MISSING as 未绑定 Git 授权', () => {
      const fields = layerPushErrorFields({
        git_remote: {
          last_push_error:
            '从 task2app 拉取 GitHub AccessToken 失败：HTTP 409 https://api.example/layer-github-oauth-access-tokens/: {"detail":"请先在任务详情「关联项目」中为每个仓库选择并保存 Git 授权账号。（缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work）","detail_safe":"请先在任务详情绑定 GitHub 授权账号","error_code":"BINDING_MISSING","failed_stage":"binding_check","ok":false}',
        },
      })
      expect(fields.pushErrorLabel).toBe('未绑定 Git 授权')
      expect(fields.pushErrorKind).toBe('binding')
      expect(fields.pushErrorTitle).toContain('Git 授权')
      expect(fields.pushErrorTitle).not.toContain('关联项目')
      expect(fields.pushErrorBindHref).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(fields.pushErrorBindHref).toContain('grant_kind=pending')
      expect(decodeURIComponent(fields.pushErrorBindHref)).toContain(
        'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work',
      )
    })

    it('returns empty when no last_push_error', () => {
      expect(layerPushErrorFields({ git_remote: { ahead: 1 } })).toEqual({
        pushErrorLabel: '',
        pushErrorTitle: '',
        pushErrorTraceId: '',
        pushErrorKind: '',
      })
    })
  })

  describe('formatPushErrorClipboardText', () => {
    it('copies title and traceId when both present', () => {
      expect(formatPushErrorClipboardText({
        pushErrorLabel: 'push 失败',
        pushErrorTitle: '该仓库未找到可用的 OAuth access_token',
        pushErrorTraceId: 'tid-push',
      })).toBe('该仓库未找到可用的 OAuth access_token\ntraceId: tid-push')
    })

    it('copies title only when traceId is missing', () => {
      expect(formatPushErrorClipboardText({
        pushErrorLabel: 'push 失败',
        pushErrorTitle: 'network unreachable',
      })).toBe('network unreachable')
    })

    it('falls back to label when title is empty', () => {
      expect(formatPushErrorClipboardText({
        pushErrorLabel: 'push 无权限',
        pushErrorTitle: '  ',
        pushErrorTraceId: 'tid-perm',
      })).toBe('push 无权限\ntraceId: tid-perm')
    })

    it('returns empty when node has no error text', () => {
      expect(formatPushErrorClipboardText({})).toBe('')
      expect(formatPushErrorClipboardText(null)).toBe('')
    })
  })

  describe('collectLatestLayerPushError', () => {
    it('returns empty when snapshot has no layers or errors', () => {
      expect(collectLatestLayerPushError(null)).toBe('')
      expect(collectLatestLayerPushError({ layers: [] })).toBe('')
      expect(collectLatestLayerPushError({ layers: [{ git_remote: { ahead: 1 } }] })).toBe('')
    })

    it('prefers a permission-denied error over a later generic error', () => {
      expect(
        collectLatestLayerPushError({
          layers: [
            { git_remote: { last_push_error: permissionDetail } },
            { git_remote: { last_push_error: 'network unreachable' } },
          ],
        }),
      ).toBe(permissionDetail)
    })

    it('returns the last generic error when none are permission-denied', () => {
      expect(
        collectLatestLayerPushError({
          layers: [
            { git_remote: { last_push_error: '该仓库未找到可用的 OAuth access_token' } },
            { git_remote: { last_push_error: '  network unreachable  ' } },
          ],
        }),
      ).toBe('network unreachable')
    })
  })
}
