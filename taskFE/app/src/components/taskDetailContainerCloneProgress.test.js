// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  extractBootstrapCloneFailedRepoUrls,
  isCloneProgressFailureMessage,
  parseBootstrapCloneLogSections,
} from '../utils/taskDetailContainerCloneProgress.js'

const SAMPLE_FAIL_LOG = `
【项目克隆】正在并行克隆任务关联仓库（任务详情已拉取）…

━━ (14/36) https://gitlab.daydaymoney.com/example-user/relayToTrae.git
→ relayToTrae
Cloning into '/app/onlineProject_state/layers/x/relayToTrae'...
remote: The project you were looking for could not be found or you don't have permission to view it.
fatal: repository 'https://gitlab.daydaymoney.com/example-user/relayToTrae.git/' not found

[bootstrap-clone] 克隆失败: git exit 128: Cloning into ...

━━ (15/36) https://gitlab.daydaymoney.com/example-user/runAll.git
→ runAll
Receiving objects: 100% (10/10), done.

━━ (16/36) https://gitlab.daydaymoney.com/example-user/scripts.git
→ scripts
fatal: repository 'https://gitlab.daydaymoney.com/example-user/scripts.git/' not found

[bootstrap-clone] 克隆失败: git exit 128

【项目克隆】已结束（存在失败）。
失败仓库（2）：
- relayToTrae — https://gitlab.daydaymoney.com/example-user/relayToTrae.git（git exit 128）
- scripts — https://gitlab.daydaymoney.com/example-user/scripts.git（fatal: not found）
`

describe('parseBootstrapCloneLogSections', () => {
  it('splits by ━━ (i/n) url headers', () => {
    const { sections } = parseBootstrapCloneLogSections(SAMPLE_FAIL_LOG)
    expect(sections.length).toBe(3)
    expect(sections[0].url).toContain('relayToTrae.git')
    expect(sections[1].url).toContain('runAll.git')
    expect(sections[2].url).toContain('scripts.git')
  })
})

describe('extractBootstrapCloneFailedRepoUrls', () => {
  it('extracts failed urls from section fail notes and footer lines', () => {
    const urls = extractBootstrapCloneFailedRepoUrls(SAMPLE_FAIL_LOG)
    expect(urls).toEqual(
      expect.arrayContaining([
        'https://gitlab.daydaymoney.com/example-user/relayToTrae.git',
        'https://gitlab.daydaymoney.com/example-user/scripts.git',
      ]),
    )
    expect(urls).toHaveLength(2)
    expect(urls.some((u) => u.includes('runAll'))).toBe(false)
  })

  it('extracts from footer-only lines when sections lack fail markers', () => {
    const text = `
【项目克隆】已结束（存在失败）。
失败仓库（1）：
- foo — https://gitlab.daydaymoney.com/g/foo.git（boom）
`
    expect(extractBootstrapCloneFailedRepoUrls(text)).toEqual([
      'https://gitlab.daydaymoney.com/g/foo.git',
    ])
  })
})

describe('isCloneProgressFailureMessage', () => {
  it('matches 失败 / 未完成 / fatal', () => {
    expect(isCloneProgressFailureMessage('【项目克隆】失败: x')).toBe(true)
    expect(isCloneProgressFailureMessage('【项目克隆】未完成：失败 2 个仓库')).toBe(true)
    expect(isCloneProgressFailureMessage('fatal: not found')).toBe(true)
    expect(isCloneProgressFailureMessage('【项目克隆】克隆完成')).toBe(false)
  })
})
