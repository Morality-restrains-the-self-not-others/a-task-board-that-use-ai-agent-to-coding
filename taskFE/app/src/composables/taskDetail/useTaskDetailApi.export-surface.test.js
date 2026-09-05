// @vitest-environment node
/**
 * TaskDetail.vue destructures useTaskDetail's return. A missing key becomes
 * `undefined`, then runtimeSectionProps does `x.value` and Vue throws
 * "Cannot read properties of undefined (reading 'value')" — which aborts
 * the computed and leaves 关联项目 on empty defaults (production
 * task_881388002226499584).
 */
if (!process.env.VITEST) {
  console.log('[skip] useTaskDetailApi.export-surface.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  function taskDetailUseTaskDetailKeys(src) {
    const m = src.match(/const \$ = useTaskDetail[\s\S]*?const \{([\s\S]*?)\} = \$/)
    if (!m) return []
    return m[1]
      .split(',')
      .map((k) => k.replace(/\/\/.*$/, '').trim())
      .filter(Boolean)
  }

  function apiReturnKeys(src) {
    const m = src.match(/export function buildUseTaskDetailApi[\s\S]*?\n  return \{([\s\S]*)\n  \}\n\}/)
    if (!m) return new Set()
    const body = m[1].replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '')
    return new Set(
      [...body.matchAll(/(?:^|[,{])\s*(?:get |set )?([A-Za-z_][A-Za-z0-9_]*)/gm)].map((x) => x[1]),
    )
  }

  describe('useTaskDetail API surface vs TaskDetail.vue', () => {
    it('exports every key TaskDetail destructures from useTaskDetail()', () => {
      const detail = readFileSync(join(here, '../../views/TaskDetail.vue'), 'utf8')
      const api = readFileSync(join(here, 'useTaskDetailApi.js'), 'utf8')
      const keys = taskDetailUseTaskDetailKeys(detail)
      const returned = apiReturnKeys(api)
      expect(keys.length).toBeGreaterThan(50)
      const missing = keys.filter((k) => !returned.has(k))
      expect(missing).toEqual([])
    })

    it('pulls staleRepoSyncNeedsReclone off d and returns it next to sibling repo-sync keys', () => {
      const api = readFileSync(join(here, 'useTaskDetailApi.js'), 'utf8')
      expect(api).toMatch(
        /staleRepoSyncLoading, staleRepoSyncError, staleRepoSyncNeedsReclone, repoBranchesCache/,
      )
      expect(api).toMatch(
        /staleRepoSyncLoading, staleRepoSyncError, staleRepoSyncNeedsReclone, syncStaleTaskRepoAddresses/,
      )
    })
  })
}
