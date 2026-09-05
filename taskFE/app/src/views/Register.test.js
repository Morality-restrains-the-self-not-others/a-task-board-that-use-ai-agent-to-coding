// @vitest-environment node
import { describe, expect, it } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const src = fs.readFileSync(path.join(__dirname, 'Register.vue'), 'utf8')

// OPT-20260821-025: 父组件把本次提交 submitErrorTraceId 透传给字段级错误节点。
describe('Register.vue 字段级错误 traceId 透传', () => {
  it('PhoneRegister 接收 :trace-id="submitErrorTraceId"', () => {
    expect(src).toContain(':trace-id="submitErrorTraceId"')
  })
  it('EmailRegister 接收 :trace-id="submitErrorTraceId"', () => {
    expect(src).toContain(':trace-id="submitErrorTraceId"')
  })
})
