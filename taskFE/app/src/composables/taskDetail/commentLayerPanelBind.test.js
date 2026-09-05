// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentLayerPanelBind.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { createCommentLayerPanelStore } = await import('./commentLayerPanelStore.js')
const { fileTreeLayerIdFromSlot, layerPanelViewFromSlot, layerGraphMetaLineFromSnapshot } = await import('./commentLayerPanelBind.js')
const { buildLayerBodyBindForComment } = await import('./taskDetailCommentsSectionHelpers.js')

describe('comment layer panel bind isolation', () => {
  it('file tree layer_id comes only from that comment slot', () => {
    const store = createCommentLayerPanelStore()
    store.patch('cmt_a', {
      snapshot: {
        layers: [{ layer_id: 'layer-a' }],
        jobs: [{ id: 'job-a', layer_id: 'layer-a', command_kind: 'trae' }],
      },
      selectedNode: { nodeKind: 'layer', id: '__layer__:layer-a', layerId: 'layer-a' },
    })
    store.patch('cmt_b', {
      snapshot: {
        layers: [{ layer_id: 'layer-b' }],
        jobs: [{ id: 'job-b', layer_id: 'layer-b', command_kind: 'trae' }],
      },
      selectedNode: { nodeKind: 'layer', id: '__layer__:layer-b', layerId: 'layer-b' },
    })
    expect(fileTreeLayerIdFromSlot(store.get('cmt_a'))).toBe('layer-a')
    expect(fileTreeLayerIdFromSlot(store.get('cmt_b'))).toBe('layer-b')
  })

  it('buildLayerBodyBindForComment does not leak A snapshot into B', () => {
    const props = {
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerPanelByCommentId: {
        cmt_a: {
          snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [] },
          selectedNode: { nodeKind: 'layer', layerId: 'layer-a', id: '__layer__:layer-a' },
          execLogTopError: 'A 失败',
          containerEndpointRegistered: true,
          commandText: 'cmd-a',
        },
        cmt_b: {
          snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
          selectedNode: { nodeKind: 'layer', layerId: 'layer-b', id: '__layer__:layer-b' },
          execLogTopError: '',
          containerEndpointRegistered: false,
          commandText: 'cmd-b',
        },
      },
      selectedLayerGraphFileTreeLayerId: 'layer-shared-bug',
      layerExecLogTopError: 'page-level leak',
      containerEndpointRegistered: true,
      layerGraphZNodes: [],
    }
    const bindA = buildLayerBodyBindForComment(props, 'cmt_a')
    const bindB = buildLayerBodyBindForComment(props, 'cmt_b')
    expect(bindA.selectedLayerGraphFileTreeLayerId).toBe('layer-a')
    expect(bindB.selectedLayerGraphFileTreeLayerId).toBe('layer-b')
    expect(bindA.layerExecLogTopError).toBe('A 失败')
    expect(bindB.layerExecLogTopError).toBe('')
    expect(bindA.containerEndpointRegistered).toBe(true)
    expect(bindB.containerEndpointRegistered).toBe(false)
    expect(bindA.commentId).toBe('cmt_a')
    expect(bindB.commentId).toBe('cmt_b')
  })

  it('cmdErrorTraceId stays isolated between comments', () => {
    const props = {
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerPanelByCommentId: {
        cmt_a: {
          snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [] },
          cmdError: '没有权限执行该操作，或登录态/容器授权已失效。请刷新页面后重试。',
          cmdErrorTraceId: 'tid-a-403',
        },
        cmt_b: {
          snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
          cmdError: '',
          cmdErrorTraceId: '',
        },
      },
      layerGraphZNodes: [],
    }
    const bindA = buildLayerBodyBindForComment(props, 'cmt_a')
    const bindB = buildLayerBodyBindForComment(props, 'cmt_b')
    expect(bindA.layerGraphCmdError).toMatch(/权限/)
    expect(bindA.layerGraphCmdErrorTraceId).toBe('tid-a-403')
    expect(bindB.layerGraphCmdError).toBe('')
    expect(bindB.layerGraphCmdErrorTraceId).toBe('')
  })

  it('A job-stream chunk and step cards do not appear in B bind', () => {
    const props = {
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerLiveOutputDisplay: 'page-level leak from A',
      layerJobOutputDisplay: 'page job leak',
      layerJobCommandHead: 'page cmd leak',
      layerAgentStepCards: [{ key: 'leak-a', rawStep: { content: 'A leak' } }],
      selectedZTreeLayerChangesPanel: { layer_id: 'layer-shared', changes: [{ path: 'leak.txt' }] },
      layerPanelByCommentId: {
        cmt_a: {
          snapshot: {
            layers: [{ layer_id: 'layer-a' }],
            jobs: [{ id: 'job-a', layer_id: 'layer-a', command_kind: 'trae' }],
          },
          selectedNode: { nodeKind: 'job', id: 'job-a' },
          liveOutputMap: { 'job-a': 'chunk from A' },
          jobExecutionPayload: {
            job: { id: 'job-a', command: 'run A', output: 'done A' },
            steps: { steps: [{ type: 'think', content: 'A step' }] },
          },
          layerChangesByLayerId: {
            'layer-a': { layer_id: 'layer-a', changes: [{ path: 'a.txt', kind: 'M' }], change_count: 1 },
          },
        },
        cmt_b: {
          snapshot: {
            layers: [{ layer_id: 'layer-b' }],
            jobs: [{ id: 'job-b', layer_id: 'layer-b', command_kind: 'trae' }],
          },
          selectedNode: { nodeKind: 'job', id: 'job-b' },
          liveOutputMap: { 'job-b': 'chunk from B' },
          jobExecutionPayload: {
            job: { id: 'job-b', command: 'run B', output: 'done B' },
            steps: { steps: [{ type: 'think', content: 'B step' }] },
          },
          layerChangesByLayerId: {
            'layer-b': { layer_id: 'layer-b', changes: [{ path: 'b.txt', kind: 'M' }], change_count: 1 },
          },
        },
      },
    }
    const bindA = buildLayerBodyBindForComment(props, 'cmt_a')
    const bindB = buildLayerBodyBindForComment(props, 'cmt_b')
    expect(bindA.layerLiveOutputDisplay).toBe('chunk from A')
    expect(bindB.layerLiveOutputDisplay).toBe('chunk from B')
    expect(bindA.layerJobCommandHead).toBe('run A')
    expect(bindB.layerJobCommandHead).toBe('run B')
    expect(bindA.layerAgentStepCards.some((c) => c.rawStep?.content === 'A step')).toBe(true)
    expect(bindA.layerAgentStepCards.some((c) => c.rawStep?.content === 'B step')).toBe(false)
    expect(bindB.layerAgentStepCards.some((c) => c.rawStep?.content === 'B step')).toBe(true)
    expect(bindB.layerAgentStepCards.some((c) => c.rawStep?.content === 'A step')).toBe(false)
    expect(bindA.selectedZTreeLayerChangesPanel.changes[0].path).toBe('a.txt')
    expect(bindB.selectedZTreeLayerChangesPanel.changes[0].path).toBe('b.txt')
  })

  it('missing slot does not leak page-level live output or step cards', () => {
    const bindA = buildLayerBodyBindForComment({
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerPanelByCommentId: {},
      layerLiveOutputDisplay: 'page-level leak from A',
      layerAgentStepCards: [{ key: 'leak-a' }],
      layerJobCommandHead: 'leaked cmd',
      selectedZTreeLayerChangesPanel: { layer_id: 'leak', changes: [] },
    }, 'cmt_a')
    expect(bindA.layerLiveOutputDisplay).toBe('')
    expect(bindA.layerAgentStepCards).toEqual([])
    expect(bindA.layerJobCommandHead).toBe('')
    expect(bindA.selectedZTreeLayerChangesPanel).toBe(null)
  })

  it('missing slot does not leak page-level file tree layer_id', () => {
    const bindA = buildLayerBodyBindForComment({
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerPanelByCommentId: {},
      selectedLayerGraphFileTreeLayerId: 'layer-shared-bug',
      layerGraphZNodes: [{ layerId: 'layer-shared-bug' }],
      layerExecLogTopError: 'page leak',
      containerEndpointRegistered: true,
    }, 'cmt_a')
    expect(bindA.selectedLayerGraphFileTreeLayerId).toBe('')
    expect(bindA.layerExecLogTopError).toBe('')
    expect(bindA.containerEndpointRegistered).toBe(false)
    expect(bindA.layerGraphZNodes).toEqual([])
  })

  it('layerGraphMetaLine shows pending when latest instruction is running', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1', created_at: '2026-08-22T01:00:00Z' }],
      jobs: [
        {
          id: 'job-old',
          layer_id: 'L1',
          command_kind: 'trae',
          status: 'completed',
          created_at: '2026-08-22T01:00:00Z',
          finished_at: '2026-08-22T01:05:00Z',
        },
        {
          id: 'job-new',
          layer_id: 'L1',
          command_kind: 'trae',
          status: 'running',
          created_at: '2026-08-22T02:00:00Z',
        },
      ],
      layers_root: '/app/onlineProject_state/layers',
    })
    expect(line).toContain('可写层 1 个')
    expect(line).toContain('任务 2 个')
    expect(line).toContain('最近指令完成 pending')
    expect(line).toContain('服务扫描 /app/onlineProject_state/layers')
  })

  it('layerGraphMetaLine shows pending when latest instruction is pending', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1' }],
      jobs: [
        {
          id: 'job-p',
          layer_id: 'L1',
          command_kind: 'shell',
          status: 'PENDING',
          created_at: '2026-08-22T03:00:00Z',
        },
      ],
    })
    expect(line).toContain('最近指令完成 pending')
  })

  it('layerGraphMetaLine shows local finished_at of latest non-clone job', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1' }],
      jobs: [
        {
          id: 'job-clone',
          layer_id: 'L1',
          command_kind: 'clone',
          status: 'running',
          created_at: '2026-08-22T09:00:00Z',
        },
        {
          id: 'job-done',
          layer_id: 'L1',
          command_kind: 'trae',
          status: 'completed',
          created_at: '2026-08-22T08:00:00Z',
          finished_at: '2026-08-22T14:01:03.000Z',
        },
      ],
    })
    expect(line).toContain('最近指令完成')
    expect(line).toMatch(/最近指令完成 2026/)
    expect(line).not.toContain('pending')
  })

  it('layerGraphMetaLine omits finish caption when only clone jobs exist', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1' }],
      jobs: [
        {
          id: 'job-clone',
          layer_id: 'L1',
          command_kind: 'clone',
          status: 'completed',
          created_at: '2026-08-22T08:00:00Z',
          finished_at: '2026-08-22T08:01:00Z',
        },
      ],
    })
    expect(line).not.toContain('最近指令完成')
  })

  it('layerGraphMetaLine omits finish caption when jobs empty', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1' }],
      jobs: [],
      layers_root: '/tmp/layers',
    })
    expect(line).not.toContain('最近指令完成')
    expect(line).toContain('服务扫描 /tmp/layers')
  })

  it('layerGraphMetaLine uses em dash when terminal job lacks finished_at', () => {
    const line = layerGraphMetaLineFromSnapshot({
      layers: [{ layer_id: 'L1' }],
      jobs: [
        {
          id: 'job-legacy',
          layer_id: 'L1',
          command_kind: 'trae',
          status: 'completed',
          created_at: '2026-08-22T08:00:00Z',
        },
      ],
    })
    expect(line).toContain('最近指令完成 —')
  })

  it('layerPanelViewFromSlot zNodes follow the slot snapshot not extras', () => {
    const viewA = layerPanelViewFromSlot({
      snapshot: { layers: [{ layer_id: 'only-a', created_at: '2026-01-01T00:00:00Z' }], jobs: [] },
    })
    const viewB = layerPanelViewFromSlot({
      snapshot: { layers: [{ layer_id: 'only-b', created_at: '2026-01-01T00:00:00Z' }], jobs: [] },
    })
    const idsA = viewA.layerGraphZNodes.map((n) => n.layerId).filter(Boolean)
    const idsB = viewB.layerGraphZNodes.map((n) => n.layerId).filter(Boolean)
    expect(idsA).toContain('only-a')
    expect(idsA).not.toContain('only-b')
    expect(idsB).toContain('only-b')
    expect(idsB).not.toContain('only-a')
  })

  it('layerPanelViewFromSlot 透传 containerReleased 使 git 动作禁用', () => {
    const slot = {
      snapshot: {
        layers: [
          {
            layer_id: 'L-rel',
            created_at: '2026-08-22T10:00:00Z',
            git_worktree_dirty: true,
            git_remote: { is_git: true, current_branch: 'feature/x', ahead: 2 },
          },
        ],
        jobs: [
          {
            id: 'J-rel',
            layer_id: 'L-rel',
            status: 'running',
            command_kind: 'trae',
            created_at: '2026-08-22T10:01:00Z',
          },
        ],
      },
    }
    const released = layerPanelViewFromSlot(slot, { containerReleased: true })
    const node = released.layerGraphZNodes.find((n) => n.layerId === 'L-rel')
    expect(node).toBeTruthy()
    expect(node.containerReleased).toBe(true)
    expect(node.submitAndPushDisabled).toBe(true)
    expect(node.submitAndPushTitle).toContain('服务器已释放')
    expect(node.pushDisabled).toBe(true)
    expect(released.containerReleased).toBe(true)

    const normal = layerPanelViewFromSlot(slot, {})
    const normalNode = normal.layerGraphZNodes.find((n) => n.layerId === 'L-rel')
    expect(normalNode.containerReleased).toBeUndefined()
    expect(normalNode.submitAndPushDisabled).toBe(false)
    expect(normal.containerReleased).toBe(false)
  })

  it('buildLayerBodyBindForComment 按评论 released 判定透传 containerReleased', () => {
    const props = {
      tenantId: 'T',
      workspaceId: 'W',
      taskId: 'TK',
      layerPanelByCommentId: {
        cmt_rel: {
          snapshot: {
            layers: [
              {
                layer_id: 'L-x',
                created_at: '2026-08-22T10:00:00Z',
                git_worktree_dirty: true,
                git_remote: { is_git: true, current_branch: 'feature/x', ahead: 1 },
              },
            ],
            jobs: [],
          },
        },
      },
      bindingStatusFor: () => 'released',
      serverRuntimeStatusPanel: {
        snapshotForComment: () => ({ serverRuntimeStatus: 'Released' }),
      },
      layerGraphMergeTargetBranch: 'develop',
      layerGraphZNodes: [],
    }
    const bind = buildLayerBodyBindForComment(props, 'cmt_rel')
    const node = bind.layerGraphZNodes.find((n) => n.layerId === 'L-x')
    expect(node).toBeTruthy()
    expect(node.containerReleased).toBe(true)
    expect(node.submitAndPushDisabled).toBe(true)
    expect(bind.containerReleased).toBe(true)
    expect(bind.showCommentLayerZtreeReleased).toBe(false)
    expect(bind.layerGraphZNodes.length).toBeGreaterThan(0)
  })

  it('CommentsSection per-comment body binds the comment slot not page-level command v-model', async () => {
    const { readFileSync } = await import('node:fs')
    const { dirname, join } = await import('node:path')
    const { fileURLToPath } = await import('node:url')
    const here = dirname(fileURLToPath(import.meta.url))
    const src = readFileSync(
      join(here, '../../components/task-detail/TaskDetailCommentsSection.vue'),
      'utf8',
    )
    const slot = src.split('#execution-details="{ comment')[1] || ''
    expect(slot).toContain('buildPerCommentLayerBodyBind')
    expect(slot).toContain('buildPerCommentLayerBodyListeners')
    expect(slot).not.toContain('v-model:layer-graph-command-text="layerGraphCommandText"')
  })
})
}
