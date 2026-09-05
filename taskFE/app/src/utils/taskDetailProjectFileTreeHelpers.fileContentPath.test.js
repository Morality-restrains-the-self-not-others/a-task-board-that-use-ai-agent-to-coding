import { describe, expect, it } from 'vitest'
import { appendCommentIdQuery, fileContentPathCandidates } from './taskDetailProjectFileTreeHelpers.js'

describe('fileContentPathCandidates', () => {
  it('原路径优先，并为无前缀路径补上仓库前缀', () => {
    expect(fileContentPathCandidates('README.md', ['ram-work'])).toEqual([
      'README.md',
      'ram-work/README.md',
    ])
  })

  it('已带任一仓库前缀时不再拼接', () => {
    expect(fileContentPathCandidates('ram-work/docs/a.md', ['ram-work', 'other'])).toEqual([
      'ram-work/docs/a.md',
    ])
  })

  it('多仓无前缀时按各仓库前缀生成候选', () => {
    expect(fileContentPathCandidates('a.txt', ['goPractice', 'otherRepo'])).toEqual([
      'a.txt',
      'goPractice/a.txt',
      'otherRepo/a.txt',
    ])
  })

  it('空路径返回空列表', () => {
    expect(fileContentPathCandidates('', ['ram-work'])).toEqual([])
  })
})

describe('appendCommentIdPath', () => {
  it('有 comment_id 时追加到 path', () => {
    expect(appendCommentIdQuery('/api/x/?layer_id=1', 'cmt_9')).toBe('/api/x/comment_id/cmt_9/?layer_id=1')
  })

  it('无 comment_id 时保持原路径', () => {
    expect(appendCommentIdQuery('/api/x/?layer_id=1', '')).toBe('/api/x/?layer_id=1')
  })
})
