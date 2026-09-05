// @vitest-environment node
import { describe, expect, it, beforeEach } from 'vitest'
import {
  readCommentDependencyDraft,
  resetCommentDependencyDraft,
  setCommentDependencyDraft,
} from './commentDependencyDraft.js'

describe('commentDependencyDraft', () => {
  beforeEach(() => {
    resetCommentDependencyDraft()
  })

  it('defaults to wait_previous with empty depends', () => {
    expect(readCommentDependencyDraft()).toEqual({
      executionMode: 'wait_previous',
      dependsOnCommentIds: [],
      autoCommit: false,
    })
  })

  it('clears depends when independent', () => {
    setCommentDependencyDraft({
      executionMode: 'independent',
      dependsOnCommentIds: ['c1'],
    })
    expect(readCommentDependencyDraft()).toEqual({
      executionMode: 'independent',
      dependsOnCommentIds: [],
      autoCommit: false,
    })
  })
})
