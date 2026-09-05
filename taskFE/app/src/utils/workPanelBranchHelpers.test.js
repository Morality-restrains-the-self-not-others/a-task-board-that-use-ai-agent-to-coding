// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  buildDefaultWorkBranchName,
  resolveBranchNamePlaceholders,
  sanitizeBranchSegment,
} from './workPanelBranchHelpers.js'

describe('buildDefaultWorkBranchName', () => {
  it('无标题时保留 ${taskTitle} 占位符（不得 sanitize 成 __taskTitle_）', () => {
    const name = buildDefaultWorkBranchName('feature', '', 'user@example.com', '2026-07-12T00:00:00Z')
    expect(name).toContain('${taskTitle}')
    expect(name).not.toContain('__taskTitle_')
    expect(name).toMatch(/^feature\/2026-07-12_ljy_qq\.com_aidev\$\{taskId\}_\$\{taskTitle\}$/)
  })

  it('有标题时写入 sanitize 后的标题段', () => {
    const name = buildDefaultWorkBranchName('feature', 'Hello World', 'user', '2026-07-12T00:00:00Z')
    expect(name).toContain('_Hello_World')
    expect(name).not.toContain('${taskTitle}')
  })
})

describe('resolveBranchNamePlaceholders', () => {
  it('同时替换 ${taskId} 与 ${taskTitle}', () => {
    const resolved = resolveBranchNamePlaceholders(
      'feature/2026-07-12_user_daydaymoney${taskId}_${taskTitle}',
      { taskId: 'task_1', taskTitleSegment: 'My_Feature' },
    )
    expect(resolved).toBe('feature/2026-07-12_user_daydaymoneytask_1_My_Feature')
  })

  it('修复历史误 sanitize 的 __taskTitle_', () => {
    const resolved = resolveBranchNamePlaceholders(
      'feature/2026-07-12_ljy_daydaymoneytask_129___taskTitle_',
      { taskId: 'task_129', taskTitleSegment: 'fix_gitlab_push' },
    )
    expect(resolved).toBe('feature/2026-07-12_ljy_daydaymoneytask_129_fix_gitlab_push')
    expect(resolved).not.toContain('taskTitle')
  })

  it('无 taskId 时保留 ${taskId}，仍替换标题占位符', () => {
    const resolved = resolveBranchNamePlaceholders(
      'feature/d_u_daydaymoney${taskId}_${taskTitle}',
      { taskTitleSegment: 'seg' },
    )
    expect(resolved).toBe('feature/d_u_daydaymoney${taskId}_seg')
  })
})

describe('sanitizeBranchSegment', () => {
  it('将空白与非法字符替换为下划线', () => {
    expect(sanitizeBranchSegment('a b/c', 'task')).toBe('a_b_c')
  })
})
