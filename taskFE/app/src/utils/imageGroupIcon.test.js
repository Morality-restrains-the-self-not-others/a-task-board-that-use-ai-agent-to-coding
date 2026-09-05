import test from 'node:test'
import assert from 'node:assert/strict'
import {
  imageGroupIconSrc,
  resolveInstalledImageIconSrc,
} from './imageGroupIcon.js'

test('imageGroupIconSrc 读取顶层或嵌套 image_group.icon_url', () => {
  assert.equal(
    imageGroupIconSrc({ icon_url: '/api/ai-provider/public-image-groups/1/icon?h=ab' }),
    '/api/ai-provider/public-image-groups/1/icon?h=ab',
  )
  assert.equal(imageGroupIconSrc({ image_group: { icon_url: '/x' } }), '/x')
  assert.equal(imageGroupIconSrc({}), '')
  assert.equal(imageGroupIconSrc(null), '')
})

test('resolveInstalledImageIconSrc 优先使用已安装记录自身 icon_url', () => {
  const src = resolveInstalledImageIconSrc(
    { icon_url: '/own', external_image_id: '41', name: 'trae-agent' },
    [{ id: '41', icon_url: '/catalog' }],
    [],
  )
  assert.equal(src, '/own')
})

test('resolveInstalledImageIconSrc 存量无快照时按 external_image_id 命中目录', () => {
  const src = resolveInstalledImageIconSrc(
    { name: 'trae-agent', external_image_id: '7319654012760069' },
    [{ id: '7319654012760069', name: 'trae-agent', icon_url: '/api/ai-provider/public-image-groups/1/icon?h=ab' }],
    [],
  )
  assert.equal(src, '/api/ai-provider/public-image-groups/1/icon?h=ab')
})

test('resolveInstalledImageIconSrc 无 id 命中时按 name 回退开发中目录', () => {
  const src = resolveInstalledImageIconSrc(
    { name: 'draft-agent' },
    [],
    [{ name: 'draft-agent', image_group: { icon_url: '/dev-icon' } }],
  )
  assert.equal(src, '/dev-icon')
})

test('resolveInstalledImageIconSrc 目录也无图标时返回空串', () => {
  assert.equal(
    resolveInstalledImageIconSrc({ name: 'unknown' }, [{ name: 'other', icon_url: '/x' }], []),
    '',
  )
})
