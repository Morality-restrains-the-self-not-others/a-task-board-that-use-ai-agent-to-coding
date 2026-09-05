import { describe, expect, it } from 'vitest'
import { fileTypeFromPath, fileTypeLabel } from './fileTypeFromPath.js'

describe('fileTypeFromPath', () => {
  it('T8: png → image', () => {
    expect(fileTypeFromPath('a/b/c.png')).toBe('image')
  })

  it('js/vue → code', () => {
    expect(fileTypeFromPath('src/App.vue')).toBe('code')
    expect(fileTypeFromPath('x.ts')).toBe('code')
  })

  it('modules 无扩展名 → binary', () => {
    expect(fileTypeFromPath('ram-work/jre/lib/modules')).toBe('binary')
  })

  it('md → text', () => {
    expect(fileTypeFromPath('README.md')).toBe('text')
    expect(fileTypeLabel('README.md')).toBe('文本')
  })
})
