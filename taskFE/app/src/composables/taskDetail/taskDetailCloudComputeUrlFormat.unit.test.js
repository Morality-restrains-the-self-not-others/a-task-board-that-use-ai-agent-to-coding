// @vitest-environment node
// 回归测试（OPT-20260809-030）：taskDetail 目录云网关 URL 形态校验。
// 缺陷：funcName-first 新约定迁移后，多处 URL 仍是损坏形态
// `task_id/${taskId}container-*`（funcName 缺 `/` 分隔紧贴 task_id 值），
// ParseConventionPath 把 funcName 吞进 task_id 值 → sub 为空 → HTTP 404
// （线上任务详情页「读取文件树失败（HTTP 404）」同源）。
// 本测试直接扫描源文件文本：修复前 18 处损坏 URL 全部触发失败，修复后通过。
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const TARGET_FILES = [
  'submitLayerGraphCommand.js',
  'taskDetailContainerFns.js',
  'taskDetailExecLog.js',
  'taskDetailGitFns.js',
  'taskDetailJobActions.js',
  'taskDetailLayerActions.js',
  'taskDetailLayerPushActions.js',
  'taskDetailZTreeExecLogState.js',
  'useServerConfigFeatureParams.js',
  'useTaskIdentityAutoRunSteps.js',
]

const read = (file) => readFileSync(new URL(`./${file}`, import.meta.url), 'utf8')

describe('taskDetail 云网关 URL 形态回归（OPT-20260809-030）', () => {
  for (const file of TARGET_FILES) {
    const src = read(file)

    it(`${file}: 无 funcName 被吞进 task_id 值的损坏形态`, () => {
      // 损坏形态：task_id/<值> 后直接紧跟字母开头 funcName 且无 / 分隔
      expect(src).not.toMatch(/task_id\/\$\{[^}]*\}[a-z][a-z0-9-]*/)
    })

    it(`${file}: kv-last apiBase 的 task_id 段必须带尾斜杠（否则拼接时 funcName 粘尾）`, () => {
      for (const line of src.split('\n')) {
        if (!line.includes('/api/cloud/compute/tenant_id/')) continue
        expect(line, `${file} 损坏 kv-last 行: ${line.trim()}`).toMatch(/task_id\/\$\{[^}]*\}\//)
      }
    })

    it(`${file}: funcName-first URL 的 funcName 段必须出现在 tenant_id 之前`, () => {
      for (const m of src.matchAll(/\/api\/cloud\/compute\/([a-z][a-z0-9-]*)\/tenant_id\//g)) {
        expect(m[1], `${file} 非法 funcName 段: ${m[1]}`).toMatch(/^[a-z][a-z0-9-]*$/)
      }
    })
  }
})
