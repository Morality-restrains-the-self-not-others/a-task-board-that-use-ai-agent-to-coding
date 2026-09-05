// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { avatarBackgroundFromSeed, initialsAvatarDataUri } from './initialsAvatarDataUri.js'

describe('initialsAvatarDataUri', () => {
  it('returns data URI without external URL', () => {
    const uri = initialsAvatarDataUri('alice')
    expect(uri.startsWith('data:image/svg+xml,')).toBe(true)
    expect(uri).not.toContain('dicebear')
    expect(uri).not.toMatch(/https?:\/\/api\./)
  })

  it('uses first letter uppercased', () => {
    const uri = decodeURIComponent(initialsAvatarDataUri('bob'))
    expect(uri).toContain('>B<')
  })

  it('empty seed shows ? instead of confusing U', () => {
    const uri = decodeURIComponent(initialsAvatarDataUri(''))
    expect(uri).toContain('>?</text>')
    expect(uri).not.toContain('>U</text>')
  })

  it('picks stable background from seed', () => {
    expect(avatarBackgroundFromSeed('user-a')).toBe(avatarBackgroundFromSeed('user-a'))
    expect(avatarBackgroundFromSeed('user-a')).not.toBe(avatarBackgroundFromSeed('user-b'))
  })
})

