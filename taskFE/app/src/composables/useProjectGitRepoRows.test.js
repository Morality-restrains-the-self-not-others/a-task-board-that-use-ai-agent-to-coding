import { describe, expect, it } from 'vitest'
import { INVALID_REPO_URL_MSG } from '../utils/gitRepoUrlUtils.js'
import { useProjectGitRepoRows } from './useProjectGitRepoRows.js'

describe('useProjectGitRepoRows format validation', () => {
  it('allows ssh:// on submit and rejects ftp:// with format message', () => {
    const { gitRepoRows, refreshGitRepoFormatErrors, gitRepoRowFormatErrors } =
      useProjectGitRepoRows()
    gitRepoRows.value[0].url = 'ssh://git@host/path.git'
    expect(refreshGitRepoFormatErrors()).toBe('')
    expect(gitRepoRowFormatErrors.value).toEqual({})

    gitRepoRows.value[0].url = 'ftp://example.com/repo.git'
    expect(refreshGitRepoFormatErrors()).toBe(INVALID_REPO_URL_MSG)
    expect(gitRepoRowFormatErrors.value[gitRepoRows.value[0].id]).toBe(INVALID_REPO_URL_MSG)
  })

  it('loads object entries from API and keeps ssh:// payload', () => {
    const { setGitRepoRowsFromApi, trimmedGitRepoPayload, refreshGitRepoFormatErrors } =
      useProjectGitRepoRows()
    setGitRepoRowsFromApi({
      git_repo_entries: [
        { url: 'ssh://git@github.com/owner/repo.git', clone_alias: 'origin' },
      ],
    })
    expect(refreshGitRepoFormatErrors()).toBe('')
    expect(trimmedGitRepoPayload()).toEqual([
      { url: 'ssh://git@github.com/owner/repo.git', clone_alias: 'origin' },
    ])
  })
})
