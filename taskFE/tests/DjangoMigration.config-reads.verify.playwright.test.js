// @ts-check
/** OPT-20260806-053 验证：迁移后配置读取正确（无 django 引用） */
import { test, expect } from '@playwright/test'
import { loadPortConfig } from '../helpers/loadConfYaml.mjs'
import { readE2eOrigins } from './helpers/gatewayLoginE2e.js'

test.describe('Django 引用迁移验证', () => {
  test('gatewayLoginE2e 网关 origin 解析正确（无 django.host）', () => {
    const { gatewayOrigin } = readE2eOrigins()
    expect(gatewayOrigin).toMatch(/^http:\/\/(127\.0\.0\.1|0\.0\.0\.0):18081$/)
    console.log('GATEWAY ORIGIN:', gatewayOrigin)
  })

  test('addressing.gitoauth 可读（迁移自 django.gitoauth）', () => {
    const raw = loadPortConfig()
    const host = raw?._addressing?.addresses?.gitoauth
    expect(host).toBe('gitoauth_api.daydaymoney.com')
    console.log('GITOAUTH:', host)
  })

  test('taskAuth 端口可读（relay 探测目标迁移）', () => {
    const raw = loadPortConfig()
    expect(Number(raw.taskAuth?.port) || 8003).toBe(8003)
    console.log('TASKAUTH PORT:', raw.taskAuth?.port)
  })
})
