import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = path.dirname(fileURLToPath(import.meta.url))
const vue = readFileSync(path.join(here, 'ImageMarketVendorApply.vue'), 'utf8')

describe('ImageMarketVendorApply.vue', () => {
  it('证照直传走 uploadIssuedFile，提交带 Idempotency-Key', () => {
    expect(vue).toMatch(/uploadIssuedFile/)
    expect(vue).toMatch(/createClickGuard/)
    expect(vue).toMatch(/Idempotency-Key|mergeIdempotencyHeaders/)
    expect(vue).toMatch(/data-testid="vendor-app-form"/)
    expect(vue).toMatch(/data-testid="vendor-apply-cancel"/)
    expect(vue).toMatch(/emailBindingHref/)
    expect(vue).not.toMatch(/setInterval/)
  })
})
