import { isWindowsOs, getOsTypeLabel, linuxFamilyForScript } from '../utils/userDataTemplate.js'
import { generateWindowsScript } from '../utils/userDataScriptWindows.js'
import { generateLinuxScript } from '../utils/userDataScriptLinux.js'

function getContainerVar(containerVars, name, defaultVal = '') {
  const found = containerVars.find(v => v.name === name)
  return found && found.value ? found.value : defaultVal
}

/**
 * Generate the container initialization script string from form data.
 * Pure function — reads form, returns script string, no side effects.
 */
/** 启机时由 RunInstances 替换的运行时 ID 槽；生成模板时禁止写入真实值。 */
export const USERDATA_RUNTIME_PLACEHOLDERS = Object.freeze({
  ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
  // 完整任务云前缀（…/cloud），供 boot-progress 与容器 TaskApiEndPoint
  TASK_API_ENDPOINT: '__TASK2APP_TASK_CLOUD_PREFIX__',
  CONTAINER_IMAGE: '__TASK2APP_CONTAINER_IMAGE__',
  CONTAINER_NAME: '__TASK2APP_CONTAINER_NAME__',
  COMMENT_ID: '__TASK2APP_COMMENT_ID__',
  TRACE_ID: '__TASK2APP_TRACE_ID__',
})

export function generateTemplateContent(form) {
  const osType = form.os_type
  if (!osType) return form.content || ''

  const osLabel = getOsTypeLabel(osType)
  const linuxFamily = linuxFamilyForScript(osType) || 'ubuntu'
  const containerVars = form.container_variables || []
  const userdataVars = form.userdata_variables || []
  // 运行时身份/密钥/镜像/链路 ID：一律占位符（忽略表单中可能误填的真实值）
  const ph = USERDATA_RUNTIME_PLACEHOLDERS

  const params = {
    osLabel,
    containerName: ph.CONTAINER_NAME,
    containerPort: getContainerVar(containerVars, 'CONTAINER_PORT', '8080'),
    hostPort: getContainerVar(containerVars, 'HOST_PORT', '8080'),
    volumes: getContainerVar(containerVars, 'VOLUMES', ''),
    containerUser: getContainerVar(containerVars, 'CONTAINER_USER', ''),
    containerWorkdir: getContainerVar(containerVars, 'CONTAINER_WORKDIR', ''),
    containerCommand: getContainerVar(containerVars, 'CONTAINER_COMMAND', ''),
    containerEntryPoint: getContainerVar(containerVars, 'CONTAINER_ENTRYPOINT', ''),
    runtime: getContainerVar(containerVars, 'RUNTIME', 'docker'),
    privileged: getContainerVar(containerVars, 'PRIVILEGED', 'false').toLowerCase() === 'true',
    restartPolicy: getContainerVar(containerVars, 'RESTART_POLICY', 'unless-stopped'),
    businessHostPort: getContainerVar(containerVars, 'BUSINESS_HOST_PORT', '8765'),
    businessContainerPort: getContainerVar(containerVars, 'BUSINESS_CONTAINER_PORT', '8765'),
    businessApiEndpointExplicit: getContainerVar(containerVars, 'BUSINESS_API_ENDPOINT', ''),
    taskApiEndpoint: ph.TASK_API_ENDPOINT,
    accessToken: ph.ACCESS_TOKEN,
    containerImage: ph.CONTAINER_IMAGE,
    commentId: ph.COMMENT_ID,
    traceId: ph.TRACE_ID,
    userdataVars,
    linuxFamily,
    form,
  }

  if (isWindowsOs(osType)) {
    return generateWindowsScript(params)
  }
  return generateLinuxScript(params)
}

/**
 * Generate script AND update form's required variable lists.
 * Used by the "生成容器脚本" button in the modal.
 * Returns the generated script string.
 */
export function generateContainerUserDataScript(form) {
  const script = generateTemplateContent(form)

  const requiredUserDataVars = [
    { name: 'CONTAINER_IMAGE' },
    { name: 'TASK_API_ENDPOINT' },
    { name: 'ACCESS_TOKEN' },
    { name: 'COMMENT_ID' },
    { name: 'TRACE_ID' },
    { name: 'CONTAINER_NAME' },
  ]

  requiredUserDataVars.forEach(reqVar => {
    if (!form.userdata_variables.find(v => v.name === reqVar.name)) {
      form.userdata_variables.push(reqVar)
    }
  })

  const requiredContainerVars = [
    { name: 'CONTAINER_NAME' },
    { name: 'CONTAINER_PORT' },
    { name: 'HOST_PORT' },
    { name: 'BUSINESS_HOST_PORT' },
    { name: 'BUSINESS_API_ENDPOINT' },
    { name: 'BUSINESS_CONTAINER_PORT' },
    { name: 'VOLUMES' },
    { name: 'CONTAINER_USER' },
    { name: 'CONTAINER_WORKDIR' },
    { name: 'CONTAINER_COMMAND' },
    { name: 'CONTAINER_ENTRYPOINT' },
    { name: 'RUNTIME' },
    { name: 'PRIVILEGED' },
    { name: 'NETWORK_MODE' },
    { name: 'RESTART_POLICY' }
  ]

  requiredContainerVars.forEach(reqVar => {
    if (!form.container_variables.find(v => v.name === reqVar.name)) {
      form.container_variables.push(reqVar)
    }
  })

  return script
}
