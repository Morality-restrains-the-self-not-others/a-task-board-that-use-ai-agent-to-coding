// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { buildDisplayComments, isCommentContentTaskIdEcho } from './buildDisplayComments.js'

describe('buildDisplayComments', () => {
  it('returns empty for null task', () => {
    expect(buildDisplayComments(null)).toEqual([])
  })

  it('preserves comment repo_identities for execution-details Git identity chips', () => {
    const rows = buildDisplayComments({
      comments: [
        {
          id: 'u1',
          content: '@img hello',
          created_at: '2026-08-21T10:00:00Z',
          repo_identities: [
            { repo_url: 'https://git.example/a.git', git_identity_id: 'gi_1' },
          ],
        },
      ],
      ai_comments: [],
    })
    expect(rows[0].repo_identities).toEqual([
      { repo_url: 'https://git.example/a.git', git_identity_id: 'gi_1' },
    ])
  })

  it('merges user, ai, and container_agent by created_at', () => {
    const rows = buildDisplayComments({
      id: 'task_1',
      comments: [
        { id: 'u1', content: 'human', created_at: '2026-07-15T10:00:00Z' },
      ],
      ai_comments: [
        { id: 'a1', content: 'to ai', created_at: '2026-07-15T10:02:00Z', assistant_response: 'ok' },
      ],
      container_agent_comments: [
        {
          id: 'c1',
          content: '@img',
          created_at: '2026-07-15T10:01:00Z',
          assistant_response: 'agent reply',
        },
      ],
    })
    expect(rows.map((r) => r.id)).toEqual(['u1', 'c1', 'a1'])
    expect(rows.map((r) => r.commentKind)).toEqual(['user', 'container_agent', 'ai'])
    expect(rows[1].assistant_response).toBe('agent reply')
  })

  it('tolerates missing container_agent_comments', () => {
    const rows = buildDisplayComments({
      comments: [{ id: 'u1', content: 'h', created_at: '2026-07-15T10:00:00Z' }],
      ai_comments: [],
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].commentKind).toBe('user')
  })

  it('filters comments whose content is only the task id', () => {
    const taskId = 'task_13646028863068037877'
    const rows = buildDisplayComments({
      id: taskId,
      comments: [
        { id: 'bad', content: taskId, created_at: '2026-07-15T10:00:00Z' },
        { id: 'ok', content: '真实评论', created_at: '2026-07-15T10:01:00Z' },
      ],
      ai_comments: [],
    })
    expect(rows.map((r) => r.id)).toEqual(['ok'])
    expect(isCommentContentTaskIdEcho(taskId, taskId)).toBe(true)
    expect(isCommentContentTaskIdEcho('真实评论', taskId)).toBe(false)
  })

  it('nests container_agent under parent_comment_id', () => {
    const rows = buildDisplayComments({
      id: 'task_1',
      comments: [
        { id: 'u1', content: '【自动运行】@img', created_at: '2026-07-15T10:00:00Z' },
        { id: 'u2', content: 'other', created_at: '2026-07-15T10:03:00Z' },
      ],
      ai_comments: [],
      container_agent_comments: [
        {
          id: 'c1',
          content: '@img',
          parent_comment_id: 'u1',
          created_at: '2026-07-15T10:01:00Z',
          assistant_response: 'agent reply',
        },
      ],
    })
    expect(rows.map((r) => r.id)).toEqual(['u1', 'u2'])
    expect(rows[0].children.map((r) => r.id)).toEqual(['c1'])
    expect(rows[0].children[0].commentKind).toBe('container_agent')
    expect(rows[1].children).toEqual([])
  })

  it('nests human git_pr replies under parent_comment_id', () => {
    const rows = buildDisplayComments({
      id: 'task_1',
      comments: [
        { id: 'u1', content: 'exec', created_at: '2026-08-21T10:00:00Z' },
        {
          id: 'pr1',
          content: 'https://github.com/acme/repo/pull/42',
          parent_comment_id: 'u1',
          git_pr: { html_url: 'https://github.com/acme/repo/pull/42', provider: 'github' },
          created_at: '2026-08-21T10:01:00Z',
        },
      ],
      ai_comments: [],
    })
    expect(rows.map((r) => r.id)).toEqual(['u1'])
    expect(rows[0].children.map((r) => r.id)).toEqual(['pr1'])
    expect(rows[0].children[0].git_pr.html_url).toBe('https://github.com/acme/repo/pull/42')
  })

  it('keeps orphan container_agent as top-level when parent missing', () => {
    const rows = buildDisplayComments({
      comments: [],
      ai_comments: [],
      container_agent_comments: [
        {
          id: 'c1',
          content: '@img',
          parent_comment_id: 'missing',
          created_at: '2026-07-15T10:01:00Z',
        },
      ],
    })
    expect(rows.map((r) => r.id)).toEqual(['c1'])
    expect(rows[0].children).toEqual([])
  })
})
