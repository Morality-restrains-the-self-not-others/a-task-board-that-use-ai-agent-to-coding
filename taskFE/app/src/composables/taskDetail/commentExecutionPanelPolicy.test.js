import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  shouldEnableCommentRuntimeTab,
  shouldMountCommentConnectionPanel,
  shouldMountCommentRuntimeStatusSection,
} from './commentExecutionPanelPolicy.js'

const here = dirname(fileURLToPath(import.meta.url))

describe('commentExecutionPanelPolicy', () => {
  const panel = { fetchServerRuntimeStatus: () => {} }

  it('enables runtime tab for every comment when panel exists, including inactive', () => {
    expect(shouldEnableCommentRuntimeTab(panel)).toBe(true)
    expect(shouldEnableCommentRuntimeTab(null)).toBe(false)
    expect(shouldEnableCommentRuntimeTab(undefined)).toBe(false)
  })

  it('mounts runtime status section without isActive singleton gate', () => {
    expect(shouldMountCommentRuntimeStatusSection(panel)).toBe(true)
    expect(shouldMountCommentRuntimeStatusSection(null)).toBe(false)
  })

  it('mounts connection panel for every comment that owns a CSC', () => {
    expect(shouldMountCommentConnectionPanel(true)).toBe(true)
    expect(shouldMountCommentConnectionPanel(false)).toBe(false)
  })
})

describe('TaskDetailCommentsSection source: no isActive singleton gate', () => {
  const src = readFileSync(
    join(here, '../../components/task-detail/TaskDetailCommentsSection.vue'),
    'utf8',
  )

  it('does not gate runtime tab or runtime section on isActive', () => {
    expect(src).not.toMatch(/server-runtime-status-tab="isActive/)
    expect(src).not.toMatch(/v-if="isActive && serverRuntimeStatusPanel"/)
    expect(src).not.toMatch(/v-if="isActive && commentOwnsSharedContainer/)
  })

  it('binds runtime panel per comment id', () => {
    expect(src).toContain('bindCommentRuntimePanel(serverRuntimeStatusPanel, comment.id, bindingStatusFor(comment.id), comment.created_at)')
    expect(src).toContain(':server-runtime-status="runtimeStatusFromCommentPanel(serverRuntimeStatusPanel, comment.id)"')
    expect(src).toContain('shouldEnableCommentRuntimeTab')
    expect(src).toContain('shouldMountCommentRuntimeStatusSection')
    expect(src).toContain('shouldMountCommentConnectionPanel')
    expect(src).toContain('comment-execution-runtime-with-start')
    expect(src).not.toMatch(/comment-execution-runtime-with-start"[^]*:runtime-status="runtimeStatus"/)
  })

  it('binds server content panel per comment id (not task-level)', () => {
    expect(src).toContain('#server-content')
    expect(src).toContain('bindCommentServerContentPanel(serverContentPanel, comment.id)')
  })

  it('places per-comment layer body in #layer-ztree slot (no empty-comment fallback)', () => {
    expect((src.match(/#layer-ztree/g) || []).length).toBe(1)
    expect(src).toContain('layer-ztree-tab')
    expect((src.match(/<TaskDetailCommentLayerAssociationBody/g) || []).length).toBe(1)
    expect(src).not.toMatch(/execution-details-fallback/)
    expect(src).not.toMatch(/comment-id=""/)
  })
})

describe('ServerConfig.logic source: 服务器内容已迁出任务级', () => {
  const logicSrc = readFileSync(
    join(here, '../../components/ServerConfig.logic.vue'),
    'utf8',
  )

  it('task-level content tab shows moved hint instead of live fetch panel', () => {
    expect(logicSrc).toContain('server-content-moved-hint')
    expect(logicSrc).not.toMatch(/<ServerConfigServerContentSection/)
  })
})
