import { describe, expect, it } from 'vitest'
import {
  buildServerRuntimeStatusDetails,
  formatRuntimeTimestampDisplay,
  resolveRuntimeUptimeSource,
} from './serverRuntimeStatusDetails.js'

describe('serverRuntimeStatusDetails', () => {
  it('shows server start time from open runtime session', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      platform: 'aliyun',
      region: 'cn-hongkong',
      instance_id: 'i-test',
      server_started_at: '2026-07-08T02:40:14Z',
      instance_attribute: {
        body: {
          Status: 'Running',
          CreationTime: '2026-01-01T00:00:00Z',
        },
      },
    }, Date.parse('2026-07-08T03:40:14Z'))

    const startRow = rows.find((row) => row.label === '服务器启动时间')
    expect(startRow).toBeTruthy()
    expect(startRow.value).toContain('2026-07-08T02:40:14Z')
    expect(startRow.value).toContain('本地：')

    const uptimeRow = rows.find((row) => row.label === '已运行时长')
    expect(uptimeRow?.value).toBe('1 小时')
  })

  it('prefers server_started_at over CreationTime for uptime source', () => {
    expect(resolveRuntimeUptimeSource({
      server_started_at: '2026-07-08T02:40:14Z',
      instance_attribute: { body: { CreationTime: '2026-01-01T00:00:00Z' } },
    })).toBe('2026-07-08T02:40:14Z')
  })

  it('shows server start time even without cloud instance attributes', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: null,
      message: '该任务尚未创建云实例',
      server_started_at: '2026-07-08T02:40:14Z',
    }, Date.now())

    expect(rows.some((row) => row.label === '服务器启动时间')).toBe(true)
    expect(rows.some((row) => row.label === '创建时间')).toBe(false)
  })

  it('shows expected auto release time from auto_release_time', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      platform: 'aliyun',
      region: 'cn-hongkong',
      instance_id: 'i-test',
      auto_release_time: '2026-07-13T06:30:00Z',
      instance_attribute: {
        body: {
          Status: 'Running',
          CreationTime: '2026-07-13T06:00:00Z',
        },
      },
    }, Date.parse('2026-07-13T06:10:00Z'))

    const releaseRow = rows.find((row) => row.label === '服务器预计释放时间')
    expect(releaseRow).toBeTruthy()
    expect(releaseRow.value).toContain('2026-07-13T06:30:00Z')
    expect(releaseRow.value).toContain('本地：')
  })

  it('shows expected auto release time from instance_attribute.AutoReleaseTime', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      instance_attribute: {
        body: {
          Status: 'Running',
          AutoReleaseTime: '2026-07-13T07:00:00Z',
        },
      },
    }, Date.now())

    const releaseRow = rows.find((row) => row.label === '服务器预计释放时间')
    expect(releaseRow?.value).toContain('2026-07-13T07:00:00Z')
  })

  it('omits expected release time when auto release is unset', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      instance_attribute: {
        body: { Status: 'Running' },
      },
    }, Date.now())

    expect(rows.some((row) => row.label === '服务器预计释放时间')).toBe(false)
  })

  it('formats invalid timestamp as raw string', () => {
    expect(formatRuntimeTimestampDisplay('not-a-date')).toBe('not-a-date')
  })

  it('shows internet charge type and bandwidth from instance attributes', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      platform: 'aliyun',
      region: 'cn-qingdao',
      instance_id: 'i-m5ehuk0i7rfpobe9v39c',
      instance_attribute: {
        body: {
          Status: 'Running',
          InternetChargeType: 'PayByTraffic',
          InternetMaxBandwidthOut: 5,
          InternetMaxBandwidthIn: 5,
        },
      },
    }, Date.now())

    expect(rows.find((row) => row.label === '带宽计费模式')?.value).toBe('按流量计费')
    expect(rows.find((row) => row.label === '公网出带宽')?.value).toBe('5 Mbps')
    expect(rows.find((row) => row.label === '公网入带宽')?.value).toBe('5 Mbps')
  })

  it('falls back to EIP bandwidth when instance internet bandwidth is absent', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      instance_attribute: {
        body: {
          Status: 'Running',
          EipAddress: {
            InternetChargeType: 'PayByBandwidth',
            Bandwidth: 10,
          },
        },
      },
    }, Date.now())

    expect(rows.find((row) => row.label === '带宽计费模式')?.value).toBe('按带宽计费')
    expect(rows.find((row) => row.label === '公网出带宽')?.value).toBe('10 Mbps')
    expect(rows.some((row) => row.label === '公网入带宽')).toBe(false)
  })

  it('falls back to EIP bandwidth when instance outbound bandwidth is zero', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      instance_attribute: {
        body: {
          Status: 'Running',
          InternetChargeType: 'PayByTraffic',
          InternetMaxBandwidthOut: 0,
          EipAddress: {
            InternetChargeType: 'PayByBandwidth',
            Bandwidth: 10,
          },
        },
      },
    }, Date.now())

    expect(rows.find((row) => row.label === '带宽计费模式')?.value).toBe('按带宽计费')
    expect(rows.find((row) => row.label === '公网出带宽')?.value).toBe('10 Mbps')
  })

  it('appends lag from comment created_at onto 创建时间 when they differ by minutes', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      instance_attribute: {
        body: {
          Status: 'Running',
          CreationTime: '2026-08-29T02:57:00Z',
        },
      },
    }, Date.parse('2026-08-29T03:00:00Z'), '', '2026-08-29T01:42:46.000000000Z')

    const creation = rows.find((row) => row.label === '创建时间')
    expect(creation?.value).toContain('2026-08-29T02:57:00Z')
    expect(creation?.value).toContain('距评论')
    expect(creation?.value).toMatch(/1 小时/)
  })

  it('omits comment lag when creation is within a minute of the comment', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      instance_attribute: {
        body: { Status: 'Running', CreationTime: '2026-08-29T01:42:50Z' },
      },
    }, Date.now(), '', '2026-08-29T01:42:46Z')
    const creation = rows.find((row) => row.label === '创建时间')
    expect(creation?.value).toContain('2026-08-29T01:42:50Z')
    expect(creation?.value).not.toContain('距评论')
  })

  it('omits bandwidth rows when charge type and bandwidth are unset', () => {
    const rows = buildServerRuntimeStatusDetails({
      status: 'success',
      runtime_status: 'Running',
      instance_attribute: {
        body: { Status: 'Running' },
      },
    }, Date.now())

    expect(rows.some((row) => row.label === '带宽计费模式')).toBe(false)
    expect(rows.some((row) => row.label === '公网出带宽')).toBe(false)
    expect(rows.some((row) => row.label === '公网入带宽')).toBe(false)
  })
})
