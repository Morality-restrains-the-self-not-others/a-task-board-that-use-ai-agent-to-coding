import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, it } from 'node:test'

const src = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'SecureInteractiveRichTextFrame.vue'),
  'utf8',
)

describe('SecureInteractiveRichTextFrame postMessage origin', () => {
  it('message 监听回调内比较 origin', () => {
    assert.match(src, /addEventListener\(\s*'message'/)
    assert.match(src, /ev\.origin\s*!==\s*window\.location\.origin/)
  })
})
