import { describe, expect, it } from 'vitest'
import {
  replaceUserdataRuntimePlaceholdersForPreview,
  rewriteBrowserTaskDetailUrlToCloudPrefixInText,
} from './userdataContainerImageReplace.js'

describe('rewriteBrowserTaskDetailUrlToCloudPrefixInText', () => {
  it('将浏览器任务详情 URL 替换为 cloud_prefix', () => {
    const cp = 'http://api.test/api/tenant/t1/workspace/w1/task/k9/comment/cmt_1/cloud'
    const inText =
      'docker run -e TaskApiEndPoint=http://app.test/tenant/t1/workspace/w1/task-detail/k9/ \\\n' +
      '  img'
    const out = rewriteBrowserTaskDetailUrlToCloudPrefixInText(
      inText,
      cp,
      't1',
      'w1',
      'k9'
    )
    expect(out).not.toContain('task-detail')
    expect(out).toContain(cp)
  })

  it('ID 不匹配时不替换', () => {
    const cp = 'http://api.test/api/tenant/a/workspace/b/task/c/comment/cmt_1/cloud'
    const inText = 'http://x.com/tenant/a/workspace/b/task-detail/wrong/'
    const out = rewriteBrowserTaskDetailUrlToCloudPrefixInText(
      inText,
      cp,
      'a',
      'b',
      'c'
    )
    expect(out).toBe(inText)
  })
})

describe('replaceUserdataRuntimePlaceholdersForPreview', () => {
  it('占位符替换后对误粘任务详情 URL 纠错（与后端 RunInstances 预览对齐）', () => {
    const plain = 'LINE=http://host/tenant/10/workspace/20/task-detail/30/'
    const { text, missingSlots } = replaceUserdataRuntimePlaceholdersForPreview(plain, {
      task_api_endpoint: 'http://host',
      tenant_id: '10',
      workspace_id: '20',
      task_id: '30',
      comment_id: 'cmt_1',
      container_image_url: 'nginx',
      access_token: 'x',
      ssh_public_key: 'k',
      ssh_match_address: '0.0.0.0/0',
    })
    expect(missingSlots).toEqual([])
    expect(text).toContain('http://host/api/tenant/10/workspace/20/task/30/comment/cmt_1/cloud')
    expect(text).not.toContain('task-detail')
  })

  it('无 comment_id 时不生成旧 …/task/{id}/cloud', () => {
    const plain = 'EP=__TASK2APP_TASK_CLOUD_PREFIX__ LINE=http://host/tenant/10/workspace/20/task-detail/30/'
    const { text } = replaceUserdataRuntimePlaceholdersForPreview(plain, {
      task_api_endpoint: 'http://host',
      tenant_id: '10',
      workspace_id: '20',
      task_id: '30',
      container_image_url: 'nginx',
      access_token: 'x',
      ssh_public_key: 'k',
      ssh_match_address: '0.0.0.0/0',
    })
    expect(text).toContain('__TASK2APP_TASK_CLOUD_PREFIX__')
    expect(text).not.toContain('/task/30/cloud')
  })

  it('有 comment_id 时 cloud_prefix 含 /comment/{cid}/', () => {
    const plain = 'EP=__TASK2APP_TASK_CLOUD_PREFIX__'
    const { text, missingSlots } = replaceUserdataRuntimePlaceholdersForPreview(plain, {
      task_api_endpoint: 'http://host',
      tenant_id: '10',
      workspace_id: '20',
      task_id: '30',
      comment_id: 'cmt_1',
      container_image_url: 'nginx',
      access_token: 'x',
      ssh_public_key: 'k',
      ssh_match_address: '0.0.0.0/0',
    })
    expect(missingSlots).toEqual([])
    expect(text).toContain('http://host/api/tenant/10/workspace/20/task/30/comment/cmt_1/cloud')
  })
})
