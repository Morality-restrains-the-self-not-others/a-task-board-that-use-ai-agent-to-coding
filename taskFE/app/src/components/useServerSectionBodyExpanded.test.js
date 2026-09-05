import { describe, it, expect } from 'vitest'
import { useServerSectionBodyExpanded } from './useServerSectionBodyExpanded.js'

describe('useServerSectionBodyExpanded', () => {
  it('默认展开', () => {
    const { serverSectionBodyExpanded } = useServerSectionBodyExpanded()
    expect(serverSectionBodyExpanded.value).toBe(true)
  })

  it('toggle 在展开与折叠间切换', () => {
    const { serverSectionBodyExpanded, toggleServerSectionBody } = useServerSectionBodyExpanded()
    toggleServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(false)
    toggleServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(true)
  })

  it('expand 在折叠后强制展开（切换 Tab 场景）', () => {
    const {
      serverSectionBodyExpanded,
      toggleServerSectionBody,
      expandServerSectionBody,
    } = useServerSectionBodyExpanded()
    toggleServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(false)
    expandServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(true)
  })
})
