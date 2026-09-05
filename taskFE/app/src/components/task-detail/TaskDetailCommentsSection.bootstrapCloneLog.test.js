import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))

describe('TaskDetailCommentsSection bootstrap clone-log prop', () => {
  it('declares bootstrapCloneLogText for writable-layer failure hint', () => {
    const src = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
    assert.match(src, /bootstrapCloneLogText:\s*\{\s*type:\s*String/)
  })

  it('T35: comment clone rows prefer full bootstrap log over ztree layer log', () => {
    const src = readFileSync(join(here, 'useCommentSectionCloneProgressRows.js'), 'utf8')
    assert.match(src, /bootstrapLog:\s*props\.bootstrapCloneLogText\s*\|\|\s*props\.layerCloneLogText/)
  })
})
