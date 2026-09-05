import { describe, it, expect } from 'vitest'
import { useServerSectionBodyExpanded } from './useServerSectionBodyExpanded.js'

describe('useServerSectionBodyExpanded', () => {
  it('默认收起', () => {
    const { serverSectionBodyExpanded } = useServerSectionBodyExpanded()
    expect(serverSectionBodyExpanded.value).toBe(false)
  })

  it('toggle 在展开与折叠间切换', () => {
    const { serverSectionBodyExpanded, toggleServerSectionBody } = useServerSectionBodyExpanded()
    toggleServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(true)
    toggleServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(false)
  })

  it('expand 在折叠后强制展开（切换 Tab 场景）', () => {
    const {
      serverSectionBodyExpanded,
      expandServerSectionBody,
    } = useServerSectionBodyExpanded()
    expect(serverSectionBodyExpanded.value).toBe(false)
    expandServerSectionBody()
    expect(serverSectionBodyExpanded.value).toBe(true)
  })
})
