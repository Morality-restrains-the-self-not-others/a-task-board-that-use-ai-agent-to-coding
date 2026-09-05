// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] WorkspaceAssociation.traceId.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { default: WorkspaceAssociation } = await import('./WorkspaceAssociation.vue')
  const { default: modalService } = await import('../utils/modalService.js')

const apiFetch = vi.hoisted(() => vi.fn())

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch,
}))

describe('WorkspaceAssociation data-traceId on request error', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    modalService.close()
  })

  it('sets modalService.state.traceId when associateWorkspace fails', async () => {
    const headers = new Headers({ 'X-Trace-Id': 'tid-assoc-fail-1' })
    const failResp = {
      ok: false,
      status: 501,
      headers,
      traceId: 'tid-assoc-fail-1',
      json: async () => ({
        error: 'endpoint not yet implemented in taskProjectService',
        trace_id: 'tid-assoc-fail-1',
      }),
    }
    apiFetch.mockResolvedValueOnce(failResp)

    const wrapper = mount(WorkspaceAssociation, {
      props: {
        project: { id: 'proj_1' },
        workspaces: [],
        availableWorkspaces: [{ id: 'ws_1', name: 'WS1' }],
        tenantId: 't1',
      },
    })

    await wrapper.find('select').setValue('ws_1')
    const addBtn = wrapper.findAll('button').find((b) => b.text().includes('添加关联'))
    await addBtn.trigger('click')
    await flushPromises()

    expect(modalService.state.traceId).toBe('tid-assoc-fail-1')
    expect(String(modalService.state.message || '')).toContain('endpoint not yet implemented')
  })
})
}
