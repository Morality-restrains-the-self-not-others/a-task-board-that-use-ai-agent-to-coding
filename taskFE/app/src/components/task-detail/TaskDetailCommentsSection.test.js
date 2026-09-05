/**
 * TaskDetailCommentsSection.feedDisplayComments 与 decorateAgentRepliesWithLayerPr 的接线回归。
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import {
  collectPrHtmlUrlsFromZNodes,
  decorateAgentRepliesWithLayerPr,
} from '../../composables/taskDetail/decorateAgentRepliesWithLayerPr.js'
import { gitPrHtmlUrlOf } from '../../composables/taskDetail/taskDetailGitPrReply.js'

function buildFeedDisplayComments(displayComments, layerGraphZNodes, opts = {}) {
  return decorateAgentRepliesWithLayerPr(
    displayComments,
    collectPrHtmlUrlsFromZNodes(layerGraphZNodes),
    opts,
  )
}

test('feedDisplayComments hides container_agent prompt echo and attaches ztree PR', () => {
  const parentBody = '【自动运行】\n写一个 hello world\n\n用 java'
  const pr = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/2'
  const rows = buildFeedDisplayComments(
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
          },
        ],
      },
    ],
    [{ children: [{ prHtmlUrl: pr }] }],
  )
  const child = rows[0].children[0]
  assert.equal(child.content, '')
  assert.equal(gitPrHtmlUrlOf(child), pr)
})

test('feedDisplayComments omits platform pending agent placeholder while server is still starting', () => {
  const parentBody = '【自动运行】\n写一个 hello world'
  const rows = buildFeedDisplayComments(
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
  assert.equal(rows[0].children.length, 0)
})
