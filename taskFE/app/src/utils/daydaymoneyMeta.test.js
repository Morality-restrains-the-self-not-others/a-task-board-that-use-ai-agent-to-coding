// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  mergeDaydaymoneyTags,
  metaTagsToHeaderContent,
  parseDaydaymoneyYaml,
} from './daydaymoneyMeta.js'

const SAMPLE = `
version: 1
service_id: taskProjectService
display_name: Task Project Service
tags:
  - svc:taskProjectService
  - domain:project
`

describe('parseDaydaymoneyYaml', () => {
  it('parses valid yaml', () => {
    const meta = parseDaydaymoneyYaml(SAMPLE)
    expect(meta.service_id).toBe('taskProjectService')
    expect(meta.display_name).toBe('Task Project Service')
    expect(meta.tags).toEqual(['svc:taskProjectService', 'domain:project'])
  })

  it('auto-injects svc tag when missing', () => {
    const meta = parseDaydaymoneyYaml(`
version: 1
service_id: foo
tags:
  - domain:bar
`)
    expect(meta.tags[0].toLowerCase()).toBe('svc:foo')
  })

  it('rejects forbidden top-level keys', () => {
    expect(() => parseDaydaymoneyYaml('service_id: x\nworkspace_id: w1')).toThrow(/禁止顶层键/)
    expect(() => parseDaydaymoneyYaml('service_id: x\nproject_id: p1')).toThrow(/禁止顶层键/)
    expect(() => parseDaydaymoneyYaml('service_id: x\ncompany_id: c1')).toThrow(/禁止顶层键/)
    expect(() => parseDaydaymoneyYaml('service_id: x\ntenant_id: t1')).toThrow(/禁止顶层键/)
  })

  it('rejects invalid service_id', () => {
    expect(() => parseDaydaymoneyYaml('service_id: 123bad\ntags:\n  - svc:123bad')).toThrow(/service_id/)
  })
})

describe('mergeDaydaymoneyTags', () => {
  it('merges without duplicates (case-insensitive)', () => {
    const merged = mergeDaydaymoneyTags(['svc:foo', 'team:a'], ['domain:saas', 'SVC:foo'], 'foo')
    expect(merged).toEqual(['svc:foo', 'team:a', 'domain:saas'])
  })

  it('ensures svc tag from serviceId', () => {
    const merged = mergeDaydaymoneyTags([], ['domain:x'], 'mySvc')
    expect(merged[0]).toBe('svc:mySvc')
  })
})

describe('metaTagsToHeaderContent', () => {
  it('joins normalized tags with comma', () => {
    expect(metaTagsToHeaderContent(['svc:task2app', 'domain:saas'])).toBe('svc:task2app,domain:saas')
  })
})
