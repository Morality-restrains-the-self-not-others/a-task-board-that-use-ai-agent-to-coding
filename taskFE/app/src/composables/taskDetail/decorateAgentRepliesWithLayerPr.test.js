// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { gitPrHtmlUrlOf } from './taskDetailGitPrReply.js'
import {
  collectPrHtmlUrlsFromZNodes,
  decorateAgentRepliesWithLayerPr,
  isContainerAgentPromptEcho,
} from './decorateAgentRepliesWithLayerPr.js'

describe('isContainerAgentPromptEcho', () => {
  it('matches @{image} plus parent body', () => {
    const parent = '【自动运行】\n写一个 hello world\n\n用 java'
    const child = `@trae-agent ${parent}`
    expect(isContainerAgentPromptEcho(child, parent)).toBe(true)
  })

  it('rejects unrelated agent content', () => {
    expect(isContainerAgentPromptEcho('@trae-agent done', '写一个 hello world')).toBe(false)
  })
})

describe('collectPrHtmlUrlsFromZNodes', () => {
  it('walks nested ztree nodes for prHtmlUrl', () => {
    const urls = collectPrHtmlUrlsFromZNodes([
      {
        name: 'root',
        children: [
          { prHtmlUrl: 'https://gitlab.example/a/b/-/merge_requests/2' },
          { prHtmlUrl: 'https://gitlab.example/a/b/-/merge_requests/2' },
        ],
      },
    ])
    expect(urls).toEqual(['https://gitlab.example/a/b/-/merge_requests/2'])
  })
})

describe('decorateAgentRepliesWithLayerPr', () => {
  it('hides prompt echo and attaches ztree PR on pending container_agent child', () => {
    const parentBody = '【自动运行】\n写一个 hello world\n\n用 java'
    const pr = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/2'
    const rows = decorateAgentRepliesWithLayerPr(
      [
        {
          id: 'cmt_parent',
          commentKind: 'user',
          content: parentBody,
          children: [
            {
              id: 'cmt_agent',
              commentKind: 'container_agent',
              content: `@trae-agent ${parentBody}`,
              assistant_response: null,
              run_status: 'pending',
            },
          ],
        },
      ],
      [pr],
    )
    const child = rows[0].children[0]
    expect(child.content).toBe('')
    expect(gitPrHtmlUrlOf(child)).toBe(pr)
    expect(child.git_pr.provider).toBe('gitlab')
  })

  it('does not overwrite an existing git_pr', () => {
    const rows = decorateAgentRepliesWithLayerPr(
      [
        {
          id: 'p',
          content: 'hello',
          children: [
            {
              id: 'a',
              commentKind: 'container_agent',
              content: '@img hello',
              git_pr: { html_url: 'https://github.com/acme/x/pull/1', provider: 'github' },
            },
          ],
        },
      ],
      ['https://gitlab.example/a/b/-/merge_requests/9'],
    )
    expect(gitPrHtmlUrlOf(rows[0].children[0])).toBe('https://github.com/acme/x/pull/1')
  })

  it('omits pending prompt-echo child with no assistant_response or PR (platform placeholder)', () => {
    const parentBody = '【自动运行】\n写一个 hello world'
    const rows = decorateAgentRepliesWithLayerPr(
      [
        {
          id: 'cmt_parent',
          commentKind: 'user',
          content: parentBody,
          children: [
            {
              id: 'cmt_agent',
              commentKind: 'container_agent',
              content: `@trae-agent ${parentBody}`,
              assistant_response: null,
              run_status: 'pending',
            },
          ],
        },
      ],
      [],
    )
    expect(rows[0].children).toEqual([])
  })

  it('keeps pending child once the container wrote assistant_response', () => {
    const rows = decorateAgentRepliesWithLayerPr(
      [
        {
          id: 'p',
          commentKind: 'user',
          content: 'do work',
          children: [
            {
              id: 'a',
              commentKind: 'container_agent',
              content: '@img do work',
              assistant_response: 'done',
              run_status: 'completed',
            },
          ],
        },
      ],
      [],
    )
    expect(rows[0].children).toHaveLength(1)
    expect(rows[0].children[0].assistant_response).toBe('done')
  })

  it('keeps empty child while container is streaming or actively writing', () => {
    const parent = {
      id: 'p',
      commentKind: 'user',
      content: 'do work',
      children: [
        {
          id: 'a',
          commentKind: 'container_agent',
          content: '@img do work',
          assistant_response: '',
          run_status: 'streaming',
        },
      ],
    }
    const streaming = decorateAgentRepliesWithLayerPr([parent], [])
    expect(streaming[0].children).toHaveLength(1)

    const pendingButLiveStream = decorateAgentRepliesWithLayerPr(
      [{
        ...parent,
        children: [{ ...parent.children[0], run_status: 'pending' }],
      }],
      [],
      { activeAgentId: 'a', streamBusy: true, streamText: 'hello' },
    )
    expect(pendingButLiveStream[0].children).toHaveLength(1)

    const starting = decorateAgentRepliesWithLayerPr(
      [{
        ...parent,
        children: [{ ...parent.children[0], run_status: 'starting' }],
      }],
      [],
    )
    expect(starting[0].children).toHaveLength(1)
  })

  it('keeps container-created non-echo content such as edit_run placeholder text', () => {
    const rows = decorateAgentRepliesWithLayerPr(
      [
        {
          id: 'p',
          commentKind: 'user',
          content: 'do work',
          children: [
            {
              id: 'a',
              commentKind: 'container_agent',
              content: '修改指令后执行：交付中…',
              assistant_response: null,
              run_status: 'pending',
            },
          ],
        },
      ],
      [],
    )
    expect(rows[0].children).toHaveLength(1)
    expect(rows[0].children[0].content).toBe('修改指令后执行：交付中…')
  })
})
