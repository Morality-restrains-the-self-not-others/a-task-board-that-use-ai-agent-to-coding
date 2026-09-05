<template>
  <div
    v-if="hasAny"
    class="text-xs text-text-light space-y-1 border-t border-gray-100 pt-2"
    data-testid="gitlab-region-deploy-paths"
  >
    <div class="font-medium text-text mb-1">部署路径（调整请改配置文件）</div>
    <div class="flex flex-wrap gap-x-1 gap-y-1 items-baseline">
      <span class="shrink-0 text-text-light">服务进程</span>
      <code
        class="bg-gray-100 px-1.5 py-0.5 rounded break-all"
        data-testid="gitlab-region-service-process"
      >{{ serviceProcessLabel }}</code>
    </div>
    <div class="flex flex-wrap gap-x-1 gap-y-1 items-baseline">
      <span class="shrink-0 text-text-light">配置文件</span>
      <code
        class="bg-gray-100 px-1.5 py-0.5 rounded break-all"
        data-testid="gitlab-region-config-file"
      >{{ configFile || '（未解析）' }}</code>
      <span
        v-if="configFile && configExists === false"
        class="text-amber-600"
        data-testid="gitlab-region-config-missing"
      >文件不存在</span>
    </div>
    <div class="flex flex-wrap gap-x-1 gap-y-1 items-baseline">
      <span
        class="shrink-0 text-text-light"
        data-testid="gitlab-region-gitlab-home-label"
        title="来自 conf gitlabHome，由 run.sh 导出为环境变量 GITLAB_HOME"
      >GITLAB_HOME</span>
      <code
        class="bg-gray-100 px-1.5 py-0.5 rounded break-all"
        data-testid="gitlab-region-data-dir"
      >{{ dataDir || '（未配置 GITLAB_HOME）' }}</code>
    </div>
    <div v-if="serviceStartLabel" class="flex flex-wrap gap-x-1 gap-y-1 items-baseline">
      <span class="shrink-0 text-text-light">启动命令</span>
      <code
        class="bg-gray-100 px-1.5 py-0.5 rounded break-all"
        data-testid="gitlab-region-service-start"
      >{{ serviceStartLabel }}</code>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  formatGitlabRegionServiceProcess,
  formatGitlabRegionServiceStart,
} from './gitlabRegionDeployPaths.js'

const props = defineProps({
  serviceProcess: { type: String, default: '' },
  serviceStart: { type: String, default: '' },
  configFile: { type: String, default: '' },
  dataDir: { type: String, default: '' },
  containerName: { type: String, default: '' },
  configExists: { type: Boolean, default: undefined },
})

const hasAny = computed(() =>
  Boolean(props.serviceProcess || props.configFile || props.dataDir || props.serviceStart)
)

const serviceProcessLabel = computed(() =>
  formatGitlabRegionServiceProcess(props.serviceProcess, props.containerName)
)

const serviceStartLabel = computed(() =>
  formatGitlabRegionServiceStart(props.serviceProcess, props.serviceStart)
)
</script>
