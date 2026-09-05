<template>
  <template v-if="containerPageUrl && !pendingReveal && !httpUnreachable">
    <a
      id="open-container-page-btn"
      :href="containerPageUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-primary border border-primary/40 rounded-md hover:bg-primary/5 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50"
      :aria-disabled="pageBusy ? 'true' : 'false'"
      @click="onOpenContainerPageClick"
    >
      <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
      </svg>
      {{ pageBusy ? '准备中…' : '打开容器页面' }}
    </a>
    <a
      v-if="displayContainerVscodeUrl"
      id="open-container-vscode-btn"
      :href="displayContainerVscodeUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-700 border border-slate-300 rounded-md hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-slate-400 disabled:opacity-50"
      :aria-disabled="vscodeBusy ? 'true' : 'false'"
      @click="onOpenContainerVscodeClick"
    >
      <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
      {{ vscodeBusy ? '准备中…' : '打开容器开发页面' }}
    </a>
  </template>
</template>

<script setup>
import { ref } from 'vue'
import { openContainerPageWithIngressEnsure } from '../../utils/openContainerPage.js'

/**
 * 任务关联区的「打开容器页面 / 打开容器开发页面」共享链接组件（OPT-20260810-034）。
 * 原 TaskDetailTaskLayerAssociationPanel 与 TaskDetailCommentLayerZtreeStatus
 * 各写一份含 ingress ensure 点击逻辑与样式类，这里统一为一处，避免双份维护导致
 * loading 态与就绪态按钮行为漂移。保留 #open-container-page-btn / #open-container-vscode-btn id。
 */
const props = defineProps({
  containerPageUrl: { type: String, default: '' },
  displayContainerVscodeUrl: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  commentId: { type: String, default: '' },
  pendingReveal: { type: Boolean, default: false },
  httpUnreachable: { type: Boolean, default: false },
})

const pageBusy = ref(false)
const vscodeBusy = ref(false)

async function onOpenContainerPageClick(event) {
  event?.preventDefault?.()
  if (pageBusy.value) return
  pageBusy.value = true
  try {
    await openContainerPageWithIngressEnsure({
      containerPageUrl: props.containerPageUrl,
      tenantId: props.tenantId,
      workspaceId: props.workspaceId,
      taskId: props.taskId,
      commentId: props.commentId,
    })
  } finally {
    pageBusy.value = false
  }
}

async function onOpenContainerVscodeClick(event) {
  event?.preventDefault?.()
  if (vscodeBusy.value) return
  vscodeBusy.value = true
  try {
    await openContainerPageWithIngressEnsure({
      containerPageUrl: props.displayContainerVscodeUrl,
      tenantId: props.tenantId,
      workspaceId: props.workspaceId,
      taskId: props.taskId,
      commentId: props.commentId,
    })
  } finally {
    vscodeBusy.value = false
  }
}
</script>
