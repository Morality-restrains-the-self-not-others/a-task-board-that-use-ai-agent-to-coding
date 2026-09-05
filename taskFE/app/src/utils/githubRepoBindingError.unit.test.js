// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  assignGithubRepoBindingCatchError,
  assignGithubRepoBindingError,
  makeGithubRepoBindingHttpError,
} from './githubRepoBindingError.js'

describe('githubRepoBindingError', () => {
  it('assigns message and traceId from body', () => {
    const errorRef = { value: '' }
    const traceRef = { value: '' }
    assignGithubRepoBindingError(errorRef, traceRef, { message: 'ported?', trace_id: 't-1' }, {
      fallback: 'fallback',
    })
    expect(errorRef.value).toBe('ported?')
    expect(traceRef.value).toBe('t-1')
  })

  it('makeGithubRepoBindingHttpError carries traceId', () => {
    const err = makeGithubRepoBindingHttpError({ detail: 'x', traceId: 'abc' }, null, 'fb')
    expect(err.message).toBe('x')
    expect(err.traceId).toBe('abc')
  })

  it('assignGithubRepoBindingCatchError reads err.traceId via extractTraceId', () => {
    const errorRef = { value: '' }
    const traceRef = { value: '' }
    const e = new Error('boom')
    e.traceId = 'tid'
    assignGithubRepoBindingCatchError(errorRef, traceRef, e, 'fb')
    expect(errorRef.value).toBe('boom')
    expect(traceRef.value).toBe('tid')
  })
})
