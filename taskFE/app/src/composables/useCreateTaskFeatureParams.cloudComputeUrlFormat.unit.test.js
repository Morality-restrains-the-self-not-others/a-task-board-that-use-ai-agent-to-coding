// @vitest-environment node
// 回归测试（OPT-20260809-030）：useCreateTaskFeatureParams 的云网关 URL 形态。
// 缺陷：feature-params-env-preview 曾以损坏形态 `task_id/${taskId}feature-params-env-preview/`
// 拼接，ParseConventionPath 把 funcName 吞进 task_id 值 → HTTP 404。
// 修复后为 funcName-first：compute/feature-params-env-preview/tenant_id/…。
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const src = readFileSync(new URL('./useCreateTaskFeatureParams.js', import.meta.url), 'utf8')

describe('useCreateTaskFeatureParams 云网关 URL 形态回归（OPT-20260809-030）', () => {
  it('feature-params-env-preview 为 funcName-first 形态', () => {
    expect(src).toContain('/api/cloud/compute/feature-params-env-preview/tenant_id/')
    expect(src).not.toMatch(/task_id\/\$\{[^}]*\}[a-z][a-z0-9-]*/)
  })
})
