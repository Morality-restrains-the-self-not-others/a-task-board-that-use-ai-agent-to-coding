/**
 * 单测：vite-plugin-build-time-meta（OPT-20260824-056）。
 * 覆盖 meta 标签格式、注入位置、插件形态与固定时间可测性。
 */
import { describe, it, expect } from 'vitest'
import {
  buildTimeMetaTag,
  injectBuildTimeMeta,
  buildTimeMetaPlugin,
} from './vite-plugin-build-time-meta.js'

describe('vite-plugin-build-time-meta', () => {
  const FIXED = '2026-08-24T04:00:00.000Z'

  it('buildTimeMetaTag 输出合法 meta 标签', () => {
    expect(buildTimeMetaTag(FIXED)).toBe(
      `<meta name="build-time" content="${FIXED}">`
    )
  })

  it('injectBuildTimeMeta 注入到 </head> 前', () => {
    const html = '<!DOCTYPE html>\n<html>\n<head><title>t</title></head>\n<body></body>\n</html>'
    const out = injectBuildTimeMeta(html, FIXED)
    expect(out).toContain(`<meta name="build-time" content="${FIXED}">`)
    // meta 必须在 </head> 之前
    expect(out.indexOf('build-time')).toBeLessThan(out.indexOf('</head>'))
    // 原内容保留
    expect(out).toContain('<title>t</title>')
    expect(out).toContain('<body></body>')
  })

  it('injectBuildTimeMeta 无 </head> 时退化为前置', () => {
    const out = injectBuildTimeMeta('<html><body>x</body></html>', FIXED)
    expect(out.startsWith(`<meta name="build-time" content="${FIXED}">`)).toBe(true)
  })

  it('plugin 仅作用于 build 且注入固定时间', () => {
    const plugin = buildTimeMetaPlugin({ now: () => new Date(FIXED) })
    expect(plugin.name).toBe('task2app-build-time-meta')
    expect(plugin.apply).toBe('build')
    const html = '<html><head></head><body></body></html>'
    const out = plugin.transformIndexHtml(html)
    expect(out).toContain(`<meta name="build-time" content="${FIXED}">`)
  })

  it('plugin 默认注入当前时间（非固定）', () => {
    const plugin = buildTimeMetaPlugin()
    const before = Date.now()
    const out = plugin.transformIndexHtml('<html><head></head></html>')
    const m = out.match(/<meta name="build-time" content="([^"]+)">/)
    expect(m).toBeTruthy()
    const ts = Date.parse(m[1])
    expect(Number.isNaN(ts)).toBe(false)
    // 允许毫秒级抖动（两次 Date 调用间）
    expect(Math.abs(ts - before)).toBeLessThan(5000)
  })
})
