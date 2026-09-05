import { describe, expect, it } from 'vitest'
import {
  canonicalGitRepoUrl,
  gitRepoAddressMismatch,
  gitRepoUrlsEqual,
} from './gitRepoUrl.js'

describe('canonicalGitRepoUrl', () => {
  it('strips .git, trailing slash, and lowercases', () => {
    expect(canonicalGitRepoUrl('  https://github.com/Ruandao/Helloworld.git/  ')).toBe(
      'https://github.com/ruandao/helloworld',
    )
  })
})

describe('gitRepoUrlsEqual', () => {
  it('treats .git suffix as the same repo', () => {
    expect(gitRepoUrlsEqual(
      'https://github.com/acme/demo',
      'https://github.com/acme/demo.git',
    )).toBe(true)
  })

  it('treats different GitHub owners as different repos', () => {
    expect(gitRepoUrlsEqual(
      'https://github.com/ruandao/helloworld',
      'https://github.com/test-ruandao/helloworld.git',
    )).toBe(false)
  })
})

describe('gitRepoAddressMismatch', () => {
  it('is true when stored owner differs from current project URL', () => {
    expect(gitRepoAddressMismatch(
      'https://github.com/ruandao/helloworld',
      'https://github.com/test-ruandao/helloworld.git',
    )).toBe(true)
  })

  it('is false when only .git suffix differs', () => {
    expect(gitRepoAddressMismatch(
      'https://github.com/acme/demo',
      'https://github.com/acme/demo.git',
    )).toBe(false)
  })

  it('is false when either side is empty', () => {
    expect(gitRepoAddressMismatch('', 'https://github.com/acme/demo.git')).toBe(false)
    expect(gitRepoAddressMismatch('https://github.com/acme/demo', '')).toBe(false)
  })
})
