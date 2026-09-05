// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))

describe('comment ztree serving source', () => {
  it('does not compute comment ztree loading from useTaskDetail isServerRunning', () => {
    const lg = readFileSync(join(here, 'taskDetailLayerGraphState.js'), 'utf8')
    expect(lg).not.toMatch(/\bisServerRunning\b/)
    expect(lg).not.toMatch(/\bisServerStarting\b/)
    expect(lg).not.toMatch('shouldShowCommentLayerZtreeLoading')
    expect(lg).not.toMatch('shouldShowCommentLayerZtreeReleased')
  })

  it('comments bindings do not pass isServerRunning or page-level ztree loading flags', () => {
    const bindings = readFileSync(join(here, '../../views/taskDetailSectionBindings.js'), 'utf8')
    const commentsFn = bindings.split('export function useTaskDetailCommentsSectionBindings')[1] || ''
    expect(commentsFn.length).toBeGreaterThan(100)
    expect(commentsFn).not.toMatch(/\bisServerRunning\b/)
    expect(commentsFn).not.toMatch('showCommentLayerZtreeLoading')
    expect(commentsFn).not.toMatch('showCommentLayerZtreeReleased')
    expect(commentsFn).not.toMatch('commentLayerZtreeLoadingHint')
    expect(commentsFn).not.toMatch(/\bcontainerHeartbeatStatus\b/)
  })

  it('CommentsSection does not declare isServerRunning or page-level ztree loading props', () => {
    const section = readFileSync(
      join(here, '../../components/task-detail/TaskDetailCommentsSection.vue'),
      'utf8',
    )
    expect(section).not.toMatch(/\bisServerRunning\b/)
    expect(section).not.toMatch('showCommentLayerZtreeLoading:')
    expect(section).not.toMatch('showCommentLayerZtreeReleased:')
  })
})
