// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

describe('WorkspaceMachinePolicyModal idle reuse copy', () => {
  const dir = dirname(fileURLToPath(import.meta.url))
  const src = readFileSync(join(dir, 'WorkspaceMachinePolicyModal.vue'), 'utf8')

  it('does not offer cross-task idle reuse (ADR-0013)', () => {
    expect(src).not.toContain('优先在闲置机器节点上启动新容器')
    expect(src).not.toContain('preferIdleReuse')
    expect(src).toContain('闲置自动回收')
  })

  it('defaults idle recycle minutes to 5 when policy omits the field', () => {
    expect(src).toMatch(/idleRecycleMinutes = ref\(5\)/)
    expect(src).toMatch(/idle_recycle_minutes \?\? 5/)
    expect(src).not.toMatch(/idleRecycleMinutes = ref\(30\)/)
    expect(src).not.toMatch(/idle_recycle_minutes \?\? 30/)
  })
})
