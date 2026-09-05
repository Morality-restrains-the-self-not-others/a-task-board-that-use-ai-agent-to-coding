import { apiFetch } from './apiUtils.js'
import {
  buildAutoRunStartVmRequest,
  projectHasConfiguredRunTemplate,
} from './projectRunTemplateUtils.js'

/** 创建任务成功后，按项目运行模版触发云服务器启动（失败不阻断主流程） */
export async function triggerTaskAutoRun({
  tenantId,
  workspaceId,
  taskId,
  containerImageId,
  projects,
  projectId,
}) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const tk = String(taskId || '').trim()
  const imageId = String(containerImageId || '').trim()
  const pid = String(projectId || '').trim()
  if (!tid || !wid || !tk || !imageId || !pid) return

  const project = (Array.isArray(projects) ? projects : []).find(
    (item) => String(item?.id) === pid,
  )
  if (!project || !projectHasConfiguredRunTemplate(project)) return

  const requestPayload = buildAutoRunStartVmRequest({
    taskId: tk,
    containerImageId: imageId,
    runTemplate: project.server_run_template,
  })
  if (!requestPayload) return

  try {
    await apiFetch(
      `/api/cloud/compute/${requestPayload.apiPath}/tenant_id/${tid}/workspace_id/${wid}`,
      {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(requestPayload.body),
      },
    )
  } catch {
    // 自动运行失败由任务详情页服务器状态展示；此处不阻断创建流程
  }
}
