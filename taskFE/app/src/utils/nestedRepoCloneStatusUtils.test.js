// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { gitCloneRefMatchKey } from './taskDetailContainerCloneProgress.js'
import {
  formatNestedRepoCloneErrorDetail,
  lookupBootstrapLogSectionText,
  lookupCloneProgressEntry,
  nestedRepoCloneStatusBadgeClass,
  resolveNestedRepoCloneStatus,
  shouldShowNestedRepoRecloneButton,
  summarizeNestedRepoCloneStatuses,
} from './nestedRepoCloneStatusUtils.js'

describe('resolveNestedRepoCloneStatus', () => {
  it('容器未就绪且无进度 → waiting_container', () => {
    expect(resolveNestedRepoCloneStatus(null, { containerReady: false })).toEqual({
      kind: 'waiting_container',
      label: '等待容器',
      progress: 0,
      message: '',
    })
  })

  it('容器就绪无进度 → idle', () => {
    expect(resolveNestedRepoCloneStatus(null, { containerReady: true }).kind).toBe('idle')
  })

  it('引导日志完成且无进度行 → done', () => {
    const st = resolveNestedRepoCloneStatus(null, {
      containerReady: true,
      bootstrapCloneDone: true,
    })
    expect(st.kind).toBe('done')
    expect(st.label).toBe('已完成')
  })

  it('无进度但日志含已移入 → done（staging 移入父仓后进度 Map 已清空）', () => {
    const st = resolveNestedRepoCloneStatus(null, {
      containerReady: true,
      bootstrapCloneDone: false,
      bootstrapLogSectionText:
        '━━ (2/3) https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git\n→ parent/DaydaymoneyGrafana (via staging)\n[bootstrap-clone] 已移入 parent/DaydaymoneyGrafana',
    })
    expect(st.kind).toBe('done')
    expect(st.label).toBe('已完成')
  })

  it('无进度但日志含克隆失败 → error', () => {
    const st = resolveNestedRepoCloneStatus(null, {
      containerReady: true,
      bootstrapLogSectionText: '[bootstrap-clone] 克隆失败: fatal: repository not found',
    })
    expect(st.kind).toBe('error')
  })

  it('进度 100 → done', () => {
    expect(
      resolveNestedRepoCloneStatus({ progress: 100, message: 'ok' }, { containerReady: true }).kind,
    ).toBe('done')
  })

  it('进度中 → running', () => {
    const st = resolveNestedRepoCloneStatus(
      { progress: 42, message: 'Receiving objects' },
      { containerReady: true },
    )
    expect(st.kind).toBe('running')
    expect(st.progress).toBe(42)
  })

  it('失败文案且未完成 → error', () => {
    const st = resolveNestedRepoCloneStatus(
      { progress: 10, message: 'fatal: Authentication failed' },
      { containerReady: true },
    )
    expect(st.kind).toBe('error')
    expect(st.label).toBe('克隆失败')
  })

  it('「未完成」文案优先于 progress>=100 → error', () => {
    const st = resolveNestedRepoCloneStatus(
      { progress: 100, message: '【项目克隆】未完成：失败 2 个仓库：relayToTrae、scripts' },
      { containerReady: true },
    )
    expect(st.kind).toBe('error')
    expect(st.progress).toBe(0)
  })

  it('进度 0 且无文案 → queued', () => {
    expect(
      resolveNestedRepoCloneStatus({ progress: 0, message: '' }, { containerReady: true }).kind,
    ).toBe('queued')
  })
})

describe('lookupCloneProgressEntry', () => {
  it('支持 Map 与 Object', () => {
    const url = 'https://gitlab.daydaymoney.com/g/docs.git'
    const key = gitCloneRefMatchKey(url)
    const map = new Map([[key, { progress: 50, message: 'm' }]])
    expect(lookupCloneProgressEntry(map, gitCloneRefMatchKey, url).progress).toBe(50)
    expect(
      lookupCloneProgressEntry({ [key]: { progress: 80 } }, gitCloneRefMatchKey, url).progress,
    ).toBe(80)
  })
})

describe('lookupBootstrapLogSectionText', () => {
  it('从全文分段按 URL 匹配段落', () => {
    const url = 'https://gitlab.daydaymoney.com/g/AiMonitor.git'
    const full = [
      '【项目克隆】正在并行克隆…',
      '',
      `━━ (1/2) ${url}`,
      '→ parent/AiMonitor (via staging)',
      '[bootstrap-clone] 已移入 parent/AiMonitor',
      '',
      '【项目克隆】克隆完成。',
    ].join('\n')
    const text = lookupBootstrapLogSectionText(full, null, gitCloneRefMatchKey, url)
    expect(text).toContain('已移入 parent/AiMonitor')
  })
})

describe('summarizeNestedRepoCloneStatuses', () => {
  it('统计 done/running/error/idle', () => {
    const parentKey = gitCloneRefMatchKey('https://gitlab.daydaymoney.com/g/docs.git')
    const map = new Map([
      [parentKey, { progress: 100, message: 'done' }],
      [gitCloneRefMatchKey('https://gitlab.daydaymoney.com/g/task2app.git'), { progress: 30, message: 'x' }],
      [
        gitCloneRefMatchKey('https://gitlab.daydaymoney.com/g/bad.git'),
        { progress: 5, message: 'clone failed' },
      ],
    ])
    const summary = summarizeNestedRepoCloneStatuses(
      [
        { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git' },
        { path: 'task2app', url: 'https://gitlab.daydaymoney.com/g/task2app.git' },
        { path: 'bad', url: 'https://gitlab.daydaymoney.com/g/bad.git' },
        { path: 'idle', url: 'https://gitlab.daydaymoney.com/g/idle.git' },
        { path: 'no-url', url: '' },
      ],
      map,
      gitCloneRefMatchKey,
      { containerReady: true },
    )
    expect(summary).toEqual({ total: 4, done: 1, running: 1, error: 1, idle: 1 })
  })
})

describe('nestedRepoCloneStatusBadgeClass', () => {
  it('done 使用绿色样式', () => {
    expect(nestedRepoCloneStatusBadgeClass('done')).toContain('emerald')
  })
})

describe('formatNestedRepoCloneErrorDetail', () => {
  it('抽取 bootstrap-clone 失败后缀', () => {
    expect(
      formatNestedRepoCloneErrorDetail('[bootstrap-clone] 克隆失败: remote: Not Found'),
    ).toBe('remote: Not Found')
  })

  it('抽取编号失败行中的具体错误', () => {
    expect(
      formatNestedRepoCloneErrorDetail(
        '【项目克隆】(3/10) 失败 relayToTrae: fatal: repository not found',
      ),
    ).toContain('fatal: repository not found')
  })

  it('空文案返回空', () => {
    expect(formatNestedRepoCloneErrorDetail('')).toBe('')
  })
})

describe('shouldShowNestedRepoRecloneButton', () => {
  it('失败态始终展示', () => {
    expect(shouldShowNestedRepoRecloneButton('error', { hasUrl: true })).toBe(true)
  })

  it('容器就绪的 idle 展示', () => {
    expect(
      shouldShowNestedRepoRecloneButton('idle', { containerReady: true, hasUrl: true }),
    ).toBe(true)
  })

  it('无 URL 不展示', () => {
    expect(shouldShowNestedRepoRecloneButton('error', { hasUrl: false })).toBe(false)
  })
})
