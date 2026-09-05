import { describe, expect, it } from 'vitest'
import {
  pickPrimaryDefaultNetwork,
  shouldShowSameVpcHint,
} from './defaultMachineNetworkHint.js'

describe('shouldShowSameVpcHint', () => {
  it('is false for empty or invalid lists', () => {
    expect(shouldShowSameVpcHint([])).toBe(false)
    expect(shouldShowSameVpcHint(null)).toBe(false)
    expect(shouldShowSameVpcHint(undefined)).toBe(false)
  })

  it('is true when at least one default machine config exists', () => {
    expect(shouldShowSameVpcHint([{ authorization_id: 'a1', region: 'cn-hangzhou' }])).toBe(true)
  })
})

describe('pickPrimaryDefaultNetwork', () => {
  it('returns null when there is no default machine', () => {
    expect(pickPrimaryDefaultNetwork([])).toBeNull()
  })

  it('prefers the row that already has vpc_id', () => {
    const primary = pickPrimaryDefaultNetwork([
      { authorization_id: 'a0', platform: 'aliyun', region: 'cn-beijing' },
      {
        authorization_id: 'a1',
        platform_type: 'aliyun',
        region: 'cn-hangzhou',
        vpc_id: 'vpc-1',
        vswitch_id: 'vsw-1',
        zone_id: 'cn-hangzhou-h',
      },
    ])
    expect(primary).toEqual({
      authorization_id: 'a1',
      platform_type: 'aliyun',
      region: 'cn-hangzhou',
      vpc_id: 'vpc-1',
      vswitch_id: 'vsw-1',
      zone_id: 'cn-hangzhou-h',
    })
  })
})
