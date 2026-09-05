/** 与同批 taskFE 视图/组件拆分变更配对回归（BDD 共演进门禁）。含 WorkPanel 脚本外移 utils。 */
import { describe, expect, it } from 'vitest'
import {
  buildZTreeNodesFromLayers,
  buildZTreeNodesSerialFromLayers,
  LAYER_GRAPH_ROOT_ID,
  LAYER_TREE_NODE_PREFIX,
  simpleDataToTreeRoots,
} from './layerZtreeNodes.js'

function findLayerNode(nodes, layerId) {
  return nodes.find((n) => n.id === `${LAYER_TREE_NODE_PREFIX}${layerId}`)
}

describe('buildZTreeNodesSerialFromLayers', () => {
  it('多条任务时层 job_status/mind_state 滞后于 jobs 列表则层级行应对齐终态（非 running）', () => {
    const layers = [
      {
        layer_id: 'L-stale',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        job_status: 'running',
        mind_state: 'running',
        command: 'echo hi',
      },
    ]
    const jobs = [
      {
        id: 'J-a',
        layer_id: 'L-stale',
        status: 'completed',
        command_kind: 'trae',
        command: 'first',
        created_at: '2026-04-11T10:01:00Z',
      },
      {
        id: 'J-b',
        layer_id: 'L-stale',
        status: 'completed',
        command_kind: 'trae',
        command: 'second',
        created_at: '2026-04-11T10:03:00Z',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, jobs)
    const layerRow = findLayerNode(serial, 'L-stale')
    expect(layerRow).toBeTruthy()
    expect(layerRow.name).toMatch(/^completed/i)
    expect(layerRow.ztStyle).not.toBe('active')
  })

  it('仅存在 clone 任务时合并到可写层行：展示 running 与克隆指令', () => {
    const layers = [
      {
        layer_id: 'L-clone',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: null,
        job_status: 'pending',
        mind_state: 'running',
        command: '',
      },
    ]
    const jobs = [
      {
        id: 'J-clone-1',
        layer_id: 'L-clone',
        status: 'running',
        command_kind: 'clone',
        command: 'git clone https://example.com/repo.git',
        created_at: '2026-04-11T10:00:01Z',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, jobs)
    const layerRow = findLayerNode(serial, 'L-clone')
    expect(layerRow).toBeTruthy()
    expect(layerRow.name).toMatch(/^running/i)
    expect(layerRow.name).toContain('git clone')
    expect(layerRow.ztStyle).toBe('active')
    expect(serial.filter((n) => n.nodeKind === 'job').length).toBe(0)
  })

  it('仅存在 clone 任务且已完成时层行应为 completed', () => {
    const layers = [
      {
        layer_id: 'L-clone2',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: null,
        job_status: 'running',
        mind_state: 'running',
        command: '',
      },
    ]
    const jobs = [
      {
        id: 'J-clone-done',
        layer_id: 'L-clone2',
        status: 'completed',
        command_kind: 'clone',
        command: 'git clone https://example.com/done.git',
        created_at: '2026-04-11T10:00:01Z',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, jobs)
    const layerRow = findLayerNode(serial, 'L-clone2')
    expect(layerRow).toBeTruthy()
    expect(layerRow.name).toMatch(/^completed/i)
    expect(layerRow.name).toContain('git clone')
    expect(layerRow.ztStyle).not.toBe('active')
  })

  it('虚拟根下平铺：parent_layer_id 仅写入 logicalParentLayerId，层行均挂虚拟根', () => {
    const layers = [
      {
        layer_id: 'L1',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        parent_layer_id: '',
      },
      {
        layer_id: 'L2',
        created_at: '2026-04-11T10:05:00Z',
        git_worktree_dirty: true,
        parent_layer_id: 'L1',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, [])
    expect(serial.some((n) => n.id === LAYER_GRAPH_ROOT_ID)).toBe(true)
    const l1 = serial.find((n) => n.id === `${LAYER_TREE_NODE_PREFIX}L1`)
    const l2 = serial.find((n) => n.id === `${LAYER_TREE_NODE_PREFIX}L2`)
    expect(l1?.pId).toBe(LAYER_GRAPH_ROOT_ID)
    expect(l2?.pId).toBe(LAYER_GRAPH_ROOT_ID)
    expect(l2?.logicalParentLayerId).toBe('L1')
    const roots = simpleDataToTreeRoots(serial)
    expect(roots.length).toBe(1)
    expect(roots[0].nodeKind).toBe('virtual')
    expect(roots[0].children?.length).toBe(2)
    expect(roots[0].children[0].id).toBe(`${LAYER_TREE_NODE_PREFIX}L1`)
    expect(roots[0].children[1].id).toBe(`${LAYER_TREE_NODE_PREFIX}L2`)
  })

  it('同层多条非 clone 任务时任务行应挂在该层节点下', () => {
    const layers = [
      {
        layer_id: 'L-multi',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
      },
    ]
    const jobs = [
      {
        id: 'J1',
        layer_id: 'L-multi',
        status: 'completed',
        command_kind: 'trae',
        command: 'first',
        created_at: '2026-04-11T10:01:00Z',
      },
      {
        id: 'J2',
        layer_id: 'L-multi',
        status: 'completed',
        command_kind: 'trae',
        command: 'second',
        created_at: '2026-04-11T10:02:00Z',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, jobs)
    const j1 = serial.find((n) => n.id === 'J1')
    const j2 = serial.find((n) => n.id === 'J2')
    expect(j1?.pId).toBe(LAYER_GRAPH_ROOT_ID)
    expect(j2?.pId).toBe(LAYER_GRAPH_ROOT_ID)
    const roots = simpleDataToTreeRoots(serial)
    expect(roots.length).toBe(1)
    expect(roots[0].nodeKind).toBe('virtual')
    expect(roots[0].children?.length).toBe(3)
  })

  it('虚拟根子节点按创建时间串行排列（父层、子层、任务行混排）', () => {
    const layers = [
      {
        layer_id: 'L-parent',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        parent_layer_id: '',
      },
      {
        layer_id: 'L-child',
        created_at: '2026-04-11T10:01:00Z',
        git_worktree_dirty: false,
        parent_layer_id: 'L-parent',
      },
    ]
    /** 父层两条非 clone 任务，避免 soleVisible 合并导致无独立任务行 */
    const jobs = [
      {
        id: 'J-mid',
        layer_id: 'L-parent',
        status: 'completed',
        command_kind: 'trae',
        command: 'first',
        created_at: '2026-04-11T10:02:00Z',
      },
      {
        id: 'J-late',
        layer_id: 'L-parent',
        status: 'completed',
        command_kind: 'trae',
        command: 'after child',
        created_at: '2026-04-11T10:04:00Z',
      },
    ]
    const serial = buildZTreeNodesSerialFromLayers(layers, jobs)
    const roots = simpleDataToTreeRoots(serial)
    expect(roots.length).toBe(1)
    expect(roots[0].nodeKind).toBe('virtual')
    const ch = roots[0].children || []
    expect(ch.length).toBe(4)
    expect(ch[0].id).toBe(`${LAYER_TREE_NODE_PREFIX}L-parent`)
    expect(ch[1].nodeKind).toBe('layer')
    expect(ch[1].id).toBe(`${LAYER_TREE_NODE_PREFIX}L-child`)
    expect(ch[2].id).toBe('J-mid')
    expect(ch[3].id).toBe('J-late')
  })
})

describe('buildZTreeNodesFromLayers git_worktree_dirty 兼容类型', () => {
  it('字符串 "true" 也应允许提交按钮可点击', () => {
    const layers = [
      {
        layer_id: 'L1',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: 'true',
      },
    ]
    const jobs = [
      {
        id: 'J1',
        layer_id: 'L1',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]

    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L1')
    expect(layerNode).toBeTruthy()
    expect(layerNode.canSubmit).toBe(true)
    expect(layerNode.submitDisabled).toBe(false)
  })

  it('字符串 "false" 应视为已提交，有未推送提交时可推送且不可重复提交', () => {
    const layers = [
      {
        layer_id: 'L2',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: 'false',
        git_remote: { is_git: true, ahead: 1, no_upstream: false },
      },
    ]
    const jobs = [
      {
        id: 'J2',
        layer_id: 'L2',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]

    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L2')
    expect(layerNode).toBeTruthy()
    // 有 git 时始终展示「提交」，干净时禁用（与容器 UI / 文件变动列表面板对齐，避免「有变动却无按钮」）
    expect(layerNode.canSubmit).toBe(true)
    expect(layerNode.submitDisabled).toBe(true)
    expect(layerNode.canPush).toBe(true)
    expect(layerNode.pushDisabled).toBe(false)
  })

  it('工作区干净且无 git_remote 时仍应显示禁用的提交按钮', () => {
    const layers = [
      {
        layer_id: 'L-clean-submit',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
      },
    ]
    const jobs = [
      {
        id: 'J-clean-submit',
        layer_id: 'L-clean-submit',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesSerialFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L-clean-submit')
    expect(layerNode.canSubmit).toBe(true)
    expect(layerNode.submitDisabled).toBe(true)
    expect(layerNode.submitTitle).toMatch(/暂无未提交变更/)
  })

  it('层级节点行内应展示模型与总 token', () => {
    const layers = [
      {
        layer_id: 'L3',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: true,
      },
    ]
    const jobs = [
      {
        id: 'J3',
        layer_id: 'L3',
        status: 'completed',
        command_kind: 'trae',
        command: '实现功能',
        created_at: '2026-04-11T10:02:00Z',
        llm_model: 'claude-sonnet-4-20250514',
        llm_total_tokens: 128,
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L3')
    expect(layerNode).toBeTruthy()
    expect(layerNode.name).toContain('模型:claude-sonnet-4-20250514')
    expect(layerNode.name).toContain('token:128')
    expect(layerNode.title).toContain('用量: 模型:claude-sonnet-4-20250514 token:128')
  })

  it('git_remote.ahead 应出现在 pushAheadLabel（与 skill.md / GET /api/layers 一致）', () => {
    const layers = [
      {
        layer_id: 'L4',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: 3, no_upstream: false },
      },
    ]
    const jobs = [
      {
        id: 'J4',
        layer_id: 'L4',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L4')
    expect(layerNode.pushAheadLabel).toBe('3 个提交可推送')
    expect(layerNode.canPush).toBe(true)
    expect(layerNode.pushDisabled).toBe(false)
  })

  it('git_remote.last_push_error 应出现在 pushErrorLabel', () => {
    const layers = [
      {
        layer_id: 'L-err',
        created_at: '2026-08-22T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: {
          is_git: true,
          ahead: 1,
          no_upstream: false,
          last_push_error: '该仓库未找到可用的 OAuth access_token',
          last_push_error_trace_id: 'tid-push',
        },
      },
    ]
    const jobs = [
      {
        id: 'J-err',
        layer_id: 'L-err',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-08-22T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L-err')
    expect(layerNode.pushErrorLabel).toBe('push 失败')
    expect(layerNode.pushErrorTitle).toContain('OAuth access_token')
    expect(layerNode.pushErrorTraceId).toBe('tid-push')
    expect(layerNode.canSubmitAndPush).toBe(true)
  })

  it('git_remote.last_push_error 权限拒绝应显示 push 无权限', () => {
    const layers = [
      {
        layer_id: 'L-perm',
        created_at: '2026-08-29T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: {
          is_git: true,
          ahead: 1,
          no_upstream: false,
          last_push_error: 'remote: Permission to ruandao/helloworld.git denied to alice.',
          last_push_error_trace_id: 'tid-perm',
        },
      },
    ]
    const jobs = [
      {
        id: 'J-perm',
        layer_id: 'L-perm',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-08-29T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L-perm')
    expect(layerNode.pushErrorLabel).toBe('push 无权限')
    expect(layerNode.pushErrorKind).toBe('permission')
    expect(layerNode.pushErrorTitle).toContain('无写权限')
    expect(layerNode.pushErrorTraceId).toBe('tid-perm')
  })

  it('git_remote.ahead===0 且 last_pushed_count>0 时应显示已推送并隐藏推送按钮', () => {
    const layers = [
      {
        layer_id: 'L4b',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: 0, no_upstream: false, last_pushed_count: 2 },
      },
    ]
    const jobs = [
      {
        id: 'J4b',
        layer_id: 'L4b',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L4b')
    expect(layerNode.pushAheadLabel).toBe('2 个提交已推送')
    expect(layerNode.canPush).toBe(false)
    expect(layerNode.pushDisabled).toBe(true)
    expect(layerNode.canOpenPr).toBe(false)
  })

  it('git_remote.pr_html_url 存在时应显示 PR 按钮', () => {
    const layers = [
      {
        layer_id: 'L4pr',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: {
          is_git: true,
          ahead: 0,
          no_upstream: false,
          last_pushed_count: 1,
          pr_html_url: 'https://github.com/acme/repo/pull/9',
        },
      },
    ]
    const jobs = [
      {
        id: 'J4pr',
        layer_id: 'L4pr',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L4pr')
    expect(layerNode.canOpenPr).toBe(true)
    expect(layerNode.prHtmlUrl).toBe('https://github.com/acme/repo/pull/9')
    expect(layerNode.canPush).toBe(false)
  })

  it('git_remote.ahead===0 且无 last_pushed_count 时不显示推送按钮', () => {
    const layers = [
      {
        layer_id: 'L4c',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: 0, no_upstream: false },
      },
    ]
    const jobs = [
      {
        id: 'J4c',
        layer_id: 'L4c',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L4c')
    expect(layerNode.pushAheadLabel).toBe('')
    expect(layerNode.canPush).toBe(false)
  })

  it('git_remote.no_upstream 时应禁用推送并显示无上游', () => {
    const layers = [
      {
        layer_id: 'L5',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: null, no_upstream: true },
      },
    ]
    const jobs = [
      {
        id: 'J5',
        layer_id: 'L5',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesFromLayers(layers, jobs)
    const layerNode = findLayerNode(nodes, 'L5')
    expect(layerNode.pushAheadLabel).toBe('无上游')
    expect(layerNode.canPush).toBe(false)
    expect(layerNode.pushDisabled).toBe(true)
  })

  it('有 git 且干净时展示可点的合并到目标分支', () => {
    const layers = [
      {
        layer_id: 'L-merge',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: 0, current_branch: 'feature/x' },
      },
    ]
    const jobs = [
      {
        id: 'J-merge',
        layer_id: 'L-merge',
        status: 'completed',
        command_kind: 'trae',
        created_at: '2026-04-11T10:01:00Z',
      },
    ]
    const nodes = buildZTreeNodesSerialFromLayers(layers, jobs, {
      mergeTargetBranch: 'develop',
    })
    const layerNode = findLayerNode(nodes, 'L-merge')
    expect(layerNode.canMerge).toBe(true)
    expect(layerNode.mergeDisabled).toBe(false)
  })

  it('当前已在合并目标分支时不展示合并按钮', () => {
    const layers = [
      {
        layer_id: 'L-on-target',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: false,
        git_remote: { is_git: true, ahead: 0, current_branch: 'develop' },
      },
    ]
    const nodes = buildZTreeNodesSerialFromLayers(layers, [], {
      mergeTargetBranch: 'develop',
    })
    const layerNode = findLayerNode(nodes, 'L-on-target')
    expect(layerNode.canMerge).toBe(false)
  })

  it('工作区 dirty 时合并按钮禁用', () => {
    const layers = [
      {
        layer_id: 'L-dirty-merge',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: true,
        git_remote: { is_git: true, current_branch: 'feature/x' },
      },
    ]
    const jobs = []
    const nodes = buildZTreeNodesSerialFromLayers(layers, jobs, {
      mergeTargetBranch: 'develop',
    })
    const layerNode = findLayerNode(nodes, 'L-dirty-merge')
    expect(layerNode.canMerge).toBe(true)
    expect(layerNode.mergeDisabled).toBe(true)
    expect(layerNode.mergeTitle).toMatch(/请先提交/)
  })

  it('无 git 时不展示合并按钮', () => {
    const layers = [
      {
        layer_id: 'L-nogit',
        created_at: '2026-04-11T10:00:00Z',
        git_worktree_dirty: null,
      },
    ]
    const nodes = buildZTreeNodesSerialFromLayers(layers, [])
    const layerNode = findLayerNode(nodes, 'L-nogit')
    expect(layerNode.canMerge).toBe(false)
  })
})

describe('bootstrap_pending 引导空层锚点（OPT-20260820-002）', () => {
  const emptyLayer = {
    layer_id: 'L-boot',
    created_at: '2026-08-20T01:00:00Z',
    command: null,
    job_status: null,
    mind_state: 'pending',
    git_worktree_dirty: false,
    meta_kind: 'empty',
    bootstrap_pending: true,
  }

  it('串行视图输出「正在准备可写层」且 ztStyle=active（非 clean）', () => {
    const nodes = buildZTreeNodesSerialFromLayers([emptyLayer], [])
    const row = findLayerNode(nodes, 'L-boot')
    expect(row).toBeTruthy()
    expect(row.name).toContain('正在准备可写层')
    expect(row.ztStyle).toBe('active')
    expect(row.bootstrapAnchor).toBe(true)
  })

  it('父子树视图同样输出「正在准备可写层」', () => {
    const nodes = buildZTreeNodesFromLayers([emptyLayer], [])
    const row = findLayerNode(nodes, 'L-boot')
    expect(row).toBeTruthy()
    expect(row.name).toContain('正在准备可写层')
    expect(row.ztStyle).toBe('active')
  })

  it('仅 meta_kind=empty 无 bootstrap_pending 时也兜底为进行中', () => {
    const legacy = { ...emptyLayer, bootstrap_pending: undefined }
    const nodes = buildZTreeNodesSerialFromLayers([legacy], [])
    const row = findLayerNode(nodes, 'L-boot')
    expect(row.name).toContain('正在准备可写层')
    expect(row.ztStyle).toBe('active')
  })

  it('普通层不受影响（不显示「正在准备可写层」）', () => {
    const nodes = buildZTreeNodesSerialFromLayers(
      [{ layer_id: 'L-normal', created_at: '2026-08-20T02:00:00Z', command: 'echo hi', git_worktree_dirty: false, job_status: 'completed', mind_state: 'idle_done' }],
      [],
    )
    const row = findLayerNode(nodes, 'L-normal')
    expect(row.name).not.toContain('正在准备可写层')
  })

  it('bootstrap_failed 空层标记 bootstrapAnchor 且文案不是「正在准备可写层」', () => {
    const failed = {
      ...emptyLayer,
      bootstrap_pending: false,
      bootstrap_failed: true,
      mind_state: 'failed',
      job_status: 'failed',
      bootstrap_error: 'repo-clone-credentials 未返回完整',
    }
    const nodes = buildZTreeNodesSerialFromLayers([failed], [])
    const row = findLayerNode(nodes, 'L-boot')
    expect(row.bootstrapAnchor).toBe(true)
    expect(row.name).not.toContain('正在准备可写层')
    expect(row.name).toContain('repo-clone-credentials')
  })
})

describe('containerReleased 门控（OPT-20260822-001(3)）', () => {
  const gitLayer = {
    layer_id: 'L-rel',
    created_at: '2026-08-22T10:00:00Z',
    git_worktree_dirty: true,
    git_remote: { is_git: true, current_branch: 'feature/x', ahead: 2 },
  }
  const gitJob = {
    id: 'J-rel',
    layer_id: 'L-rel',
    status: 'running',
    command_kind: 'trae',
    created_at: '2026-08-22T10:01:00Z',
  }

  it('串行视图 containerReleased=true 时提交并创建PR 按钮禁用并带明确文案', () => {
    const nodes = buildZTreeNodesSerialFromLayers([gitLayer], [gitJob], {
      containerReleased: true,
    })
    const row = findLayerNode(nodes, 'L-rel')
    expect(row.containerReleased).toBe(true)
    expect(row.canSubmitAndPush).toBe(true)
    expect(row.submitAndPushDisabled).toBe(true)
    expect(row.submitAndPushTitle).toContain('服务器已释放')
    expect(row.submitAndPushTitle).toContain('提交并创建 PR')
    expect(row.pushDisabled).toBe(true)
    expect(row.pushTitle).toContain('服务器已释放')
    expect(row.submitDisabled).toBe(true)
    expect(row.submitTitle).toContain('服务器已释放')
    expect(row.mergeDisabled).toBe(true)
    expect(row.mergeTitle).toContain('服务器已释放')
  })

  it('父子树视图 containerReleased=true 同样禁用 git 动作', () => {
    const nodes = buildZTreeNodesFromLayers([gitLayer], [gitJob], {
      containerReleased: true,
    })
    const row = findLayerNode(nodes, 'L-rel')
    expect(row.containerReleased).toBe(true)
    expect(row.submitAndPushDisabled).toBe(true)
    expect(row.pushDisabled).toBe(true)
    expect(row.submitDisabled).toBe(true)
  })

  it('默认（未传 containerReleased）不影响既有按钮状态', () => {
    const cleanLayer = {
      ...gitLayer,
      layer_id: 'L-clean',
      git_worktree_dirty: false,
      git_remote: { is_git: true, current_branch: 'feature/x', ahead: 2 },
    }
    const nodes = buildZTreeNodesFromLayers([cleanLayer], [gitJob])
    const row = findLayerNode(nodes, 'L-clean')
    expect(row.containerReleased).toBeUndefined()
    // 干净层有 ahead 未推送：自然状态为提交并创建PR/推送可用，提交按钮因无可提交变更而禁用
    expect(row.submitAndPushDisabled).toBe(false)
    expect(row.pushDisabled).toBe(false)
    expect(row.submitDisabled).toBe(true)
  })

  it('无 job 时 containerReleased 空节点分支同样置禁用标志', () => {
    const nodes = buildZTreeNodesFromLayers([gitLayer], [], {
      containerReleased: true,
    })
    const row = findLayerNode(nodes, 'L-rel')
    expect(row.containerReleased).toBe(true)
    expect(row.submitAndPushDisabled).toBe(true)
    expect(row.pushDisabled).toBe(true)
    expect(row.submitDisabled).toBe(true)
  })
})
