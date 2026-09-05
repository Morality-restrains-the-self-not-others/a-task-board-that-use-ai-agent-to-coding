// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  parseCommentCloneProgressFromLogs,
  cloneProgressRowsFromLiveMap,
  cloneProgressRowsFromEntries,
  resolveCommentCloneProgress,
  hasActiveCommentCloneProgress,
  overallCloneProgressPct,
  expectedCloneRepoTotal,
  shouldShowCloneManualRetry,
  mergeCommentCloneProgressRows,
  omitRedundantGlobalCloneRow,
  applyBootstrapCloneDoneToRows,
  looksLikeGitRepoRef,
} from './commentCloneProgressFromLogs.js'

describe('parseCommentCloneProgressFromLogs', () => {
  it('T1: keeps latest percent for the same repo', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/2) alpha … 33%',
      '[18:45:12] 【项目克隆】(1/2) alpha … 66%',
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].label).toBe('alpha')
    expect(rows[0].progress).toBe(66)
    expect(rows[0].failed).toBe(false)
  })

  it('T2: splits two repos', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/2) alpha … 33%',
      '[18:45:11] 【项目克隆】(2/2) beta … 10%',
      '[18:45:12] 【项目克隆】(1/2) alpha … 90%',
    ])
    expect(rows.map((r) => [r.label, r.progress])).toEqual([
      ['alpha', 90],
      ['beta', 10],
    ])
  })

  it('T3: completion line without percent sets 100', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:46:00] 项目克隆 (1/1) 完成 alpha',
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].label).toBe('alpha')
    expect(rows[0].progress).toBe(100)
  })

  it('T4: failure line marks failed', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:46:01] 【项目克隆】(1/2) 失败 relayToTrae: fatal: repository not found',
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].label).toBe('relayToTrae')
    expect(rows[0].failed).toBe(true)
  })

  it('T5: scheduling-only logs yield empty', () => {
    expect(parseCommentCloneProgressFromLogs([
      '[18:44:33] 容器调度排队中',
      '[18:44:33] 正在启动容器实例',
      '[18:44:39] aliyun服务器启动成功！',
    ])).toEqual([])
  })

  it('strips trace_id suffix and uses global done to fill remaining', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/1) alpha … 40% trace_id=abc',
      '[18:45:20] 【项目克隆】克隆完成。',
    ])
    expect(rows[0].progress).toBe(100)
  })

  it('global start message without percent yields a single 0% row', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:00] 【项目克隆】开始并行克隆任务关联仓库…',
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].key).toBe('__global__')
    expect(rows[0].progress).toBe(0)
    expect(rows[0].label).toBe('项目克隆')
  })
})

describe('resolveCommentCloneProgress', () => {
  it('T6: live map for the comment wins over older logs', () => {
    const rows = resolveCommentCloneProgress({
      commentId: 'C1',
      statusLogs: ['[18:45:10] 【项目克隆】(1/1) alpha … 10%'],
      byCommentId: {
        C1: {
          'https://git.example/alpha.git': { progress: 55, message: 'live 55' },
        },
      },
      taskLevelEntries: [],
      soleActiveCommentId: 'C1',
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].progress).toBe(55)
    expect(rows[0].message).toBe('live 55')
  })

  it('T7: without live map, sole-active comment uses task-level entries', () => {
    const rows = resolveCommentCloneProgress({
      commentId: 'C1',
      statusLogs: [],
      byCommentId: {},
      taskLevelEntries: [
        { key: 'https://git.example/alpha.git', repoUrl: 'https://git.example/alpha.git', progress: 41, message: 'sse' },
      ],
      soleActiveCommentId: 'C1',
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].progress).toBe(41)
    expect(rows[0].label).toContain('alpha')
  })

  it('does not leak task-level entries onto a non-active sibling comment', () => {
    const rows = resolveCommentCloneProgress({
      commentId: 'C2',
      statusLogs: [],
      byCommentId: {},
      taskLevelEntries: [
        { key: 'https://git.example/alpha.git', repoUrl: 'https://git.example/alpha.git', progress: 41, message: 'sse' },
      ],
      soleActiveCommentId: 'C1',
    })
    expect(rows).toEqual([])
  })

  it('empty commentId falls back to task-level entries', () => {
    const rows = resolveCommentCloneProgress({
      commentId: '',
      statusLogs: [],
      byCommentId: {},
      taskLevelEntries: [
        { key: '__global__', repoUrl: '', progress: 12, message: 'boot' },
      ],
      soleActiveCommentId: '',
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].progress).toBe(12)
  })
})

describe('hasActiveCommentCloneProgress', () => {
  it('true when at least one comment has a non-empty live map', () => {
    expect(
      hasActiveCommentCloneProgress({
        C1: { 'https://git.example/alpha.git': { progress: 55, message: 'live' } },
        C2: {},
      }),
    ).toBe(true)
  })

  it('false when all comment maps are empty / missing', () => {
    expect(hasActiveCommentCloneProgress({ C1: {}, C2: {} })).toBe(false)
    expect(hasActiveCommentCloneProgress({})).toBe(false)
    expect(hasActiveCommentCloneProgress(null)).toBe(false)
    expect(hasActiveCommentCloneProgress(undefined)).toBe(false)
  })
})

describe('cloneProgressRowsFromLiveMap / entries', () => {
  it('maps __global__ label to 项目克隆', () => {
    const rows = cloneProgressRowsFromLiveMap({
      __global__: { progress: 8, message: '开始' },
    })
    expect(rows[0].label).toBe('项目克隆')
    expect(rows[0].progress).toBe(8)
  })

  it('maps entries via short clone label', () => {
    const rows = cloneProgressRowsFromEntries([
      {
        key: 'https://git.example/group/repo.git',
        repoUrl: 'https://git.example/group/repo.git',
        progress: 70,
        message: 'm',
        recvProgress: 80,
        unpackProgress: 20,
      },
    ])
    expect(rows[0].label).toBe('group/repo')
    expect(rows[0].recvProgress).toBe(80)
    expect(rows[0].unpackProgress).toBe(20)
    expect(rows[0].repoUrl).toBe('https://git.example/group/repo.git')
  })
})

describe('overall clone progress and retry metadata', () => {
  it('T12: overall uses (i/n) total as denominator even when only some rows exist', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[00:51:01] 【项目克隆】(31/34) doneRepo … 100%',
      '[00:51:02] 【项目克隆】(32/34) 网络抖动，准备第 2/3 次重试…',
    ])
    expect(expectedCloneRepoTotal(rows)).toBe(34)
    expect(overallCloneProgressPct(rows)).toBe(Math.round(100 / 34))
  })

  it('T13: retry line is not failed and keeps prior percent', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[00:51:00] 【项目克隆】(32/34) taskTenantService … 40%',
      '[00:51:02] 【项目克隆】(32/34) 网络抖动，准备第 2/3 次重试…',
    ])
    const row = rows.find((r) => r.retrying) || rows[0]
    expect(row.retrying).toBe(true)
    expect(row.failed).toBe(false)
    expect(row.retryAttempt).toBe(2)
    expect(row.retryMax).toBe(3)
    expect(row.progress).toBe(40)
    expect(shouldShowCloneManualRetry(row)).toBe(false)
  })

  it('T14: after exhausted failure, row is failed and retry needs git URL', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[00:51:02] 【项目克隆】(32/34) 网络抖动，准备第 3/3 次重试…',
      '[00:51:08] 【项目克隆】(32/34) 失败 taskTenantService: fatal: could not read from remote repository',
    ])
    const row = rows.find((r) => r.failed)
    expect(row).toBeTruthy()
    expect(row.retrying).toBe(false)
    expect(shouldShowCloneManualRetry(row)).toBe(false)
    expect(shouldShowCloneManualRetry({
      ...row,
      repoUrl: 'https://git.example/taskTenantService.git',
    })).toBe(true)
  })

  it('omits global summary row when per-repo rows already exist', () => {
    const rows = omitRedundantGlobalCloneRow([
      {
        key: '__global__',
        label: '项目克隆',
        progress: 0,
        message: '【项目克隆】部分失败：失败 1/1 个仓库：ram-work（其余已就绪，引导继续）',
        failed: true,
        repoUrl: '',
      },
      {
        key: 'https://git.example/ram-work.git',
        label: 'ram-work',
        progress: 0,
        message: '【项目克隆】(1/1) 失败 ram-work: git exit 128',
        failed: true,
        repoUrl: 'https://git.example/ram-work.git',
      },
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].label).toBe('ram-work')
    expect(shouldShowCloneManualRetry(rows[0])).toBe(true)
  })

  it('T15: live map merges with logs so completed repos stay in overall', () => {
    const rows = resolveCommentCloneProgress({
      commentId: 'C1',
      statusLogs: [
        '[18:45:10] 【项目克隆】(1/2) alpha … 100%',
        '[18:45:11] 【项目克隆】(2/2) beta … 10%',
      ],
      byCommentId: {
        C1: {
          'https://git.example/beta.git': { progress: 55, message: '【项目克隆】(2/2) beta … 55%' },
        },
      },
      taskLevelEntries: [],
      soleActiveCommentId: 'C1',
    })
    expect(rows).toHaveLength(2)
    const byLabel = Object.fromEntries(rows.map((r) => [r.label, r.progress]))
    expect(byLabel.alpha).toBe(100)
    expect(byLabel.beta).toBe(55)
    expect(overallCloneProgressPct(rows)).toBe(78)
  })

  it('T30: later percent line cannot pull a completed repo back from 100', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/1) ram-work … 9%',
      '[18:46:00] 项目克隆 (1/1) 完成 ram-work',
      '[18:46:01] 【项目克隆】(1/1) ram-work … 9%',
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].label).toBe('ram-work')
    expect(rows[0].progress).toBe(100)
    expect(rows[0].message).toContain('完成')
  })

  it('T31: live 9% does not overlay log 完成 100%', () => {
    const rows = resolveCommentCloneProgress({
      commentId: 'C1',
      statusLogs: [
        '[18:45:10] 【项目克隆】(1/1) ram-work … 9%',
        '[18:46:00] 项目克隆 (1/1) 完成 ram-work',
      ],
      byCommentId: {
        C1: {
          'https://git.example/ram-work.git': {
            progress: 9,
            message: '【项目克隆】(1/1) ram-work … 9%',
          },
        },
      },
      taskLevelEntries: [],
      soleActiveCommentId: 'C1',
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].progress).toBe(100)
    expect(rows[0].message).toContain('完成')
  })

  it('merge prefers live retrying row but keeps previous progress when live is 0', () => {
    const merged = mergeCommentCloneProgressRows(
      [{ key: 'alpha', label: 'alpha', progress: 66, message: '… 66%', failed: false, repoTotal: 2 }],
      [{
        key: 'https://git.example/alpha.git',
        label: 'alpha',
        progress: 0,
        message: '【项目克隆】(1/2) 网络抖动，准备第 2/3 次重试…',
        failed: false,
        retrying: true,
        retryAttempt: 2,
        retryMax: 3,
        repoTotal: 2,
        repoUrl: 'https://git.example/alpha.git',
      }],
    )
    expect(merged).toHaveLength(1)
    expect(merged[0].retrying).toBe(true)
    expect(merged[0].progress).toBe(66)
    expect(merged[0].repoUrl).toContain('git.example/alpha')
  })
})

describe('applyBootstrapCloneDoneToRows', () => {
  it('T35: bootstrap 克隆完成 snaps a 9% log row to 100%', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/1) ram-work … 9%',
    ])
    const next = applyBootstrapCloneDoneToRows(
      rows,
      'Resolving deltas: 100% (1/1), done.\n\n【项目克隆】克隆完成。\n',
    )
    expect(next).toHaveLength(1)
    expect(next[0].progress).toBe(100)
    expect(next[0].recvProgress).toBe(100)
    expect(next[0].unpackProgress).toBe(100)
  })

  it('does not snap failed rows when bootstrap log is done', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[18:45:10] 【项目克隆】(1/1) 失败 ram-work: git exit 128',
    ])
    const next = applyBootstrapCloneDoneToRows(rows, '【项目克隆】克隆完成。')
    expect(next[0].failed).toBe(true)
    expect(next[0].progress).not.toBe(100)
  })
})

describe('looksLikeGitRepoRef', () => {
  it('accepts ssh:// git remotes', () => {
    expect(looksLikeGitRepoRef('ssh://git@github.com/owner/repo.git')).toBe(true)
    expect(looksLikeGitRepoRef('git@github.com:owner/repo.git')).toBe(true)
    expect(looksLikeGitRepoRef('https://github.com/owner/repo.git')).toBe(true)
    expect(looksLikeGitRepoRef('not-a-repo')).toBe(false)
  })
})

