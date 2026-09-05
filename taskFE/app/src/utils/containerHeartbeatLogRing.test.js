// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  CONTAINER_HEARTBEAT_LOG_RING_MAX,
  appendHeartbeatLogRing,
  formatHeartbeatLogLine,
  summarizeHeartbeatLogRing,
} from './containerHeartbeatLogRing.js'

describe('appendHeartbeatLogRing', () => {
  it('appends and trims to ring capacity', () => {
    const out = appendHeartbeatLogRing(['a', 'b'], ['c', 'd', 'e'], 3)
    expect(out).toEqual(['c', 'd', 'e'])
  })

  it('keeps all when under capacity', () => {
    expect(appendHeartbeatLogRing(['a'], ['b'], 5)).toEqual(['a', 'b'])
  })

  it('uses default max and ignores empty incoming', () => {
    const filled = Array.from({ length: CONTAINER_HEARTBEAT_LOG_RING_MAX }, (_, i) => `L${i}`)
    expect(appendHeartbeatLogRing(filled, []).length).toBe(CONTAINER_HEARTBEAT_LOG_RING_MAX)
    const next = appendHeartbeatLogRing(filled, ['newest'])
    expect(next).toHaveLength(CONTAINER_HEARTBEAT_LOG_RING_MAX)
    expect(next[next.length - 1]).toBe('newest')
    expect(next[0]).toBe('L1')
  })
})

describe('formatHeartbeatLogLine / summarizeHeartbeatLogRing', () => {
  it('formats structured http_request lines', () => {
    const line =
      '{"ts":"2026-07-09T08:00:12.345Z","level":"info","msg":"http_request","method":"GET","path":"/api/saas-heartbeat-probe?seq=1","status":"200","duration_ms":2}'
    expect(formatHeartbeatLogLine(line)).toContain('08:00:12')
    expect(formatHeartbeatLogLine(line)).toContain('saas-heartbeat-probe')
  })

  it('summarizes count and latest preview', () => {
    const s = summarizeHeartbeatLogRing(['old', 'latest-line'])
    expect(s.count).toBe(2)
    expect(s.latestPreview).toBe('latest-line')
    expect(summarizeHeartbeatLogRing([]).count).toBe(0)
  })
})
