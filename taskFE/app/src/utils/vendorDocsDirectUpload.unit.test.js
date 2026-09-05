import { describe, expect, it } from 'vitest'
import {
  buildDirectUploadHeaders,
  hasPostFormFields,
  isSameOriginUploadURL,
  uploadIssuedFile,
} from './vendorDocsDirectUpload.js'

// 契约与 taskAiProvider/frontend/tests/directUpload.unit.test.js 保持一致（OPT-20260901-008）：
// COS 预签名直传跨仓双份实现，任何一处桶 CORS/预签名字段变更须同步两文件，
// 此处断言与 taskAiProvider 测例同构的 no-cors POST 路径与 Authorization 规则。

describe('vendorDocsDirectUpload', () => {
  it('COS form_fields 走 POST Object', () => {
    expect(hasPostFormFields({ form_fields: { key: 'k', policy: 'p' } })).toBe(true)
    expect(hasPostFormFields({ form_fields: {} })).toBe(false)
  })

  it('相对路径才算同源', () => {
    expect(isSameOriginUploadURL('/api/ai-provider/vendor-application/local-put/?x=1')).toBe(true)
    expect(isSameOriginUploadURL('https://cos.example/put')).toBe(false)
  })

  it('uploadIssuedFile COS POST 用 FormData、no-cors、不带 Authorization', async () => {
    const posts = []
    const file = { name: 'a.png', type: 'image/png', size: 8 }
    const issued = {
      file_key: 'k1',
      upload_url: 'https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/',
      method: 'POST',
      form_fields: {
        key: 'k1',
        policy: 'cG9saWN5',
        'q-sign-algorithm': 'sha1',
        'q-ak': 'AKIDxx',
        'q-key-time': '1;2',
        'q-signature': 'deadbeef',
        'x-cos-server-side-encryption': 'AES256',
      },
      headers: { 'Content-Type': 'image/png' },
      traceId: 'tid-post',
    }
    const fetchImpl = async (url, opts) => {
      posts.push({ url, opts })
      return { ok: false, type: 'opaque', status: 0, headers: { get: () => '' } }
    }
    await uploadIssuedFile(file, issued, { fetchImpl, bearerToken: 'vendor-secret-token' })
    expect(posts).toHaveLength(1)
    expect(posts[0].url).toBe(issued.upload_url)
    expect(posts[0].opts.method).toBe('POST')
    expect(posts[0].opts.mode).toBe('no-cors')
    expect(posts[0].opts.credentials).toBe('omit')
    expect(posts[0].opts.headers).toBeUndefined()
    expect(posts[0].opts.body).toBeInstanceOf(FormData)
    expect(posts[0].opts.body.get('key')).toBe('k1')
    expect(posts[0].opts.body.has('file')).toBe(true)
  })

  it('uploadIssuedFile COS POST Failed to fetch 不阻断（complete Head 确认）', async () => {
    const issued = {
      upload_url: 'https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/',
      method: 'POST',
      form_fields: { key: 'k1', policy: 'p' },
      traceId: 'tid-post',
    }
    const fetchImpl = async () => {
      throw new TypeError('Failed to fetch')
    }
    await uploadIssuedFile({ name: 'a.png', size: 1 }, issued, { fetchImpl })
  })

  it('uploadIssuedFile 同源 PUT 带 Bearer 且校验 ok', async () => {
    const puts = []
    const issued = {
      upload_url: '/api/vendor/image-groups/icon-local-put/?file_key=1',
      method: 'PUT',
      headers: { 'Content-Type': 'image/png' },
    }
    const fetchImpl = async (url, opts) => {
      puts.push({ url, opts })
      return { ok: true, headers: { get: () => '' } }
    }
    await uploadIssuedFile({ name: 'a.png', size: 8 }, issued, {
      fetchImpl,
      bearerToken: 'vendor-secret-token',
    })
    expect(puts[0].opts.credentials).toBe('include')
    expect(puts[0].opts.headers.Authorization).toBe('Bearer vendor-secret-token')
    expect(puts[0].opts.body.name).toBe('a.png')
  })

  it('buildDirectUploadHeaders COS URL 不带 Authorization', () => {
    const headers = buildDirectUploadHeaders({
      issuedHeaders: { 'Content-Type': 'image/png' },
      uploadURL: 'https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/',
      bearerToken: 'vendor-secret-token',
    })
    expect(headers.Authorization).toBeUndefined()
  })
})
