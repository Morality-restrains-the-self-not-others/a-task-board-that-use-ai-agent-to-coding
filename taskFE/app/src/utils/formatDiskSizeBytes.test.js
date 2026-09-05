import { describe, expect, it } from 'vitest'
import { formatDiskSizeBytes } from './formatDiskSizeBytes.js'

describe('formatDiskSizeBytes', () => {
  it('formats bytes through GB', () => {
    expect(formatDiskSizeBytes(0)).toBe('0 B')
    expect(formatDiskSizeBytes(512)).toBe('512 B')
    expect(formatDiskSizeBytes(2048)).toBe('2 KB')
    expect(formatDiskSizeBytes(1288490188)).toBe('1.2 GB')
  })

  it('rejects invalid input', () => {
    expect(() => formatDiskSizeBytes(-1)).toThrow(/invalid bytes/)
    expect(() => formatDiskSizeBytes(NaN)).toThrow(/invalid bytes/)
  })
})
