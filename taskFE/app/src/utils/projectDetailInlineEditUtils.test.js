// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] projectDetailInlineEditUtils.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    canEnableDefaultAutoRun,
    buildTagsPatchBody,
    buildImagePatchBody,
    buildAutoRunPatchBody,
    buildClearImagePatchBody,
  } = await import('./projectDetailInlineEditUtils.js')

  describe('projectDetailInlineEditUtils', () => {
    it('buildTagsPatchBody normalizes tags', () => {
      expect(buildTagsPatchBody([' Alpha ', 'beta', 'alpha'])).toEqual({
        tags: ['Alpha', 'beta'],
      })
    })

    it('buildImagePatchBody resolves name from installed list', () => {
      expect(
        buildImagePatchBody('42', [{ id: '42', name: 'ubuntu-dev' }, { id: '9', name: 'other' }]),
      ).toEqual({
        container_image_id: '42',
        container_image: 'ubuntu-dev',
      })
    })

    it('buildImagePatchBody attaches live run template when provided', () => {
      expect(
        buildImagePatchBody('42', [{ id: '42', name: 'ubuntu-dev' }], {
          platform: 'aliyun',
          region: 'cn-hangzhou',
          selected_instance: 'ecs.g7.xlarge',
        }),
      ).toEqual({
        container_image_id: '42',
        container_image: 'ubuntu-dev',
        server_run_template: {
          platform: 'aliyun',
          region: 'cn-hangzhou',
          selected_instance: 'ecs.g7.xlarge',
        },
      })
    })

    it('buildImagePatchBody omits incomplete live draft and uses complete saved template', () => {
      const incompleteLive = { platform: 'aliyun', region: 'cn-hangzhou', label: 'draft' }
      const saved = {
        platform: 'aliyun',
        region: 'cn-qingdao',
        cloud_platform_id: 'p1',
        selected_instance: 'ecs.c6.large',
      }
      expect(buildImagePatchBody('42', [{ id: '42', name: 'ubuntu-dev' }], incompleteLive, saved)).toEqual({
        container_image_id: '42',
        container_image: 'ubuntu-dev',
        server_run_template: saved,
      })
    })

    it('buildImagePatchBody omits incomplete template when saved is also incomplete', () => {
      expect(
        buildImagePatchBody(
          '42',
          [{ id: '42', name: 'ubuntu-dev' }],
          { platform: 'aliyun', region: 'cn-hangzhou' },
          { region: 'cn-beijing' },
        ),
      ).toEqual({
        container_image_id: '42',
        container_image: 'ubuntu-dev',
      })
    })

    it('buildImagePatchBody clears image fields when empty', () => {
      expect(buildImagePatchBody('', [])).toEqual({
        container_image_id: '',
        container_image: '',
      })
    })

    it('buildAutoRunPatchBody merges existing template', () => {
      const project = {
        server_run_template: { platform: 'aliyun', region: 'cn-hangzhou', default_auto_run: false },
      }
      expect(buildAutoRunPatchBody(project, true)).toEqual({
        server_run_template: {
          platform: 'aliyun',
          region: 'cn-hangzhou',
          default_auto_run: true,
        },
      })
    })

    it('buildClearImagePatchBody clears image and turns off auto_run', () => {
      const project = {
        server_run_template: { platform: 'aliyun', default_auto_run: true },
      }
      expect(buildClearImagePatchBody(project)).toEqual({
        container_image_id: '',
        container_image: '',
        server_run_template: {
          platform: 'aliyun',
          default_auto_run: false,
        },
      })
    })

    it('canEnableDefaultAutoRun requires image and configured template', () => {
      expect(canEnableDefaultAutoRun({ container_image_id: '', server_run_template: {} })).toBe(false)
      expect(
        canEnableDefaultAutoRun({
          container_image_id: '1',
          server_run_template: {},
        }),
      ).toBe(false)
      expect(
        canEnableDefaultAutoRun({
          container_image_id: '1',
          server_run_template: { platform: 'aliyun', region: 'cn-hangzhou' },
        }),
      ).toBe(true)
    })
  })
}
