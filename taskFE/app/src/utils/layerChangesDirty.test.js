/** 与 layerChangesDirty 共演进：文件变动是否意味着可提交（排除纯父层 diff）。 */
import { describe, expect, it } from 'vitest'
import {
  formatLayerSubmitFileChangesCaption,
  layerChangesPayloadImpliesWorktreeDirty,
  submitTitleForParentDiffOnly,
} from './layerChangesDirty.js'

describe('layerChangesPayloadImpliesWorktreeDirty', () => {
  it('仅 git_layer_diff_only 时不应解锁提交', () => {
    expect(
      layerChangesPayloadImpliesWorktreeDirty({
        same: false,
        change_count: 1,
        changes: [
          {
            path: 'somanyad/hello.lisp',
            kind: 'added',
            git_staged: false,
            git_unstaged: false,
            git_layer_diff_only: true,
          },
        ],
      }),
    ).toBe(false)
  })

  it('存在未暂存变更时应解锁提交', () => {
    expect(
      layerChangesPayloadImpliesWorktreeDirty({
        same: false,
        change_count: 1,
        changes: [
          {
            path: 'a.txt',
            kind: 'modified',
            git_staged: false,
            git_unstaged: true,
            git_layer_diff_only: false,
          },
        ],
      }),
    ).toBe(true)
  })

  it('存在暂存变更时应解锁提交', () => {
    expect(
      layerChangesPayloadImpliesWorktreeDirty({
        same: false,
        change_count: 2,
        changes: [
          {
            path: 'old.txt',
            git_layer_diff_only: true,
            git_staged: false,
            git_unstaged: false,
          },
          {
            path: 'new.txt',
            git_staged: true,
            git_unstaged: false,
            git_layer_diff_only: false,
          },
        ],
      }),
    ).toBe(true)
  })

  it('仅父层差异时文案应标明相对父层而非可提交变化', () => {
    const cap = formatLayerSubmitFileChangesCaption({
      same: false,
      change_count: 3,
      changes: [
        { path: 'a', git_layer_diff_only: true, git_staged: false, git_unstaged: false },
        { path: 'b', git_layer_diff_only: true, git_staged: false, git_unstaged: false },
        { path: 'c', git_layer_diff_only: true, git_staged: false, git_unstaged: false },
      ],
    })
    expect(cap.parentDiffOnly).toBe(true)
    expect(cap.label).toBe('3 个相对父层差异')
    expect(cap.title).toMatch(/相对父层/)
    expect(submitTitleForParentDiffOnly(3)).toMatch(/3 个文件仅为相对父层差异/)
  })

  it('含可提交变更时文案仍为文件变化', () => {
    const cap = formatLayerSubmitFileChangesCaption({
      same: false,
      change_count: 1,
      changes: [{ path: 'a.txt', git_unstaged: true, git_staged: false }],
    })
    expect(cap.parentDiffOnly).toBe(false)
    expect(cap.label).toBe('1 个文件变化')
  })
})
