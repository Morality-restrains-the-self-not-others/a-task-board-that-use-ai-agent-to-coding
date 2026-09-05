import { describe, expect, it } from 'vitest'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { shortContentHash, stripHashQuery } from '../../vite-plugin-asset-cache-bust.js'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

describe('vite-plugin-asset-cache-bust', () => {
  it('shortContentHash is stable hex slice', () => {
    const file = path.join(__dirname, 'assetUrl.js')
    const h = shortContentHash(file)
    expect(h).toMatch(/^[a-f0-9]{12}$/)
    expect(shortContentHash(file)).toBe(h)
  })

  it('stripHashQuery removes h param only', () => {
    expect(stripHashQuery('/a.js?h=abc&vue=1')).toBe('/a.js?vue=1')
    expect(stripHashQuery('/b.vue?vue&type=script&h=dead')).toBe('/b.vue?vue&type=script')
  })
})
