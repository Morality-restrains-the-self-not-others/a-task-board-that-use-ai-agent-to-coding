import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

describe('ProjectEdit clone-alias markup', () => {
  it('renders a single clone-alias input per row template', () => {
    const dir = dirname(fileURLToPath(import.meta.url))
    const src = readFileSync(join(dir, 'ProjectEdit.vue'), 'utf8')
    const matches = src.match(/data-testid="git-repo-clone-alias-input"/g) || []
    expect(matches.length).toBe(1)
  })
})
