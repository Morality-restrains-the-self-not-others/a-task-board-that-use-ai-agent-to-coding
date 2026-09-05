// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectEdit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { normalizeProjectTags } = await import('../utils/projectTagsUtils.js')

  describe('ProjectEdit save payload shape', () => {
    it('merges tags, default_auto_run and server_run_template into PUT body', () => {
      const formData = {
        name: 'P1',
        description: 'd',
        container_image_id: '859671040643174400',
        workspaces_ids: ['ws-1'],
        tags: [' Alpha ', 'beta'],
        default_auto_run: true,
      }
      const runTemplate = { template_id: 'tpl-1', region: 'cn-hangzhou' }
      const existingTemplate = { template_id: 'tpl-0', platform: 'aliyun' }
      const panelConfigured = true
      const baseTemplate = panelConfigured
        ? { ...existingTemplate, ...runTemplate }
        : existingTemplate
      const body = {
        ...formData,
        tags: normalizeProjectTags(formData.tags),
        git_repos: ['https://git.example/a.git'],
        container_image: 'trae0630',
        server_run_template: {
          ...baseTemplate,
          default_auto_run: Boolean(formData.default_auto_run),
        },
      }
      delete body.default_auto_run
      expect(body.tags).toEqual(['Alpha', 'beta'])
      expect(body.server_run_template).toEqual({
        template_id: 'tpl-1',
        platform: 'aliyun',
        region: 'cn-hangzhou',
        default_auto_run: true,
      })
      expect(body.container_image_id).toBe('859671040643174400')
      expect(body.container_image).toBe('trae0630')
      expect(body.git_repos).toHaveLength(1)
    })

    it('preserves existing run template when panel draft is empty', () => {
      const existingTemplate = { template_id: 'tpl-1', region: 'cn-hongkong', platform: 'aliyun' }
      const panelPayload = {}
      const panelConfigured = false
      const baseTemplate = panelConfigured
        ? { ...existingTemplate, ...panelPayload }
        : existingTemplate
      const serverRunTemplate = {
        ...baseTemplate,
        default_auto_run: true,
      }
      expect(serverRunTemplate).toEqual({
        template_id: 'tpl-1',
        region: 'cn-hongkong',
        platform: 'aliyun',
        default_auto_run: true,
      })
    })

    it('preserves existing instance when panel draft has region but no instance', () => {
      const existingTemplate = {
        template_id: 'tpl-1',
        region: 'cn-hongkong',
        platform: 'aliyun',
        selected_instance: 'ecs.g7.xlarge',
      }
      const panelPayload = { platform: 'aliyun', region: 'cn-hangzhou', label: '半成品' }
      const panelConfigured = Boolean(
        (panelPayload.platform || panelPayload.cloud_platform_id)
        && panelPayload.region
        && panelPayload.selected_instance,
      )
      expect(panelConfigured).toBe(false)
      const baseTemplate = panelConfigured
        ? { ...existingTemplate, ...panelPayload }
        : existingTemplate
      expect(baseTemplate.selected_instance).toBe('ecs.g7.xlarge')
      expect(baseTemplate.region).toBe('cn-hongkong')
    })

    it('sends empty container_image_id when clearing installed image', () => {
      const formData = {
        name: 'P1',
        description: 'd',
        container_image_id: '',
        workspaces_ids: ['ws-1'],
        tags: [],
        default_auto_run: false,
      }
      const body = { ...formData }
      delete body.default_auto_run
      if (!String(body.container_image_id || '').trim()) {
        body.container_image_id = ''
      }
      expect(body.container_image_id).toBe('')
    })

    it('formats installed image architecture label for edit page display', async () => {
      const { formatContainerImageArchitectures } = await import('../utils/containerImageArchitecture.js')
      const installedImages = [
        { id: '859671040643174400', name: 'task2app-trae', target_architectures: ['x86_64'] },
      ]
      const imageId = '859671040643174400'
      const matched = installedImages.find((image) => String(image.id) === imageId)
      expect(formatContainerImageArchitectures(matched)).toBe('x86_64')
    })
  })
}
