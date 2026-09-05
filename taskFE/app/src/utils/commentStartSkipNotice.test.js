import { describe, expect, it } from 'vitest'
import { splitCommentStartSkipNotice } from './commentStartSkipNotice.js'

describe('splitCommentStartSkipNotice', () => {
  it('returns empty parts for empty content', () => {
    expect(splitCommentStartSkipNotice('')).toEqual({ body: '', skipNotice: '' })
    expect(splitCommentStartSkipNotice(null)).toEqual({ body: '', skipNotice: '' })
  })

  it('keeps ordinary comments unchanged', () => {
    const raw = '【自动运行】\n将当前时间，硬编码进 now.md 文件中\n$trae-agent-skill'
    expect(splitCommentStartSkipNotice(raw)).toEqual({ body: raw, skipNotice: '' })
  })

  it('extracts 未启动服务器 nested-git auth notice from auto-run comment', () => {
    const body = '【自动运行】\n将当前时间，硬编码进 now.md 文件中\n$trae-agent-skill /general-coding'
    const notice = '未启动服务器：无法获取子 Git 仓库列表：未检测到可用授权。请先在个人资料完成 Git 网站绑定，或先登录本地 GitLab 后重试。'
    const got = splitCommentStartSkipNotice(`${body}\n\n${notice}`)
    expect(got.body).toBe(body)
    expect(got.skipNotice).toBe(notice)
  })
})
