/**
 * 产物/源码新鲜度校验（OPT-20260824-056）。
 *
 * 用途：E2E/验收前对比 :4000 产物 index.html 内嵌的 build-time 与源码最新 mtime，
 * 检测「产物与源码漂移」——旧 public/html 承载旧构建而源码已推进时，UI 断言打旧产物，
 * 失败诊断浪费多轮。漂移时给出明确提示，避免把「产物过期」误当功能回归。
 *
 * 纯函数（parse / buildTimeToMs / compareFreshness）与 IO（latestSourceMtime /
 * checkProdBuildFreshness）分离，便于 node:test 与 Playwright 共用。
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export const BUILD_TIME_META_NAME = 'build-time'

/** 从 index.html 文本解析 build-time meta（ISO-8601 字符串或 null）。 */
export function parseBuildTimeFromHtml(html) {
  const m = String(html || '').match(
    new RegExp(`<meta\\s+name="${BUILD_TIME_META_NAME}"\\s+content="([^"]+)"`, 'i')
  )
  return m ? m[1] : null
}

/** ISO-8601 → epoch ms；解析失败返回 null。 */
export function buildTimeToMs(iso) {
  if (!iso) return null
  const ms = Date.parse(iso)
  return Number.isNaN(ms) ? null : ms
}

/**
 * 对比产物与源码新鲜度（纯函数）。
 * @param {{ buildTimeMs: number|null, sourceMtimeMs: number, graceMs?: number }}
 * @returns {{ fresh: boolean|null, reason?: string, driftMs: number }}
 *   fresh=null 表示产物无 build-time 标记，无法校验。
 */
export function compareFreshness({ buildTimeMs, sourceMtimeMs, graceMs = 0 }) {
  if (buildTimeMs === null) {
    return { fresh: null, reason: '产物无 build-time 标记（旧产物未含 OPT-056 插件），无法校验新鲜度', driftMs: 0 }
  }
  const driftMs = sourceMtimeMs - buildTimeMs
  const fresh = driftMs <= graceMs
  if (fresh) return { fresh, driftMs: Math.max(0, driftMs) }
  return {
    fresh: false,
    reason: `产物构建于 ${new Date(buildTimeMs).toISOString()}，源码最新修改 ${new Date(sourceMtimeMs).toISOString()}（漂移 ${Math.round(driftMs / 1000)}s）`,
    driftMs,
  }
}

/** 递归扫描目录，返回最新文件 mtimeMs（无文件返回 0）。 */
export function latestSourceMtime(sourceDir) {
  let max = 0
  const walk = (dir) => {
    let entries
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true })
    } catch {
      return
    }
    for (const ent of entries) {
      if (ent.name === 'node_modules' || ent.name.startsWith('.') || ent.name === 'dist') continue
      const p = path.join(dir, ent.name)
      if (ent.isDirectory()) walk(p)
      else {
        try {
          const st = fs.statSync(p)
          if (st.mtimeMs > max) max = st.mtimeMs
        } catch {
          /* 忽略不可读文件 */
        }
      }
    }
  }
  walk(sourceDir)
  return max
}

/** 默认源码目录：taskFE/app/src（Vite root）。 */
export function defaultSourceDir() {
  return path.resolve(__dirname, '..', '..', 'app', 'src')
}

/**
 * 综合校验：抓取产物 index.html → 解析 build-time → 与源码最新 mtime 比较。
 * 产物不可达时返回 { reachable:false } 并告警，不抛错（便于测试环境无服务时 skip）。
 *
 * @param {{ baseUrl: string, sourceDir?: string, fetchFn?: Function, logger?: object, graceMs?: number }}
 *   fetchFn(url) → { ok:boolean, text():Promise<string> }（默认 globalThis.fetch，Playwright 可传 request）。
 */
export async function checkProdBuildFreshness({
  baseUrl,
  sourceDir = defaultSourceDir(),
  fetchFn = globalThis.fetch,
  logger = console,
  graceMs = 60_000,
}) {
  let html
  try {
    const res = await fetchFn(`${baseUrl}/`)
    if (!res.ok) throw new Error(`fetch ${baseUrl}/ status=${res.status}`)
    html = await res.text()
  } catch (err) {
    logger.warn(`[prod-build-freshness] 无法访问产物 ${baseUrl}/：${err.message}，跳过新鲜度校验`)
    return { reachable: false, fresh: null }
  }

  const buildTime = parseBuildTimeFromHtml(html)
  const buildTimeMs = buildTimeToMs(buildTime)
  const sourceMtimeMs = latestSourceMtime(sourceDir)
  const verdict = compareFreshness({ buildTimeMs, sourceMtimeMs, graceMs })

  if (verdict.fresh === false) {
    logger.warn(`[prod-build-freshness] ⚠ 产物过期：${verdict.reason}。E2E 可能打到旧产物，请先 atomic-vite-build.sh 重新构建并切 public/html。`)
  } else if (verdict.fresh === null) {
    logger.warn(`[prod-build-freshness] ⚠ ${verdict.reason}。`)
  } else {
    logger.log(`[prod-build-freshness] 产物新鲜（build-time=${buildTime}，源码最新 ${new Date(sourceMtimeMs).toISOString()}）。`)
  }

  return { reachable: true, buildTime, buildTimeMs, sourceMtimeMs, ...verdict }
}
