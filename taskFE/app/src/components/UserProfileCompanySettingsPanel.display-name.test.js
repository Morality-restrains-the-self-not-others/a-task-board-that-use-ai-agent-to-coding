// @vitest-environment node
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = path.dirname(fileURLToPath(import.meta.url))
const source = readFileSync(
  path.join(here, 'UserProfileCompanySettingsPanel.vue'),
  'utf8',
)

describe('UserProfileCompanySettingsPanel 公司展示 (OPT-20260828-025)', () => {
  it('公司名缺失时展示「未设置」而不是 company_id', () => {
    expect(source).toContain("item.company_name || '未设置'")
    expect(source).not.toContain('item.company_name || item.company_id')
  })

  it('头像缩略名不再回退到 company_id', () => {
    expect(source).not.toContain("item.company_name || item.company_id ||")
    expect(source).toMatch(/item\.member_name \|\| item\.company_name \|\| 'member'/)
  })
})
