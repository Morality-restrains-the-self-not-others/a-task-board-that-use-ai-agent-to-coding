// @vitest-environment node
/**
 * 源码扫描：走网关的 container-layer / container-job / auto-run-steps 必须经 comment_id helper。
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const ACTION_RE = /container-(?:layer-|job-|auto-run-steps|bootstrap-clone-log|clone-log)|repo-reclone/
const HELPER_RE =
  /containerForwardCommentId|containerComputeRequest|appendCommentIdPath|appendCommentIdQuery|setCommentIdSearchParam|jsonPostWithCommentId|withCommentIdBody|getContainerCompute|postContainerCompute|containerComputeFuncFirstUrl|containerComputeKvLastUrl|containerComputeLegacyTaskUrl/
const SKIP_NAME_RE = /\.test\.|\.spec\.|scan\.test/

function walk(dir, acc = []) {
  for (const name of readdirSync(dir)) {
    if (name === 'node_modules') continue
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) walk(p, acc)
    else if (/\.(js|vue)$/.test(name) && !SKIP_NAME_RE.test(name)) acc.push(p)
  }
  return acc
}

const here = dirname(fileURLToPath(import.meta.url))
const roots = [
  here,
  join(here, '../../components/task-detail'),
  join(here, '../../utils'),
  join(here, '..'),
]

describe('container-layer 转发必须带 comment_id helper', () => {
  const files = [...new Set(roots.flatMap((r) => walk(r)))]
    .filter((p) => ACTION_RE.test(readFileSync(p, 'utf8')))
    .filter((p) => {
      const src = readFileSync(p, 'utf8')
      return src.includes('apiFetch(') || /\/api\/cloud\/(?:compute\/)?container-/.test(src)
    })

  it('扫描到至少一处转发源文件', () => {
    expect(files.length).toBeGreaterThan(5)
  })

  for (const p of files) {
    it(`${p.split('/app/src/')[1] || p} 使用 comment_id helper`, () => {
      const src = readFileSync(p, 'utf8')
      expect(src).toMatch(HELPER_RE)
    })
  }
})
