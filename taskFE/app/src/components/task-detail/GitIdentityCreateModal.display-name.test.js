// @vitest-environment node
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = path.dirname(fileURLToPath(import.meta.url))
const source = readFileSync(path.join(here, 'GitIdentityCreateModal.vue'), 'utf8')

describe('GitIdentityCreateModal 公司展示 (OPT-20260828-025)', () => {
  it('公司名缺失时展示「未设置」而不是 company_id', () => {
    expect(source).toContain("c.company_name || '未设置'")
    expect(source).not.toContain('c.company_name || c.company_id')
  })
})
