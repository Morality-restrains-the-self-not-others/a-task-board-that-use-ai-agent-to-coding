// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  BRANCH_PREVIEW_REQUEST_TIMEOUT_MS,
  BRANCH_PREVIEW_TIMEOUT_ERROR,
  resolveBranchPreviewFetchError,
} from './branchPreviewUtils.js'

describe('branchPreviewUtils', () => {
  it('超时预算覆盖后端 gitOauth 5s + GitLab 10s', () => {
    expect(BRANCH_PREVIEW_REQUEST_TIMEOUT_MS).toBeGreaterThanOrEqual(15000)
  })

  it('AbortError 返回分支预览专用超时文案', () => {
    expect(resolveBranchPreviewFetchError({ name: 'AbortError' })).toBe(BRANCH_PREVIEW_TIMEOUT_ERROR)
    expect(resolveBranchPreviewFetchError(new Error('network'))).toBe('network')
  })
})
