import { readFileSync } from 'node:fs'
import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { useTaskDetailFilesForLineLimit } from './useTaskDetailBundle.js'

function lineCount(abs) {
  return readFileSync(abs, 'utf8').split('\n').length - 1
}

describe('useTaskDetail 行数门禁', () => {
  it('门面与拆出的 composable 均 ≤500 行', () => {
    const files = useTaskDetailFilesForLineLimit()
    assert.ok(files.some((p) => p.endsWith('useTaskDetail.js')), 'missing facade')
    for (const abs of files) {
      const n = lineCount(abs)
      assert.ok(n <= 500, `${abs} 有 ${n} 行，超过 500`)
    }
  })
})
