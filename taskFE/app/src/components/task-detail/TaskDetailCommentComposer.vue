<template>
  <div
    class="task-detail-comment-composer"
    :data-testid="rootTestId || undefined"
  >
    <CommentExecutionDependencyPicker
      v-if="showDependencyPicker"
      v-model:execution-mode="executionMode"
      v-model:depends-on-comment-ids="dependsOnCommentIds"
      v-model:auto-commit="autoCommit"
      class="mb-2"
      :predecessor-options="predecessorOptions"
      :show-queue-toggle="showQueueToggle"
      :task="task"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      @task-updated="emit('task-updated', $event)"
    />
    <div :data-testid="inputTestId || undefined">
      <CommentImageMentionEditor
        v-if="atModeEnabled"
        v-model="newComment"
        :installed-images="mentionImageOptions"
        :placeholder="atModePlaceholder"
        @keydown="onKeydown"
        @mention-change="onMentionChange"
      />
      <textarea
        v-else
        :id="nativeTextareaId"
        v-model="newComment"
        class="w-full text-sm p-2 border border-gray-300 rounded-md resize-y min-h-[4rem] focus:outline-none focus:ring-primary focus:border-primary"
        rows="3"
        :placeholder="plainPlaceholder"
        @keydown="onKeydown"
      />
    </div>
    <!-- $镜像 后：镜像说明在左、智能体资源配置在右（composer 就地挂载，非 Teleport） -->
    <div
      v-show="showRunConfig"
      class="mt-2 flex items-start gap-4"
      data-testid="comment-composer-run-config-row"
    >
      <div
        class="min-w-0 flex-1"
        data-testid="comment-composer-image-select"
      >
        <template v-if="showImageSelect && showRunConfig">
          <p
            class="text-xs text-gray-600 mb-2"
            data-testid="comment-composer-mentioned-image"
          >
            将使用镜像 {{ pendingMentionDisplayLabel }} 运行
          </p>
          <ServerConfigImageSectionHints
            v-if="imageOptions.length"
            :installed-images="imageOptions"
            :selected-image-id="composerSelectedImageId"
            :task="imageHintsTask"
            :runtime-status="null"
            :viewer-task-id="imageHintsTaskId"
            :tenant-id="tenantId"
            :workspace-id="workspaceId"
          />
        </template>
      </div>
      <div
        data-testid="comment-composer-feature-params-slot"
        class="min-w-0 flex-1 [&>*]:mt-0"
      >
        <ServerConfigFeatureParamsBlock
          v-if="showImageSelect && featureParamsView"
          :tenant-id="featureParamsView.tenantId"
          :feature-params-source="featureParamsView.featureParamsSource"
          :personal-configs="featureParamsView.personalConfigs"
          :selected-personal-config-id="featureParamsView.selectedPersonalConfigId"
          :resolved-env-preview="featureParamsView.resolvedEnvPreview"
          :is-env-preview-loading="featureParamsView.isEnvPreviewLoading"
          :env-preview-expanded="featureParamsView.envPreviewExpanded"
          :persist-error="featureParamsView.persistError"
          :sources-available="featureParamsView.sourcesAvailable"
          @source-change="featureParamsView.onSourceChange"
          @preview-env="featureParamsView.fetchEnvPreview"
          @update:feature-params-source="featureParamsView.onFeatureParamsSourceUpdate"
          @update:selected-personal-config-id="featureParamsView.onPersonalConfigIdUpdate"
          @update:env-preview-expanded="featureParamsView.setEnvPreviewExpanded"
        />
      </div>
    </div>
    <!-- 硬件在智能体资源配置下方：先选环境变量，再调本次运行硬件 -->
    <CommentComposerRepoIdentity
      v-if="showRunConfig"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      :task-id="viewerTaskId"
      :task-projects-with-details="taskProjectsWithDetails"
      :comments="comments"
      :task-repo-identities="taskRepoIdentities"
      @repo-oauth-readiness="onRepoOAuthReadiness"
    />
    <CommentComposerHardwareCard
      v-if="showImageSelect && showRunConfig"
      v-model:selected-image-id="composerSelectedImageId"
      :project-server-run-template="projectServerRunTemplate"
      :installed-images="imageOptions"
      :task="task"
      :workspace-id="workspaceId"
    />
    <div class="mt-2 flex justify-end flex-wrap gap-2">
      <p
        v-if="oauthBlockedReason"
        class="w-full text-[11px] text-amber-800 leading-snug m-0"
        data-testid="comment-composer-oauth-blocked-reason"
        role="alert"
      >
        {{ oauthBlockedReason }}
      </p>
      <button
        :id="submitButtonId"
        type="button"
        class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
        data-testid="task-detail-comment-submit"
        :disabled="Boolean(oauthBlockedReason) || submitBusy"
        :aria-busy="submitBusy ? 'true' : 'false'"
        @click.stop="onSubmitClick"
      >
        {{ submitBusy ? '提交中…' : (pendingMentionId ? '提交并运行' : '提交评论') }}
      </button>
      <button
        v-if="showCancel"
        type="button"
        class="text-xs text-gray-500 hover:text-gray-700 px-2 py-2"
        data-testid="task-detail-comment-cancel"
        @click.stop="emit('cancel')"
      >
        取消
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import CommentImageMentionEditor from './CommentImageMentionEditor.vue'
import CommentExecutionDependencyPicker from './CommentExecutionDependencyPicker.vue'
import CommentComposerHardwareCard from './CommentComposerHardwareCard.vue'
import CommentComposerRepoIdentity from './CommentComposerRepoIdentity.vue'
import ServerConfigFeatureParamsBlock from '../ServerConfigFeatureParamsBlock.vue'
import ServerConfigImageSectionHints from '../ServerConfigImageSectionHints.vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { createClickGuard } from '../../utils/clickGuard.js'
import { collectLinkedRepoUrls, resolveCommentRunOauthBlockedReason } from '../../utils/commentRepoIdentity.js'
import { rememberGrantTicketFromSearch } from '../../utils/grantTicketSession.js'
import { setPendingImageMention } from '../../composables/taskDetail/commentImageMentionState.js'
import { bridgedSelectedImageId } from '../../composables/taskDetail/taskDetailImageSelectionBridge.js'
import { useCommentComposerFeatureParamsView } from '../../composables/taskDetail/taskDetailFeatureParamsBridge.js'
import { formatInstalledImageRunLabel } from '../../utils/installedImageLabel.js'
import {
  readCommentDependencyDraft,
  setCommentDependencyDraft,
} from '../../composables/taskDetail/commentDependencyDraft.js'

if (typeof window !== 'undefined') {
  rememberGrantTicketFromSearch(window.location?.search || '')
}

const newComment = defineModel({ type: String, required: true })
const executionMode = defineModel('executionMode', { type: String, default: 'wait_previous' })
const dependsOnCommentIds = defineModel('dependsOnCommentIds', { type: Array, default: () => [] })
const autoCommit = defineModel('autoCommit', { type: Boolean, default: false })

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  /** 当前任务对象（镜像归属提示的架构兜底等） */
  task: { type: Object, default: null },
  /** 当前查看的任务 id（机器节点归属提示标记「本任务」） */
  viewerTaskId: { type: String, default: '' },
  showCancel: { type: Boolean, default: false },
  /** 为 false 时不写全局 id，避免看板多卡重复 id */
  useLegacyDomIds: { type: Boolean, default: true },
  /** 根节点 data-testid；看板卡片可传 task-card-comment-composer */
  rootTestId: { type: String, default: '' },
  /** 输入区域 data-testid；看板卡片可传 task-card-comment-input */
  inputTestId: { type: String, default: '' },
  showDependencyPicker: { type: Boolean, default: true },
  predecessorOptions: { type: Array, default: () => [] },
  /** 是否在评论区展示镜像选择（与 $镜像 同步） */
  showImageSelect: { type: Boolean, default: true },
  projectServerRunTemplate: { type: Object, default: null },
  taskProjectsWithDetails: { type: Array, default: () => [] },
  /** 当前任务评论列表；最近一条 repo_identities 预填本次运行身份。 */
  comments: { type: Array, default: () => [] },
  /** 任务级 repo_identities 回退预填。 */
  taskRepoIdentities: { type: Array, default: () => [] },
  atModePlaceholder: {
    type: String,
    default: '写下你的评论… 输入 $ 选择镜像后提交并运行',
  },
  plainPlaceholder: {
    type: String,
    default: '写下你的评论… Ctrl+Enter 提交评论',
  },
})

const emit = defineEmits(['submit', 'cancel', 'keydown', 'mention-change', 'task-updated'])

const showQueueToggle = computed(() => Boolean(
  props.showDependencyPicker
    && props.task?.id
    && String(props.tenantId || '').trim()
    && String(props.workspaceId || '').trim(),
))

const featureParamsView = useCommentComposerFeatureParamsView()

const atModeEnabled = ref(true) // 默认开启容器镜像 @ 模式
const localInstalledImages = ref([])
const pendingMentionId = ref('')
const pendingMentionLabel = ref('')
const repoOAuthReadiness = ref({
  hasOAuthRepos: false,
  loading: false,
  allBound: true,
  checkError: '',
})
const submitBusy = ref(false)
const submitGuard = createClickGuard()
// $镜像 后展示运行配置（身份/硬件/feature params）；showImageSelect 仅控制镜像确认块。
const showRunConfig = computed(() => Boolean(pendingMentionId.value))
const oauthBlockedReason = computed(() => {
  if (!pendingMentionId.value) return ''
  return resolveCommentRunOauthBlockedReason(
    repoOAuthReadiness.value,
    collectLinkedRepoUrls(props.taskProjectsWithDetails),
  )
})

const onRepoOAuthReadiness = (readiness) => {
  repoOAuthReadiness.value = {
    hasOAuthRepos: Boolean(readiness?.hasOAuthRepos),
    loading: Boolean(readiness?.loading),
    allBound: Boolean(readiness?.allBound),
    checkError: String(readiness?.checkError || '').trim(),
  }
}

const onSubmitClick = () => {
  if (oauthBlockedReason.value || submitBusy.value) return
  void submitGuard.run(async () => {
    submitBusy.value = true
    try {
      emit('submit')
    } finally {
      submitBusy.value = false
    }
  })
}

const nativeTextareaId = computed(() => (props.useLegacyDomIds ? 'comment-content' : undefined))
const submitButtonId = computed(() => (props.useLegacyDomIds ? 'comment-submit-btn' : undefined))

const imageOptions = computed(() => localInstalledImages.value)

const pendingMentionDisplayLabel = computed(() => {
  const id = String(pendingMentionId.value || '')
  const img = imageOptions.value.find((x) => String(x.id) === id)
  const version = img?.version ?? img?.tag ?? ''
  return formatInstalledImageRunLabel(pendingMentionLabel.value, version)
})

const mentionImageOptions = computed(() =>
  imageOptions.value
    .map((x) => ({
      // 展开保留 image_skills / image_skills_extract_status：评论区 $镜像 后输入 / 需按
      // 镜像技能列表弹下拉（剥离后 mentionedSkills 恒空、技能菜单永不打开）
      ...x,
      id: String(x.id || ''),
      name: String(x.name || x.image_name || x.id || ''),
      version: x.version ?? x.tag,
    }))
    .filter((x) => x.id),
)

/** 与 ServerConfig.selectedImageId 经 bridge 同步（兄弟组件无法 inject） */
const composerSelectedImageId = computed({
  get() {
    return String(bridgedSelectedImageId.value || '')
  },
  set(v) {
    bridgedSelectedImageId.value = v != null ? String(v) : ''
  },
})

const imageHintsTask = computed(() => props.task || null)
const imageHintsTaskId = computed(() => String(props.viewerTaskId || ''))

async function loadAtModeContext() {
  const tenantId = props.tenantId != null ? String(props.tenantId).trim() : ''
  const workspaceId = props.workspaceId != null ? String(props.workspaceId).trim() : ''
  if (!tenantId || !workspaceId) return
  try {
    const wsResp = await apiFetch(
      `/api/projects/workspaces/tenant_id/${encodeURIComponent(tenantId)}/${encodeURIComponent(workspaceId)}/`,
      {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      },
    )
    if (wsResp.ok) {
      const ws = await wsResp.json()
      // 默认开启：当后端未返回该字段或值为 null/undefined 时，默认 true
      atModeEnabled.value = ws.container_image_at_mode_enabled != null ? !!ws.container_image_at_mode_enabled : true
    }
  } catch (e) {
    console.warn('load at-mode failed', e)
  }
  if (!atModeEnabled.value) {
    localInstalledImages.value = []
    pendingMentionId.value = ''
    setPendingImageMention(null)
    return
  }
  try {
    const imgResp = await apiFetch(
      `/api/cloud/installed-images/tenant_id/${encodeURIComponent(tenantId)}`,
      {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      },
    )
    if (!imgResp.ok) return
    const data = await imgResp.json()
    const list = Array.isArray(data) ? data : (data?.installed_images || data?.results || [])
    localInstalledImages.value = list
      .map((x) => ({
        // 展开保留 image_skills 等原字段（评论技能下拉依赖，见 mentionImageOptions 注释）
        ...x,
        id: String(x.id || ''),
        name: String(x.name || x.image_name || x.id || ''),
        version: x.version ?? x.tag,
        hardware_summary: x.hardware_summary,
        target_architectures: x.target_architectures,
      }))
      .filter((x) => x.id)
  } catch (e) {
    console.warn('load installed images failed', e)
  }
}

function onMentionChange(mention) {
  if (mention && mention.id) {
    pendingMentionId.value = String(mention.id)
    pendingMentionLabel.value = String(mention.name || mention.id)
    composerSelectedImageId.value = String(mention.id)
  } else {
    pendingMentionId.value = ''
    pendingMentionLabel.value = ''
  }
  emit('mention-change', mention)
}

function onKeydown(e) {
  emit('keydown', e)
}

watch(
  () => [props.tenantId, props.workspaceId],
  () => {
    void loadAtModeContext()
  },
)

watch(
  [executionMode, dependsOnCommentIds, autoCommit],
  () => {
    if (!props.showDependencyPicker) return
    setCommentDependencyDraft({
      executionMode: executionMode.value,
      dependsOnCommentIds: dependsOnCommentIds.value,
      autoCommit: autoCommit.value,
    })
  },
  { deep: true, immediate: true },
)

onMounted(() => {
  const draft = readCommentDependencyDraft()
  executionMode.value = draft.executionMode
  dependsOnCommentIds.value = [...draft.dependsOnCommentIds]
  autoCommit.value = !!draft.autoCommit
  void loadAtModeContext()
})

defineExpose({
  pendingMentionId,
  atModeEnabled,
  onMentionChange,
  composerSelectedImageId,
})
</script>
