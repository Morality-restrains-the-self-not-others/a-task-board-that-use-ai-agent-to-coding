// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  parseCommentCloneProgressFromLogs,
  shouldShowCloneManualRetry,
  overallCloneProgressPct,
} from './commentCloneProgressFromLogs.js'
import {
  collectCloneRepoCatalog,
  enrichCloneProgressRowsWithCatalog,
  resolveCommentCloneProgressWithCatalog,
  omitSkippedNestedCloneProgressRows,
} from './commentCloneProgressRepoCatalog.js'

const USER_FAIL_LINE =
  "【项目克隆】(1/1) 失败 ram-work: git exit 128: Cloning into '/app/onlineProject_state/layers/20260817_062029_f94295/ram-work'…"

describe('collectCloneRepoCatalog', () => {
  it('collects git_repos and git_repo_entries including nested parent/alias', () => {
    const catalog = collectCloneRepoCatalog({
      projects: [
        {
          project: {
            git_repos: ['https://gitlab.daydaymoney.com/g/ram-work.git'],
            git_repo_entries: [
              { url: 'https://gitlab.daydaymoney.com/g/ram-work.git' },
              {
                url: 'https://gitlab.daydaymoney.com/g/docs.git',
                parent_repo_url: 'https://gitlab.daydaymoney.com/g/ram-work.git',
                clone_alias: 'docs',
              },
            ],
          },
        },
      ],
    })
    expect(catalog.map((e) => e.repoUrl)).toEqual([
      'https://gitlab.daydaymoney.com/g/ram-work.git',
      'https://gitlab.daydaymoney.com/g/docs.git',
    ])
    const nested = catalog.find((e) => e.repoUrl.includes('docs.git'))
    expect(nested.parentRepoUrl).toBe('https://gitlab.daydaymoney.com/g/ram-work.git')
    expect(nested.cloneAlias).toBe('docs')
  })

  it('parses ━━ section URLs from bootstrap clone log', () => {
    const catalog = collectCloneRepoCatalog({
      bootstrapLog: '━━ (1/1) https://gitlab.daydaymoney.com/g/ram-work.git\n→ ram-work\n',
    })
    expect(catalog).toHaveLength(1)
    expect(catalog[0].repoUrl).toBe('https://gitlab.daydaymoney.com/g/ram-work.git')
  })
})

describe('enrichCloneProgressRowsWithCatalog', () => {
  it('T24: cold-open (1/1) 失败 ram-work without URL still shows 手动重试 after catalog match', () => {
    const parsed = parseCommentCloneProgressFromLogs([`[14:20:11] ${USER_FAIL_LINE}`])
    expect(parsed).toHaveLength(1)
    expect(parsed[0].failed).toBe(true)
    expect(parsed[0].repoUrl).toBe('')
    expect(shouldShowCloneManualRetry(parsed[0])).toBe(false)

    const enriched = enrichCloneProgressRowsWithCatalog(parsed, collectCloneRepoCatalog({
      projects: [{ git_repos: ['https://gitlab.daydaymoney.com/g/ram-work.git'] }],
    }))
    expect(enriched[0].repoUrl).toBe('https://gitlab.daydaymoney.com/g/ram-work.git')
    expect(shouldShowCloneManualRetry(enriched[0])).toBe(true)
  })

  it('T25: 1/1 unique catalog entry fills URL even when label is only the directory name', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[14:20:11] 【项目克隆】(1/1) 失败 ram-work: git exit 128',
    ])
    const enriched = enrichCloneProgressRowsWithCatalog(rows, [
      { repoUrl: 'https://git.example/org/ram-work.git', parentRepoUrl: '', cloneAlias: '' },
    ])
    expect(shouldShowCloneManualRetry(enriched[0])).toBe(true)
    expect(enriched[0].repoUrl).toBe('https://git.example/org/ram-work.git')
  })

  it('T26: nested catalog match attaches parentRepoUrl and cloneAlias', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[14:20:11] 【项目克隆】(2/2) 失败 docs: git exit 128',
    ])
    const enriched = enrichCloneProgressRowsWithCatalog(rows, [
      {
        repoUrl: 'https://gitlab.daydaymoney.com/g/docs.git',
        parentRepoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
        cloneAlias: 'docs',
      },
    ])
    expect(enriched[0].repoUrl).toBe('https://gitlab.daydaymoney.com/g/docs.git')
    expect(enriched[0].parentRepoUrl).toBe('https://gitlab.daydaymoney.com/g/ram-work.git')
    expect(enriched[0].cloneAlias).toBe('docs')
    expect(shouldShowCloneManualRetry(enriched[0])).toBe(true)
  })

  it('T27: ambiguous basename does not guess a URL', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[14:20:11] 【项目克隆】(1/2) 失败 ram-work: git exit 128',
    ])
    const enriched = enrichCloneProgressRowsWithCatalog(rows, [
      { repoUrl: 'https://git.example/a/ram-work.git', parentRepoUrl: '', cloneAlias: '' },
      { repoUrl: 'https://git.example/b/ram-work.git', parentRepoUrl: '', cloneAlias: '' },
    ])
    expect(enriched[0].repoUrl).toBe('')
    expect(shouldShowCloneManualRetry(enriched[0])).toBe(false)
  })

  it('keeps auto-retry rows without a manual button after enrich', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[00:51:02] 【项目克隆】(1/1) 网络抖动，准备第 2/3 次重试…',
    ])
    const enriched = enrichCloneProgressRowsWithCatalog(rows, [
      { repoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git', parentRepoUrl: '', cloneAlias: '' },
    ])
    expect(enriched[0].retrying).toBe(true)
    expect(shouldShowCloneManualRetry(enriched[0])).toBe(false)
  })
})

describe('parse fail line with embedded git URL', () => {
  it('T28: failure message that already contains a git URL sets repoUrl', () => {
    const rows = parseCommentCloneProgressFromLogs([
      '[14:20:11] 【项目克隆】(1/1) 失败 ram-work https://gitlab.daydaymoney.com/g/ram-work.git: git exit 128',
    ])
    expect(rows[0].failed).toBe(true)
    expect(rows[0].label).toBe('ram-work')
    expect(rows[0].repoUrl).toBe('https://gitlab.daydaymoney.com/g/ram-work.git')
    expect(shouldShowCloneManualRetry(rows[0])).toBe(true)
  })
})

describe('resolveCommentCloneProgressWithCatalog', () => {
  it('wires parse + catalog so cold-open comment logs get a retry URL', () => {
    const rows = resolveCommentCloneProgressWithCatalog(
      {
        commentId: 'C1',
        statusLogs: [`[14:20:11] ${USER_FAIL_LINE}`],
        byCommentId: {},
        taskLevelEntries: [],
        soleActiveCommentId: 'C1',
      },
      {
        projects: [{ git_repos: ['https://gitlab.daydaymoney.com/g/ram-work.git'] }],
      },
    )
    expect(rows).toHaveLength(1)
    expect(shouldShowCloneManualRetry(rows[0])).toBe(true)
    expect(rows[0].repoUrl).toContain('ram-work.git')
  })

  it('T35: live 9% plus bootstrap 克隆完成 becomes 100%', () => {
    const rows = resolveCommentCloneProgressWithCatalog(
      {
        commentId: 'C1',
        statusLogs: ['[18:45:10] 【项目克隆】(1/1) ram-work … 9%'],
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
      },
      {
        bootstrapLog: '━━ (1/1) https://git.example/ram-work.git\n【项目克隆】克隆完成。\n',
      },
    )
    expect(rows).toHaveLength(1)
    expect(rows[0].progress).toBe(100)
  })
})

describe('omitSkippedNestedCloneProgressRows', () => {
  it('T37: auto_clone off drops nested rows so overall is not 100/34', () => {
    const rows = [
      {
        key: 'https://git.example/ram-work.git',
        label: 'ram-work',
        progress: 100,
        repoTotal: 1,
        parentRepoUrl: '',
      },
      {
        key: 'https://git.example/task2app.git',
        label: 'task2app',
        progress: 0,
        repoTotal: 34,
        parentRepoUrl: 'https://git.example/ram-work.git',
      },
    ]
    const filtered = omitSkippedNestedCloneProgressRows(rows, { autoCloneNestedRepos: false })
    expect(filtered).toHaveLength(1)
    expect(filtered[0].label).toBe('ram-work')
    expect(overallCloneProgressPct(filtered)).toBe(100)
  })

  it('keeps nested rows when auto clone nested is on', () => {
    const rows = [
      { label: 'ram-work', progress: 100, parentRepoUrl: '' },
      { label: 'docs', progress: 0, parentRepoUrl: 'https://git.example/ram-work.git' },
    ]
    expect(omitSkippedNestedCloneProgressRows(rows, { autoCloneNestedRepos: true })).toHaveLength(2)
  })
})
