// @vitest-environment node
/**
 * 回归：模态遮罩样式已收窄为显式 .app-modal-overlay；下拉点击外部关闭使用独立
 * .dropdown-click-outside-overlay（自带 fixed+inset-0 铺满视口 + 透明背景，无灰罩）。
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const cssPath = join(dirname(fileURLToPath(import.meta.url)), 'styles.css')
const css = readFileSync(cssPath, 'utf8')

describe('styles.css — dropdown-click-outside-overlay', () => {
  it('存在独立透明遮罩规则，且声明 background-color: transparent 与 z-index: 40', () => {
    expect(css).toMatch(/\.dropdown-click-outside-overlay\s*\{/)
    const block = css.match(/\.dropdown-click-outside-overlay\s*\{([^}]+)\}/)
    expect(block).toBeTruthy()
    const body = block[1]
    expect(body).toMatch(/position:\s*fixed/)
    expect(body).toMatch(/background-color:\s*transparent/)
    expect(body).toMatch(/z-index:\s*40/)
    expect(body).toMatch(/display:\s*block/)
  })

  it('裸 .fixed.inset-0 已收窄为纯定位，不再注入灰罩 / z-9999 / 居中', () => {
    const block = css.match(/\.fixed\.inset-0\s*\{([^}]+)\}/)
    expect(block).toBeTruthy()
    const body = block[1]
    expect(body).toMatch(/position:\s*fixed/)
    expect(body).not.toMatch(/background-color/)
    expect(body).not.toMatch(/z-index:\s*9999/)
    expect(body).not.toMatch(/justify-content/)
  })

  it('模态遮罩样式已迁到显式 .app-modal-overlay 类', () => {
    const block = css.match(/\.app-modal-overlay\s*\{([^}]+)\}/)
    expect(block).toBeTruthy()
    const body = block[1]
    expect(body).toMatch(/background-color:\s*rgba\(0,\s*0,\s*0,\s*0\.5\)/)
    expect(body).toMatch(/z-index:\s*9999/)
    expect(body).toMatch(/justify-content:\s*center/)
  })
})
