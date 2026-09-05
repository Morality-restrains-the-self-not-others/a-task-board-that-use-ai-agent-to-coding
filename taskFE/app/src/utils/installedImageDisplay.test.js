// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  formatInstalledImageDisplayLabel,
  normalizeInstalledImageList,
  resolveInstalledImageById,
} from './installedImageDisplay.js'

describe('installedImageDisplay', () => {
  it('resolveInstalledImageById matches string snowflake ids', () => {
    const images = [{ id: '859671040643174400', name: 'trae-old' }]
    expect(resolveInstalledImageById(images, '859671040643174400')).toEqual(images[0])
    expect(resolveInstalledImageById(images, 859671040643174400)).toEqual(images[0])
  })

  it('formatInstalledImageDisplayLabel shows name version and architecture', () => {
    const label = formatInstalledImageDisplayLabel({
      image: { name: 'task2app-trae', version: 'v1', target_architectures: ['x86_64'] },
    })
    expect(label).toBe('task2app-trae (v1) · 架构 x86_64')
  })

  it('formatInstalledImageDisplayLabel uses storedName when image object missing', () => {
    expect(
      formatInstalledImageDisplayLabel({
        imageId: '859671040643174400',
        storedName: 'trae0630',
      }),
    ).toBe('trae0630')
  })

  it('does not hide an orphaned id behind the stored name once catalog loaded', () => {
    expect(
      formatInstalledImageDisplayLabel({
        image: null,
        imageId: '878236807722987520',
        storedName: 'trae-agent',
        treatMissingBindingAsUnavailable: true,
      }),
    ).toBe('镜像不可用或已卸载（ID：878236807722987520）')
  })

  it('formatInstalledImageDisplayLabel falls back to unavailable message instead of bare id', () => {
    expect(
      formatInstalledImageDisplayLabel({
        imageId: '859671040643174400',
      }),
    ).toBe('镜像不可用或已卸载（ID：859671040643174400）')
  })

  it('normalizeInstalledImageList coerces string ids from API', () => {
    const rows = normalizeInstalledImageList([{ id: '862412768836349952', name: 'trae' }])
    expect(rows[0].id).toBe('862412768836349952')
  })
})
