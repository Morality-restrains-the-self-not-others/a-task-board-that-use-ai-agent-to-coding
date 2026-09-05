/**
 * node:test — prodBuildFreshness 纯函数（OPT-20260824-056）。
 * 运行：node --test tests/helpers/prodBuildFreshness.test.js（taskFE 根）
 */
import { test } from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import {
  parseBuildTimeFromHtml,
  buildTimeToMs,
  compareFreshness,
  latestSourceMtime,
  checkProdBuildFreshness,
} from './prodBuildFreshness.js'

test('parseBuildTimeFromHtml 命中 meta 并返回 ISO', () => {
  const html = '<html><head><meta name="build-time" content="2026-08-24T04:00:00.000Z"></head></html>'
  assert.equal(parseBuildTimeFromHtml(html), '2026-08-24T04:00:00.000Z')
})

test('parseBuildTimeFromHtml 大小写不敏感与缺失返回 null', () => {
  const html = '<meta NAME="Build-Time" content="2026-08-24T04:00:00.000Z">'
  assert.equal(parseBuildTimeFromHtml(html), '2026-08-24T04:00:00.000Z')
  assert.equal(parseBuildTimeFromHtml('<html></html>'), null)
  assert.equal(parseBuildTimeFromHtml(''), null)
  assert.equal(parseBuildTimeFromHtml(null), null)
})

test('buildTimeToMs 解析 ISO 与非法输入', () => {
  assert.equal(buildTimeToMs('2026-08-24T04:00:00.000Z'), Date.parse('2026-08-24T04:00:00.000Z'))
  assert.equal(buildTimeToMs('not-a-date'), null)
  assert.equal(buildTimeToMs(null), null)
})

test('compareFreshness 无标记 / 新鲜 / 过期三态', () => {
  const build = Date.parse('2026-08-24T04:00:00.000Z')
  assert.deepEqual(compareFreshness({ buildTimeMs: null, sourceMtimeMs: build + 1000 }), {
    fresh: null,
    reason: '产物无 build-time 标记（旧产物未含 OPT-056 插件），无法校验新鲜度',
    driftMs: 0,
  })

  // 新鲜：源码最新修改 <= 构建时间 + grace
  assert.equal(compareFreshness({ buildTimeMs: build, sourceMtimeMs: build - 1000 }).fresh, true)
  assert.equal(compareFreshness({ buildTimeMs: build, sourceMtimeMs: build + 30_000, graceMs: 60_000 }).fresh, true)

  // 过期：源码在构建后修改超过 grace
  const stale = compareFreshness({ buildTimeMs: build, sourceMtimeMs: build + 3_600_000 })
  assert.equal(stale.fresh, false)
  assert.match(stale.reason, /漂移 3600s/)
})

test('latestSourceMtime 扫描目录取最新 mtime', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'freshness-'))
  try {
    const a = path.join(dir, 'a.js')
    fs.writeFileSync(a, 'x')
    const oldMtime = new Date(Date.now() - 3_600_000)
    fs.utimesSync(a, oldMtime, oldMtime)
    fs.mkdirSync(path.join(dir, 'nested'))
    fs.writeFileSync(path.join(dir, 'nested', 'b.vue'), 'y')
    const mtime = latestSourceMtime(dir)
    // b.vue 为最新
    assert.ok(mtime > oldMtime.getTime())
    // mtimeMs 带子毫秒精度，紧贴写入后读取时往往略大于整数截断的 Date.now()，
    // 允许 1s 偏差以容忍时间戳舍入/时钟微偏移（同时仍能拦截真实「未来时间戳」）
    assert.ok(mtime <= Date.now() + 1_000)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('latestSourceMtime 跳过 node_modules 与不存在目录', () => {
  assert.equal(latestSourceMtime('/nonexistent-dir-xyz'), 0)
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'freshness2-'))
  try {
    fs.mkdirSync(path.join(dir, 'node_modules'))
    fs.writeFileSync(path.join(dir, 'node_modules', 'junk.js'), 'junk')
    assert.equal(latestSourceMtime(dir), 0)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('checkProdBuildFreshness 用 mock fetch 返回过期报告', async () => {
  // 产物构建于 1 小时前；源码最新修改为现在 → 过期
  const buildTime = new Date(Date.now() - 3_600_000).toISOString()
  const fetchFn = async () => ({
    ok: true,
    text: async () => `<html><head><meta name="build-time" content="${buildTime}"></head></html>`,
  })
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'freshness3-'))
  try {
    fs.writeFileSync(path.join(dir, 's.js'), 'x')
    const sourceMtime = Date.now()
    const report = await checkProdBuildFreshness({
      baseUrl: 'http://127.0.0.1:4000',
      sourceDir: dir,
      fetchFn,
      logger: { warn: () => {}, log: () => {} },
    })
    assert.equal(report.reachable, true)
    assert.equal(report.buildTime, buildTime)
    // 源码 mtime 晚于 build-time（超过 grace 60s）→ 过期
    assert.equal(report.fresh, false)
    assert.ok(report.sourceMtimeMs >= sourceMtime - 10_000)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('checkProdBuildFreshness 产物不可达时返回 reachable=false 不抛错', async () => {
  const report = await checkProdBuildFreshness({
    baseUrl: 'http://127.0.0.1:1',
    sourceDir: path.join(os.tmpdir(), 'nonexistent'),
    fetchFn: async () => {
      throw new Error('ECONNREFUSED')
    },
    logger: { warn: () => {}, log: () => {} },
  })
  assert.equal(report.reachable, false)
  assert.equal(report.fresh, null)
})
