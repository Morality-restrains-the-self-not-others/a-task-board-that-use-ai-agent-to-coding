import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))

describe('taskDetailSectionBindings bootstrap failure trace', () => {
  it('passes containerBootstrapFailureTraceId into comments section props', () => {
    const src = readFileSync(join(here, 'taskDetailSectionBindings.js'), 'utf8')
    const commentsFn = src.split('export function useTaskDetailCommentsSectionBindings')[1] || ''
    assert.match(commentsFn, /containerBootstrapFailureTraceId/)
    assert.match(
      commentsFn,
      /containerBootstrapFailureTraceId:\s*containerBootstrapFailureTraceId\?\.value \?\? ''/,
    )
    assert.match(commentsFn, /bootstrapCloneLogText:\s*containerBootstrapCloneLogFull\?\.value \?\? ''/)
  })
})
