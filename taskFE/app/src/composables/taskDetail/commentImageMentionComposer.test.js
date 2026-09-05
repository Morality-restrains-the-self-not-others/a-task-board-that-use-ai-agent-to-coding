/** @vitest-environment jsdom */
if (!process.env.VITEST) {
  console.log('[skip] commentImageMentionComposer.test.js requires vitest runtime')
  process.exit(0)
}
import { describe, expect, it } from 'vitest'
import {
  extractMentionFromPlainText,
  filterInstalledImages,
  findActiveAtQuery,
  matchImageByExactName,
  replaceAtQueryWithMention,
  serializeComposerDom,
  extractMentionFromDom,
} from './commentImageMentionComposer.js'

const images = [
  { id: '1', name: 'trae0630' },
  { id: '2', name: 'coder' },
]

describe('findActiveAtQuery', () => {
  it('detects $ at start', () => {
    expect(findActiveAtQuery('$tra', 4)).toEqual({ start: 0, query: 'tra' })
  })

  it('detects $ after whitespace', () => {
    expect(findActiveAtQuery('hi $cod', 7)).toEqual({ start: 3, query: 'cod' })
  })

  it('ignores email-like $', () => {
    expect(findActiveAtQuery('a$b', 3)).toBeNull()
  })

  it('closes query when space appears after $', () => {
    expect(findActiveAtQuery('$foo bar', 5)).toBeNull()
  })
})

describe('filterInstalledImages', () => {
  it('filters by name substring', () => {
    expect(filterInstalledImages(images, 'trae').map((x) => x.id)).toEqual(['1'])
  })

  it('returns all when query empty', () => {
    expect(filterInstalledImages(images, '').length).toBe(2)
  })
})

describe('matchImageByExactName / extractMentionFromPlainText', () => {
  it('matches exact name case-insensitively', () => {
    expect(matchImageByExactName(images, 'TRAE0630')?.id).toBe('1')
  })

  it('extracts first valid $name token', () => {
    expect(extractMentionFromPlainText('请用 $trae0630 跑一下', images)).toEqual({
      id: '1',
      name: 'trae0630',
    })
  })

  it('returns null when $token is not an installed image', () => {
    expect(extractMentionFromPlainText('$unknown hi', images)).toBeNull()
  })
})

describe('replaceAtQueryWithMention', () => {
  it('replaces active query with $name', () => {
    expect(replaceAtQueryWithMention('hi $tr', 3, 6, 'trae0630')).toEqual({
      text: 'hi $trae0630',
      caret: 12,
    })
  })
})

describe('serializeComposerDom / extractMentionFromDom', () => {
  it('serializes mention chip and plain text', () => {
    const root = document.createElement('div')
    root.appendChild(document.createTextNode('请用 '))
    const chip = document.createElement('span')
    chip.dataset.mentionId = '1'
    chip.dataset.mentionName = 'trae0630'
    chip.textContent = '$trae0630'
    root.appendChild(chip)
    root.appendChild(document.createTextNode(' 跑一下'))
    expect(serializeComposerDom(root)).toBe('请用 $trae0630 跑一下')
    expect(extractMentionFromDom(root)).toEqual({ id: '1', name: 'trae0630' })
  })
})
