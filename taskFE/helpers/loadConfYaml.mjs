/**
 * loadConfYaml — 读取 monorepo conf/ 端口/域名配置（taskFE 内联版）。
 *
 * OPT-20260806-040: 原 helper 位于 ../task2app/playwright/helpers/loadConfYaml.mjs，
 * 该目录已被清空且无 git 追踪，导致所有 playwright.config.* 加载即
 * ERR_MODULE_NOT_FOUND。此文件内联到 taskFE 侧，通过 runAll/scripts/conf-read.py
 * snapshot-json（SSoT: conf/<app>/config.yaml + base.yaml 模板展开）取数，
 * 保持与既有消费端（portConfig.vue/django/domainEvents/relayToTrae…）同构。
 */

import { execFileSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
// taskFE/helpers → monorepo root
const MONOREPO_ROOT = path.resolve(__dirname, '..', '..')
const CONF_READ = path.join(MONOREPO_ROOT, 'runAll', 'scripts', 'conf-read.py')

let _snapshot = null
let _snapshotError = null

/** 运行 conf-read.py snapshot-json 并缓存结果（失败时抛错，调用方可 catch）。 */
export function loadPortConfig() {
  if (_snapshot !== null) return _snapshot
  try {
    const out = execFileSync('python3', [CONF_READ, 'snapshot-json'], {
      encoding: 'utf-8',
      timeout: 15000,
    })
    _snapshot = JSON.parse(out)
  } catch (err) {
    _snapshotError = err
    _snapshot = {}
  }
  return _snapshot
}

/**
 * clientReachableHost — 将服务监听地址转为浏览器可访问地址：
 * 0.0.0.0 / 空 / 局域网地址 → 127.0.0.1；已是回环地址则原样返回。
 *
 * @param {string|undefined|null} host
 * @param {string} [fallback='127.0.0.1']
 * @returns {string}
 */
export function clientReachableHost(host, fallback = '127.0.0.1') {
  const h = String(host ?? '').trim().replace(/^https?:\/\//, '')
  if (!h || h === '0.0.0.0' || h === '::' || h === 'localhost') return fallback
  return h
}

// 测试接缝：允许直接查看快照加载失败原因
export function _loadPortConfigError() {
  return _snapshotError
}
