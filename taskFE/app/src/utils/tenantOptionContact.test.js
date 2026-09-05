import { describe, expect, it } from 'vitest'
import { tenantContactLine, tenantContactParts } from './tenantOptionContact.js'

describe('tenantOptionContact', () => {
  it('joins phone and email', () => {
    expect(tenantContactParts({ phone: '13800138000', email: 'a@x.com' })).toEqual([
      '13800138000',
      'a@x.com',
    ])
    expect(tenantContactLine({ phone: '13800138000', email: 'a@x.com' })).toBe('13800138000 · a@x.com')
  })

  it('hides empty and placeholder 无', () => {
    expect(tenantContactParts({ phone: '无', email: '无' })).toEqual([])
    expect(tenantContactParts({ phone: '', email: '   ' })).toEqual([])
    expect(tenantContactLine({ email: 'only@x.com' })).toBe('only@x.com')
  })
})
