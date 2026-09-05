import { describe, expect, it } from 'vitest'
import {
  GIT_REPO_URL_EXAMPLES,
  INVALID_REPO_URL_MSG,
  collectProjectGitRepoUrls,
  gitRepoRowsFormatError,
  gitRepoUrlFromUnknown,
  isValidGitRepoUrl,
} from './gitRepoUrlUtils.js'

describe('gitRepoUrlUtils', () => {
  it('accepts empty optional repo url', () => {
    expect(isValidGitRepoUrl('')).toBe(true)
    expect(isValidGitRepoUrl('   ')).toBe(true)
  })

  it('accepts https repo urls', () => {
    expect(isValidGitRepoUrl('https://github.com/owner/repo.git')).toBe(true)
    expect(isValidGitRepoUrl('https://127.0.0.1/group/project.git')).toBe(true)
  })

  it('accepts git@host:path scp-style urls', () => {
    expect(isValidGitRepoUrl('git@github.com:org/repo.git')).toBe(true)
    expect(isValidGitRepoUrl('git@gitlab.daydaymoney.com:example-user/somanyad-emailD.git')).toBe(true)
    expect(isValidGitRepoUrl('https://gitlab.daydaymoney.com/example-user/somanyad-emailD.git')).toBe(true)
    expect(isValidGitRepoUrl('git@gitlab.com:group/project')).toBe(true)
  })

  it('accepts ssh:// git urls including custom port', () => {
    expect(isValidGitRepoUrl('ssh://git@github.com/owner/repo.git')).toBe(true)
    expect(isValidGitRepoUrl('ssh://git@gitlab.daydaymoney.com:2222/group/project.git')).toBe(true)
    expect(isValidGitRepoUrl('SSH://git@github.com/owner/repo.git')).toBe(true)
  })

  it('rejects invalid repo urls', () => {
    expect(isValidGitRepoUrl('not-a-valid-url')).toBe(false)
    expect(isValidGitRepoUrl('ftp://example.com/repo.git')).toBe(false)
    expect(isValidGitRepoUrl('git@host-only')).toBe(false)
    expect(isValidGitRepoUrl('ssh://git@host-only')).toBe(false)
    expect(isValidGitRepoUrl('ssh://github.com')).toBe(false)
  })

  it('documents ssh example in format hint examples', () => {
    expect(GIT_REPO_URL_EXAMPLES.some((x) => x.startsWith('git@'))).toBe(true)
    expect(GIT_REPO_URL_EXAMPLES.some((x) => x.startsWith('ssh://'))).toBe(true)
  })

  it('extracts git repo url from string or {url, clone_alias}', () => {
    expect(gitRepoUrlFromUnknown('https://gitlab.example/g/a.git')).toBe('https://gitlab.example/g/a.git')
    expect(gitRepoUrlFromUnknown({
      url: 'https://gitlab.example/g/a.git',
      clone_alias: 'a',
    })).toBe('https://gitlab.example/g/a.git')
    expect(gitRepoUrlFromUnknown({ repo_url: 'https://gitlab.example/g/b.git' })).toBe(
      'https://gitlab.example/g/b.git',
    )
    expect(gitRepoUrlFromUnknown({})).toBe('')
  })

  it('collects project urls from git_repo_entries before git_repos objects', () => {
    expect(collectProjectGitRepoUrls({
      git_repo_entries: [{ url: 'https://gitlab.example/g/a.git' }],
      git_repos: [{ url: 'https://gitlab.example/g/ignored.git' }],
    })).toEqual(['https://gitlab.example/g/a.git'])
    expect(collectProjectGitRepoUrls({
      git_repos: [{ url: 'https://gitlab.example/g/b.git' }, 'https://gitlab.example/g/b.git'],
    })).toEqual(['https://gitlab.example/g/b.git'])
  })

  it('gitRepoRowsFormatError accepts ssh:// and rejects ftp://', () => {
    expect(
      gitRepoRowsFormatError([{ id: 1, url: 'ssh://git@host/path.git' }]),
    ).toBe('')
    expect(
      gitRepoRowsFormatError([{ id: 1, url: 'ftp://example.com/repo.git' }]),
    ).toBe(INVALID_REPO_URL_MSG)
    expect(gitRepoRowsFormatError([{ id: 1, url: '' }])).toBe('')
  })
})
