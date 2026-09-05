import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const src = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'TaskDetailCommentsSection.vue'),
  'utf8',
)

describe('TaskDetailCommentsSection runtime panel bind', () => {
  it('forwards comment.created_at so CreationTime lag can render', () => {
    expect(src).toContain(
      'bindCommentRuntimePanel(serverRuntimeStatusPanel, comment.id, bindingStatusFor(comment.id), comment.created_at)',
    )
  })
})
