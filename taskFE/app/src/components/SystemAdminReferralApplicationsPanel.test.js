// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminReferralApplicationsPanel.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: vi.fn(() => ''),
  }))

  const { default: Panel } = await import('./SystemAdminReferralApplicationsPanel.vue')

  function okJson(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => null },
    }
  }

  describe('SystemAdminReferralApplicationsPanel', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(okJson({
        items: [{
          id: '2877397583960502272',
          user_id: 'u_pending_1',
          status: 'pending',
          status_display: '待审批',
          applied_at: '2026-08-19T00:15:00Z',
          expires_at: null,
          reviewed_by: null,
          reject_reason: null,
          is_active: false,
          can_revoke: false,
          personal_intro: '我是平台活跃用户，日常使用任务与云主机，希望通过推荐帮助同事上手本平台。',
          legal_name: '张三',
          profit_sharing_ratio_display: '12%',
          referral_rate_display: '12%',
        }],
        total: 1,
      }))
    })

    it('mounts and renders 推荐码申请列表 with pending approve/reject actions', async () => {
      const wrapper = mount(Panel)
      await flushPromises()

      expect(wrapper.get('[data-testid="system-admin-referral-applications-panel"]').text())
        .toContain('推荐码申请列表')
      expect(wrapper.get('[data-testid="referral-applications-table"]').text())
        .toContain('2877397583960502272')
      expect(wrapper.text()).toContain('通过')
      expect(wrapper.text()).toContain('拒绝')
      expect(wrapper.get('[data-testid="referral-app-intro"]').text())
        .toContain('我是平台活跃用户')
      expect(wrapper.get('[data-testid="referral-app-legal-name"]').text())
        .toContain('张三')
      expect(wrapper.get('[data-testid="referral-applications-table"]').text())
        .toContain('分成比例')
      expect(wrapper.get('[data-testid="referral-applications-table"]').text())
        .not.toContain('分账比例')
      expect(wrapper.get('[data-testid="referral-applications-table"]').text())
        .not.toContain('推荐比例')
      expect(wrapper.get('[data-testid="referral-app-ratio"]').text())
        .toContain('12%')
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/system-admin/referral/applications/'),
        expect.any(Object),
      )
    })

    it('renders 推荐码 column with share_code and copy button (OPT-20260824-006)', async () => {
      apiFetch.mockResolvedValueOnce(okJson({
        items: [{
          id: 'app_code',
          user_id: 'u_share_1',
          status: 'pending',
          applied_at: '2026-08-19T00:15:00Z',
          share_code: 'DR2AKvP9J9',
        }, {
          id: 'app_no_code',
          user_id: 'u_no_code',
          status: 'pending',
          applied_at: '2026-08-19T00:15:00Z',
        }],
        total: 2,
      }))
      const wrapper = mount(Panel)
      await flushPromises()

      const table = wrapper.get('[data-testid="referral-applications-table"]')
      expect(table.text()).toContain('推荐码')
      const codeCells = wrapper.findAll('[data-testid="referral-app-share-code"]')
      expect(codeCells).toHaveLength(2)
      expect(codeCells[0].text()).toContain('DR2AKvP9J9')
      expect(codeCells[1].text()).toContain('—')
      // 有码行出现复制按钮，无码行无按钮
      expect(wrapper.findAll('[data-testid="referral-app-copy-code"]')).toHaveLength(1)
    })

    it('lookup hit highlights matching application row (OPT-20260824-007)', async () => {
      apiFetch
        .mockResolvedValueOnce(okJson({
          items: [{
            id: 'app_hit',
            user_id: 'u_hit',
            status: 'pending',
            applied_at: '2026-08-19T00:15:00Z',
            share_code: 'DR2AKvP9J9',
          }],
          total: 1,
        }))
        .mockResolvedValueOnce(okJson({
          found: true,
          code: 'DR2AKvP9J9',
          user_id: 'u_hit',
          username: '测试用户',
          channel_name: '默认',
          is_default: true,
          status: 'active',
        }))
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-input"]').setValue('DR2AKvP9J9')
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      const hitRow = wrapper.findAll('tr').find((tr) => tr.text().includes('u_hit'))
      expect(hitRow).toBeTruthy()
      expect(hitRow.classes()).toContain('ring-2')
    })

    it('lookup hit outside current list does not highlight (OPT-20260824-007)', async () => {
      apiFetch
        .mockResolvedValueOnce(okJson({
          items: [{
            id: 'app_other',
            user_id: 'u_list',
            status: 'pending',
            applied_at: '2026-08-19T00:15:00Z',
          }],
          total: 1,
        }))
        .mockResolvedValueOnce(okJson({
          found: true,
          code: 'XYZ',
          user_id: 'u_not_in_list',
          username: '别人',
          channel_name: '默认',
          is_default: true,
          status: 'active',
        }))
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-input"]').setValue('XYZ')
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      const rows = wrapper.findAll('tr')
      const hitRow = rows.find((tr) => tr.text().includes('u_not_in_list'))
      expect(hitRow).toBeFalsy()
      const listRow = rows.find((tr) => tr.text().includes('u_list'))
      expect(listRow.classes()).not.toContain('ring-2')
    })

    it('approve opens reason modal then posts with Idempotency-Key', async () => {
      apiFetch
        .mockResolvedValueOnce(okJson({
          items: [{
            id: 'app_1',
            user_id: 'u1',
            status: 'pending',
            applied_at: '2026-08-19T00:15:00Z',
          }],
          total: 1,
        }))
        .mockResolvedValueOnce(okJson({ ok: true }))
        .mockResolvedValueOnce(okJson({ items: [], total: 0 }))

      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-app-approve"]').trigger('click')
      await wrapper.get('[data-testid="referral-action-reason"]').setValue('申请人介绍充分予以通过')
      await wrapper.get('[data-testid="referral-action-confirm"]').trigger('click')
      await flushPromises()

      expect(apiFetch).toHaveBeenCalledWith(
        '/api/system-admin/referral/applications/app_1/approve/',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({ 'Idempotency-Key': expect.any(String) }),
        }),
      )
    })

    it('approved active row shows 取消资格 instead of a lone dash', async () => {
      apiFetch.mockResolvedValueOnce(okJson({
        items: [{
          id: 'app_ok',
          user_id: 'u2',
          status: 'approved',
          status_display: '已通过',
          is_active: true,
          can_revoke: true,
          applied_at: '2026-08-19T00:15:00Z',
        }],
        total: 1,
      }))
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-app-revoke"]').text()).toContain('取消资格')
      expect(wrapper.get('[data-testid="referral-app-audit"]').exists()).toBe(true)
    })

    it('revoke submits reason and audit drawer loads history', async () => {
      apiFetch
        .mockResolvedValueOnce(okJson({
          items: [{
            id: 'app_ok',
            user_id: 'u2',
            status: 'approved',
            is_active: true,
            can_revoke: true,
          }],
          total: 1,
        }))
        .mockResolvedValueOnce(okJson({ status: 'revoked' }))
        .mockResolvedValueOnce(okJson({ items: [], total: 0 }))
        .mockResolvedValueOnce(okJson({
          items: [{ id: 1, action: 'revoke', operator_id: 'admin', reason: '违反推荐政策取消资格', created_at: '2026-08-22T05:00:00Z' }],
          total: 1,
        }))

      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-app-revoke"]').trigger('click')
      await wrapper.get('[data-testid="referral-action-reason"]').setValue('违反推荐政策取消资格')
      await wrapper.get('[data-testid="referral-action-confirm"]').trigger('click')
      await flushPromises()
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/system-admin/referral/applications/app_ok/revoke/',
        expect.objectContaining({ method: 'POST' }),
      )

      apiFetch.mockResolvedValueOnce(okJson({
        items: [{ id: 1, action: 'revoke', operator_id: 'admin', reason: '违反推荐政策取消资格', created_at: '2026-08-22T05:00:00Z' }],
        total: 1,
      }))
      // reload list empty — remount with revoked row to open audit
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(okJson({
        items: [{
          id: 'app_ok',
          user_id: 'u2',
          status: 'revoked',
          can_revoke: false,
        }],
        total: 1,
      }))
      const wrapper2 = mount(Panel)
      await flushPromises()
      apiFetch.mockResolvedValueOnce(okJson({
        items: [{ id: 1, action: 'revoke', operator_id: 'admin', reason: '违反推荐政策取消资格', created_at: '2026-08-22T05:00:00Z' }],
        total: 1,
      }))
      await wrapper2.get('[data-testid="referral-app-audit"]').trigger('click')
      await flushPromises()
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/system-admin/referral/applications/app_ok/audit/',
        expect.any(Object),
      )
      expect(wrapper2.get('[data-testid="referral-audit-drawer"]').text()).toContain('违反推荐政策取消资格')
    })

    // ── 推荐码反查：输入推荐码反向定位具体用户 ──

    it('lookup share code shows the owning user (found)', async () => {
      apiFetch.mockResolvedValueOnce(okJson({
        items: [{
          id: 'app_1', user_id: 'u1', status: 'pending',
          applied_at: '2026-08-19T00:15:00Z',
        }],
        total: 1,
      }))
      apiFetch.mockResolvedValueOnce(okJson({
        found: true,
        code: 'DR2AKvP9J9',
        user_id: 'ref-owner-1',
        username: '赖金燕',
        channel_name: '默认',
        is_default: true,
        status: 'active',
      }))

      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-input"]').setValue('DR2AKvP9J9')
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      expect(apiFetch).toHaveBeenCalledWith(
        '/api/system-admin/referral/share-code/lookup/?code=DR2AKvP9J9',
        expect.any(Object),
      )
      const result = wrapper.get('[data-testid="referral-code-lookup-result"]')
      expect(result.text()).toContain('已找到')
      expect(result.text()).toContain('DR2AKvP9J9')
      expect(result.get('[data-testid="referral-code-lookup-user-id"]').text()).toContain('ref-owner-1')
      expect(result.get('[data-testid="referral-code-lookup-username"]').text()).toContain('赖金燕')
      expect(result.text()).toContain('默认')
      expect(result.text()).toContain('active')
    })

    it('lookup share code with unknown code shows not-found hint', async () => {
      apiFetch.mockResolvedValueOnce(okJson({
        items: [],
        total: 0,
      }))
      apiFetch.mockResolvedValueOnce(okJson({ found: false }))

      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-input"]').setValue('NO_SUCH_CODE')
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      expect(wrapper.get('[data-testid="referral-code-lookup-not-found"]').text())
        .toContain('未找到使用该推荐码的用户')
    })

    it('lookup with empty input shows validation error and does not call API', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      expect(wrapper.get('[data-testid="referral-code-lookup-error"]').text())
        .toContain('请输入推荐码')
      const lookupCalls = apiFetch.mock.calls.filter(([url]) =>
        String(url).includes('/referral/share-code/lookup/'))
      expect(lookupCalls).toHaveLength(0)
    })

    it('lookup API failure surfaces error with trace id', async () => {
      const { extractTraceId } = await import('../utils/traceId.js')
      vi.mocked(extractTraceId).mockReturnValueOnce('trace-abc-123')

      apiFetch.mockResolvedValueOnce(okJson({
        items: [],
        total: 0,
      }))
      apiFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ detail: '查询失败，请稍后重试', trace_id: 'trace-abc-123' }),
        headers: { get: () => null },
      })

      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="referral-code-lookup-input"]').setValue('BAD_CODE')
      await wrapper.get('[data-testid="referral-code-lookup-btn"]').trigger('click')
      await flushPromises()

      const err = wrapper.get('[data-testid="referral-code-lookup-error"]')
      expect(err.text()).toContain('查询失败')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('trace-abc-123')
    })
  })
}
