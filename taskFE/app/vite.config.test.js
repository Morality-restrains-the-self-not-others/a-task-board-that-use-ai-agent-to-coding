import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const configPath = resolve(here, './vite.config.js')
const source = fs.readFileSync(configPath, 'utf8')

// 回归：生产构建必须用 esbuild minify（terser 在本机内存紧张时于
// rendering chunks 阶段 OOM Kill，阻塞 SPA 发布）。OPT-20260812-034。
describe('vite.config.js 生产构建 minify', () => {
  it('生产 minify 为 esbuild（低峰值内存）而非 terser', () => {
    expect(source).toMatch(/minify:\s*isDev\s*\?\s*false\s*:\s*'esbuild'/)
    expect(source).not.toMatch(/'terser'/)
  })

  it('不再保留仅 terser 需要的 terserOptions', () => {
    expect(source).not.toMatch(/terserOptions/)
  })

  it('从 vue.contactEmail 注入 VITE_CONTACT_EMAIL', () => {
    expect(source).toMatch(/VITE_CONTACT_EMAIL/)
    expect(source).toMatch(/contactEmail/)
  })

  it('从 vue.icpBeian 注入 VITE_ICP_BEIAN', () => {
    expect(source).toMatch(/VITE_ICP_BEIAN/)
    expect(source).toMatch(/icpBeian/)
  })
})
