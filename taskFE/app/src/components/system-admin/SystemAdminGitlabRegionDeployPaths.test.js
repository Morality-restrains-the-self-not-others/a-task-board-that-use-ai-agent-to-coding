// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabRegionDeployPaths.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const {
    formatGitlabRegionServiceProcess,
    formatGitlabRegionServiceStart,
    deriveGitlabRegionServiceStart,
  } = await import('./gitlabRegionDeployPaths.js')
  const DeployPaths = (await import('./SystemAdminGitlabRegionDeployPaths.vue')).default

  describe('formatGitlabRegionServiceProcess', () => {
    it('拼接服务名与容器名', () => {
      expect(formatGitlabRegionServiceProcess('git-service', 'gitlab')).toBe(
        'git-service（容器: gitlab）'
      )
    })

    it('仅服务名', () => {
      expect(formatGitlabRegionServiceProcess('git-service-tencent-sh-1', '')).toBe(
        'git-service-tencent-sh-1'
      )
    })

    it('均空时占位', () => {
      expect(formatGitlabRegionServiceProcess('', '')).toBe('（未解析）')
    })
  })

  describe('formatGitlabRegionServiceStart', () => {
    it('现网与 SH-1 启动命令不同', () => {
      expect(deriveGitlabRegionServiceStart('git-service')).toBe('bash gitService/run.sh start')
      expect(deriveGitlabRegionServiceStart('git-service-tencent-sh-1')).toBe(
        'bash gitService/scripts/deploy_tencent_sh_1.sh'
      )
      expect(formatGitlabRegionServiceStart('git-service', 'bash gitService/run.sh start')).not.toBe(
        formatGitlabRegionServiceStart('git-service-tencent-sh-1', 'bash gitService/run.sh start')
      )
    })

    it('新增 slug 走 GITSERVICE_CONF_APP（与 taskBill gitlabRegionServiceStart 同表）', () => {
      expect(deriveGitlabRegionServiceStart('git-service-aws-tokyo-1')).toBe(
        'GITSERVICE_CONF_APP=git-service-aws-tokyo-1 bash gitService/run.sh start'
      )
    })

    it('API 误用现网命令时按服务进程纠正', () => {
      expect(
        formatGitlabRegionServiceStart('git-service-tencent-sh-1', 'bash gitService/run.sh start')
      ).toBe('bash gitService/scripts/deploy_tencent_sh_1.sh')
    })
  })

  describe('SystemAdminGitlabRegionDeployPaths', () => {
    it('展示服务进程、配置文件、GITLAB_HOME', () => {
      const wrapper = mount(DeployPaths, {
        props: {
          serviceProcess: 'git-service',
          serviceStart: 'bash gitService/run.sh start',
          configFile: 'conf/infra/git-service/config.yaml',
          dataDir: '/home/ljy/.local/share/daydaymoney/gitService',
          containerName: 'gitlab',
          configExists: true,
        },
      })
      expect(wrapper.find('[data-testid="gitlab-region-deploy-paths"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="gitlab-region-service-process"]').text()).toBe(
        'git-service（容器: gitlab）'
      )
      expect(wrapper.find('[data-testid="gitlab-region-config-file"]').text()).toBe(
        'conf/infra/git-service/config.yaml'
      )
      expect(wrapper.find('[data-testid="gitlab-region-gitlab-home-label"]').text()).toContain(
        'GITLAB_HOME'
      )
      expect(wrapper.find('[data-testid="gitlab-region-data-dir"]').text()).toBe(
        '/home/ljy/.local/share/daydaymoney/gitService'
      )
      expect(wrapper.find('[data-testid="gitlab-region-service-start"]').text()).toBe(
        'bash gitService/run.sh start'
      )
      expect(wrapper.find('[data-testid="gitlab-region-config-missing"]').exists()).toBe(false)
    })

    it('第二实例展示独立启动命令', () => {
      const wrapper = mount(DeployPaths, {
        props: {
          serviceProcess: 'git-service-tencent-sh-1',
          serviceStart: 'bash gitService/scripts/deploy_tencent_sh_1.sh',
          configFile: 'conf/infra/git-service-tencent-sh-1/config.yaml',
          dataDir: '/var/lib/daydaymoney/gitService-tencent-sh-1',
          containerName: 'gitlab-tencent-sh-1',
          configExists: true,
        },
      })
      expect(wrapper.find('[data-testid="gitlab-region-service-start"]').text()).toBe(
        'bash gitService/scripts/deploy_tencent_sh_1.sh'
      )
      expect(wrapper.find('[data-testid="gitlab-region-service-start"]').text()).not.toBe(
        'bash gitService/run.sh start'
      )
    })

    it('API 仍返回现网启动命令时按 SH-1 服务进程纠正', () => {
      const wrapper = mount(DeployPaths, {
        props: {
          serviceProcess: 'git-service-tencent-sh-1',
          serviceStart: 'bash gitService/run.sh start',
          configFile: 'conf/infra/git-service-tencent-sh-1/config.yaml',
        },
      })
      expect(wrapper.find('[data-testid="gitlab-region-service-start"]').text()).toBe(
        'bash gitService/scripts/deploy_tencent_sh_1.sh'
      )
    })

    it('配置缺失时提示文件不存在', () => {
      const wrapper = mount(DeployPaths, {
        props: {
          serviceProcess: 'git-service-aws-tokyo-1',
          configFile: 'conf/infra/git-service-aws-tokyo-1/config.yaml',
          dataDir: '',
          configExists: false,
        },
      })
      expect(wrapper.find('[data-testid="gitlab-region-config-missing"]').text()).toContain(
        '文件不存在'
      )
      expect(wrapper.find('[data-testid="gitlab-region-gitlab-home-label"]').text()).toContain(
        'GITLAB_HOME'
      )
      expect(wrapper.find('[data-testid="gitlab-region-data-dir"]').text()).toContain(
        '未配置 GITLAB_HOME'
      )
    })

    it('无路径字段时不渲染', () => {
      const wrapper = mount(DeployPaths, { props: {} })
      expect(wrapper.find('[data-testid="gitlab-region-deploy-paths"]').exists()).toBe(false)
    })
  })
}
