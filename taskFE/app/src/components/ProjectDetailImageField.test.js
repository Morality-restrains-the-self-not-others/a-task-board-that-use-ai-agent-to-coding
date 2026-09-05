// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailImageField.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { default: ProjectDetailImageField } = await import('./ProjectDetailImageField.vue')

  const x86Template = {
    platform: 'aliyun',
    region: 'cn-qingdao',
    cloud_platform_id: 'p1',
    selected_instance: 'ecs.c6.large',
    hardware_config: { instance_type: 'ecs.c6.large' },
  }

  describe('ProjectDetailImageField architecture hint', () => {
    it('names image-required and instance-supported ISA when they mismatch', async () => {
      const wrapper = mount(ProjectDetailImageField, {
        props: {
          project: {
            container_image_id: 'img-x86',
            server_run_template: x86Template,
          },
          installedImages: [
            { id: 'img-x86', name: 'x86-img', version: 'x86_64-latest', target_architectures: ['x86_64'] },
            { id: 'img-arm', name: 'arm-img', version: 'arm64-latest', target_architectures: ['arm64'] },
          ],
          tenantId: 't1',
          projectId: 'proj_1',
          liveRunTemplateGetter: () => x86Template,
        },
      })
      await wrapper.find('[data-testid="project-container-image-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-container-image-select"]').setValue('img-arm')
      await nextTick()
      const hint = wrapper.find('[data-testid="project-image-arch-save-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('镜像要求的 CPU 架构（arm64）')
      expect(hint.text()).toContain('实例系统支持的 CPU 架构（x86_64（实例规格 ecs.c6.large））')
    })

    it('does not block save when empty declared arches infer x86_64 from version', async () => {
      const wrapper = mount(ProjectDetailImageField, {
        props: {
          project: {
            container_image_id: 'img-public',
            server_run_template: x86Template,
          },
          installedImages: [
            { id: 'img-public', name: 'trae-agent', version: 'x86_64-latest', target_architectures: ['x86_64'] },
            {
              id: 'img-private',
              name: 'trae-agent',
              version: 'private_x86_64-latest',
              target_architectures: [],
            },
          ],
          tenantId: 't1',
          projectId: 'proj_1',
          liveRunTemplateGetter: () => x86Template,
        },
      })
      await wrapper.find('[data-testid="project-container-image-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-container-image-select"]').setValue('img-private')
      await nextTick()
      expect(wrapper.find('[data-testid="project-image-arch-save-hint"]').exists()).toBe(false)
    })

    it('names both sides when declared ISA cannot be resolved', async () => {
      const wrapper = mount(ProjectDetailImageField, {
        props: {
          project: {
            container_image_id: 'img-x86',
            server_run_template: x86Template,
          },
          installedImages: [
            { id: 'img-x86', name: 'x86-img', version: 'x86_64-latest', target_architectures: ['x86_64'] },
            {
              id: 'img-unknown',
              name: 'trae-agent',
              version: 'latest',
              target_architectures: ['unknown'],
            },
          ],
          tenantId: 't1',
          projectId: 'proj_1',
          liveRunTemplateGetter: () => x86Template,
        },
      })
      await wrapper.find('[data-testid="project-container-image-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-container-image-select"]').setValue('img-unknown')
      await nextTick()
      const hint = wrapper.find('[data-testid="project-image-arch-save-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('镜像要求: unknown（无法识别为 x86_64/arm64）')
      expect(hint.text()).toContain('实例系统支持: x86_64（实例规格 ecs.c6.large）')
    })

    it('falls back to saved complete template when live draft lacks instance', async () => {
      const incompleteLive = {
        platform: 'aliyun',
        region: 'cn-qingdao',
        cloud_platform_id: 'p1',
        label: '半成品',
      }
      const wrapper = mount(ProjectDetailImageField, {
        props: {
          project: {
            container_image_id: 'img-x86',
            server_run_template: x86Template,
          },
          installedImages: [
            { id: 'img-x86', name: 'x86-img', version: 'x86_64-latest', target_architectures: ['x86_64'] },
            { id: 'img-x86-b', name: 'x86-b', version: 'x86_64-latest', target_architectures: ['x86_64'] },
          ],
          tenantId: 't1',
          projectId: 'proj_1',
          liveRunTemplateGetter: () => incompleteLive,
        },
      })
      await wrapper.find('[data-testid="project-container-image-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-container-image-select"]').setValue('img-x86-b')
      await nextTick()
      expect(wrapper.find('[data-testid="project-image-arch-save-hint"]').exists()).toBe(false)
    })

    it('shows orphaned binding instead of stored name after catalog loads', () => {
      const wrapper = mount(ProjectDetailImageField, {
        props: {
          project: {
            container_image_id: '878236807722987520',
            container_image: 'trae-agent',
            server_run_template: x86Template,
          },
          installedImages: [
            {
              id: '878236719185424384',
              name: 'trae-agent',
              version: 'x86_64-latest',
              target_architectures: ['x86_64'],
            },
          ],
          installedImagesLoaded: true,
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      expect(wrapper.get('[data-testid="project-container-image-display"]').text()).toContain(
        '镜像不可用或已卸载（ID：878236807722987520）',
      )
    })
  })
}
