'use strict'

const { describe, it } = require('node:test')
const assert = require('node:assert/strict')

const DaydaymoneyMeta = require('../lib/daydaymoney-meta.js')

describe('DaydaymoneyMeta.parseDaydaymoneyYaml', () => {
  it('parses valid yaml and injects svc tag', () => {
    const meta = DaydaymoneyMeta.parseDaydaymoneyYaml(`
version: 1
service_id: task2app
tags:
  - domain:saas
`)
    assert.equal(meta.service_id, 'task2app')
    assert.ok(meta.tags.some((t) => t.toLowerCase() === 'svc:task2app'))
  })

  it('rejects forbidden keys', () => {
    assert.throws(
      () => DaydaymoneyMeta.parseDaydaymoneyYaml('service_id: x\nworkspace_id: w'),
      /禁止顶层键/,
    )
  })
})

describe('DaydaymoneyMeta.readDaydaymoneyMetaFromDocument', () => {
  it('reads meta tags from document', () => {
    const doc = {
      querySelector(sel) {
        if (sel.includes('daydaymoney-service-id')) {
          return { getAttribute: () => 'task2app' }
        }
        if (sel.includes('daydaymoney-tags')) {
          return { getAttribute: () => 'svc:task2app,domain:saas' }
        }
        return null
      },
    }
    const meta = DaydaymoneyMeta.readDaydaymoneyMetaFromDocument(doc)
    assert.deepEqual(meta, {
      service_id: 'task2app',
      tags: ['svc:task2app', 'domain:saas'],
    })
  })

  it('returns null when service id missing', () => {
    const doc = { querySelector: () => null }
    assert.equal(DaydaymoneyMeta.readDaydaymoneyMetaFromDocument(doc), null)
  })
})

describe('DaydaymoneyMeta resolve helpers', () => {
  it('groups workspace and project ids', () => {
    const matches = [
      { workspace_id: 'w1', project_id: 'p1' },
      { workspace_id: 'w1', project_id: 'p2' },
      { workspace_id: 'w2', project_id: 'p3' },
    ]
    assert.deepEqual(DaydaymoneyMeta.uniqueWorkspaceIdsFromMatches(matches), ['w1', 'w2'])
    assert.deepEqual(DaydaymoneyMeta.projectIdsForWorkspace(matches, 'w1'), ['p1', 'p2'])
    assert.match(DaydaymoneyMeta.formatAidevResolveStatus(matches), /3 个项目/)
  })
})
