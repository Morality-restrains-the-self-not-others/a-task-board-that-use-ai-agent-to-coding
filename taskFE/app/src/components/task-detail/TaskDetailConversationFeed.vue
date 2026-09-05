<template>
  <div
    ref="rootRef"
    class="conversation-feed space-y-3"
    data-testid="task-detail-conversation-feed"
  >
    <template v-if="comments.length > 0">
      <div
        v-for="c in comments"
        :key="`${c.commentKind}-${c.id}`"
        class="space-y-2"
      >
        <div
          class="p-3 rounded-xl shadow-sm border transition-colors"
          :class="bubbleClass(c)"
          :data-comment-id="commentDomId(c)"
          data-testid="comment-bubble"
        >
          <div class="flex-1 min-w-0">
            <div
              class="flex flex-wrap items-center gap-x-2 gap-y-1"
              data-testid="comment-author-row"
            >
              <span class="inline-flex items-center gap-2 min-w-0">
                <img
                  class="w-9 h-9 rounded-full shrink-0"
                  data-testid="comment-author-avatar"
                  :src="authorAvatar(c)"
                  :alt="authorLabel(c)"
                >
                <span
                  class="text-sm font-medium text-gray-900 truncate"
                  data-testid="comment-author-name"
                >{{ authorLabel(c) }}</span>
              </span>
              <span
                v-if="c.commentKind === 'ai'"
                class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-violet-200/90 text-violet-900"
              >发送给 AI</span>
              <span
                v-else-if="c.commentKind === 'container_agent'"
                class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-violet-200/90 text-violet-900"
              >容器 Agent</span>
              <span class="text-[11px] text-gray-500">{{ formatDt(c.created_at) }}</span>
              <span
                v-if="startSkipNoticeOf(c)"
                class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-amber-100 text-amber-950 whitespace-normal break-words max-w-full"
                data-testid="comment-start-skip-notice"
                :title="startSkipNoticeOf(c)"
              >{{ startSkipNoticeOf(c) }}</span>
            </div>
            <p
              v-if="commentDisplayBody(c) && !gitPrHtmlUrlOf(c)"
              class="mt-1 text-sm text-gray-800 whitespace-pre-wrap break-words"
              v-html="renderMentionChips(commentDisplayBody(c))"
            />
            <CommentGitPrReply
              v-if="gitPrHtmlUrlOf(c)"
              :comment="c"
              :tenant-id="tenantId"
              :task-id="taskId"
              :status="statusByUrl[gitPrHtmlUrlOf(c)]"
              :oauth-readiness="oauthReadiness"
              @merged="onGitPrMerged"
            />
            <div
              v-if="(c.commentKind === 'ai' || c.commentKind === 'container_agent') && c.assistant_response"
              class="mt-2 text-sm text-gray-700 border-l-3 border-violet-400 pl-3"
            >
              <span class="text-xs font-medium text-violet-800 block mb-1">{{
                c.commentKind === 'container_agent' ? 'Agent 回复' : 'AI 回复'
              }}</span>
              <SafeMarkdownBlock
                v-if="assistantLooksLikeMarkdown(c.assistant_response)"
                class="!max-h-none max-h-96"
                :source="c.assistant_response"
              />
              <pre
                v-else
                class="whitespace-pre-wrap break-words text-[13px] leading-relaxed"
              >{{ c.assistant_response }}</pre>
            </div>
            <div
              v-if="showStreamUnder(c)"
              class="mt-2 text-sm text-gray-700 border-l-3 border-violet-400 pl-3"
              data-testid="ai-stream-output-nested"
            >
              <span v-if="streamBusy" class="text-xs text-violet-800 block mb-1">流式生成中…</span>
              <SafeMarkdownBlock
                v-if="streamText && assistantLooksLikeMarkdown(streamText)"
                class="!max-h-none max-h-80"
                :source="streamText"
              />
              <pre
                v-else-if="streamText"
                class="whitespace-pre-wrap break-words text-[13px] leading-relaxed max-h-80 overflow-y-auto"
              >{{ streamText }}</pre>
            </div>
            <slot
              v-if="c.commentKind !== 'container_agent' && hasCommentId(c)"
              name="execution-details"
              :comment="c"
              :is-active="isActiveExecutionComment(c)"
              :execution-mode="executionModeFor(c)"
            />
          </div>
        </div>

        <div
          v-if="Array.isArray(c.children) && c.children.length"
          class="ml-6 pl-3 border-l-2 border-violet-200 space-y-2"
          data-testid="comment-children"
        >
          <div
            v-for="child in c.children"
            :key="`${child.commentKind}-${child.id}`"
            class="p-3 rounded-xl shadow-sm border transition-colors"
            :class="bubbleClass(child)"
            :data-comment-id="commentDomId(child)"
            data-testid="comment-bubble"
          >
            <div class="flex-1 min-w-0">
              <div
                class="flex flex-wrap items-center gap-x-2 gap-y-1"
                data-testid="comment-author-row"
              >
                <span class="inline-flex items-center gap-2 min-w-0">
                  <img
                    class="w-9 h-9 rounded-full shrink-0"
                    data-testid="comment-author-avatar"
                    :src="authorAvatar(child)"
                    :alt="authorLabel(child)"
                  >
                  <span
                    class="text-sm font-medium text-gray-900 truncate"
                    data-testid="comment-author-name"
                  >{{ authorLabel(child) }}</span>
                </span>
                <span
                  v-if="child.commentKind === 'container_agent'"
                  class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-violet-200/90 text-violet-900"
                >容器 Agent</span>
                <span
                  v-else-if="gitPrHtmlUrlOf(child)"
                  class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-sky-200/90 text-sky-900"
                >PR</span>
                <span class="text-[11px] text-gray-500">{{ formatDt(child.created_at) }}</span>
                <span
                  v-if="startSkipNoticeOf(child)"
                  class="text-[10px] font-medium px-2 py-0.5 rounded-full bg-amber-100 text-amber-950 whitespace-normal break-words max-w-full"
                  data-testid="comment-start-skip-notice"
                  :title="startSkipNoticeOf(child)"
                >{{ startSkipNoticeOf(child) }}</span>
              </div>
              <p
                v-if="commentDisplayBody(child) && !gitPrHtmlUrlOf(child)"
                class="mt-1 text-sm text-gray-800 whitespace-pre-wrap break-words"
                v-html="renderMentionChips(commentDisplayBody(child))"
              />
              <CommentGitPrReply
                v-if="gitPrHtmlUrlOf(child)"
                :comment="child"
                :tenant-id="tenantId"
                :task-id="taskId"
                :status="statusByUrl[gitPrHtmlUrlOf(child)]"
                :oauth-readiness="oauthReadiness"
                @merged="onGitPrMerged"
              />
              <div
                v-if="child.assistant_response"
                class="mt-2 text-sm text-gray-700 border-l-3 border-violet-400 pl-3"
              >
                <span class="text-xs font-medium text-violet-800 block mb-1">Agent 回复</span>
                <SafeMarkdownBlock
                  v-if="assistantLooksLikeMarkdown(child.assistant_response)"
                  class="!max-h-none max-h-96"
                  :source="child.assistant_response"
                />
                <pre
                  v-else
                  class="whitespace-pre-wrap break-words text-[13px] leading-relaxed"
                >{{ child.assistant_response }}</pre>
              </div>
              <div
                v-if="showStreamUnder(child)"
                class="mt-2 text-sm text-gray-700 border-l-3 border-violet-400 pl-3"
                data-testid="ai-stream-output-nested"
              >
                <span v-if="streamBusy" class="text-xs text-violet-800 block mb-1">流式生成中…</span>
                <SafeMarkdownBlock
                  v-if="streamText && assistantLooksLikeMarkdown(streamText)"
                  class="!max-h-none max-h-80"
                  :source="streamText"
                />
                <pre
                  v-else-if="streamText"
                  class="whitespace-pre-wrap break-words text-[13px] leading-relaxed max-h-80 overflow-y-auto"
                >{{ streamText }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
    <p v-else class="text-center text-gray-500 py-3 text-sm">暂无评论</p>

    <div
      v-if="showFallbackStream"
      class="p-3 rounded-xl border border-violet-200 bg-violet-50/80"
      data-testid="ai-stream-output"
    >
      <div class="flex-1 min-w-0">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1 mb-1">
          <div class="w-9 h-9 rounded-full shrink-0 bg-violet-200 flex items-center justify-center text-xs font-bold text-violet-900">
            AI
          </div>
          <span v-if="streamBusy" class="text-xs text-violet-800">流式生成中…</span>
        </div>
        <SafeMarkdownBlock
          v-if="streamText && assistantLooksLikeMarkdown(streamText)"
          class="!max-h-none max-h-80"
          :source="streamText"
        />
        <pre
          v-else-if="streamText"
          class="text-sm text-gray-800 whitespace-pre-wrap break-words max-h-80 overflow-y-auto"
        >{{ streamText }}</pre>
      </div>
    </div>

    <details
      v-if="hasAgentSteps && agentStepCount > 0"
      class="rounded-lg border border-gray-200 bg-gray-50/90 px-3 py-2 text-xs text-gray-600"
    >
      <summary class="cursor-pointer font-medium text-gray-700 select-none">
        当前任务代理步骤（{{ agentStepCount }}）— 完整卡片见评论「执行细节」
      </summary>
      <p class="mt-2 text-[11px] text-gray-500">
        在对应评论的「执行细节」→ zTree「任务执行」中可查看模型徽章、工具结果与安全富文本交互。
      </p>
    </details>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import SafeMarkdownBlock from '../secure-rich-text/SafeMarkdownBlock.vue'
import CommentGitPrReply from './CommentGitPrReply.vue'
import { initialsAvatarDataUri } from '../../utils/initialsAvatarDataUri.js'
import {
  resolveCommentAuthorAvatar,
  resolveCommentAuthorDisplayName,
} from '../../utils/taskCardPeopleDisplay.js'
import {
  resolveActiveExecutionCommentId,
  resolveCommentExecutionMode,
} from '../../composables/taskDetail/useCommentExecutionContext.js'
import {
  collectGitPrHtmlUrls,
  fetchGitPrStatuses,
  gitPrHtmlUrlOf,
} from '../../composables/taskDetail/taskDetailGitPrReply.js'
import { splitCommentStartSkipNotice } from '../../utils/commentStartSkipNotice.js'

const props = defineProps({
  comments: { type: Array, default: () => [] },
  collaboratorNameById: { type: Object, default: () => ({}) },
  collaboratorAvatarById: { type: Object, default: () => ({}) },
  streamBusy: { type: Boolean, default: false },
  streamText: { type: String, default: '' },
  activeContainerAgentId: { type: String, default: '' },
  hasAgentSteps: { type: Boolean, default: false },
  agentStepCount: { type: Number, default: 0 },
  /** 可选覆盖；默认由 comments + activeContainerAgentId 推导 */
  activeExecutionCommentId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  oauthReadiness: { type: Object, default: null },
})

const resolvedActiveExecutionCommentId = computed(() => {
  const override = String(props.activeExecutionCommentId || '').trim()
  if (override) return override
  return resolveActiveExecutionCommentId(props.comments, props.activeContainerAgentId)
})

function hasCommentId(c) {
  return Boolean(String(c?.id ?? '').trim())
}

function isActiveExecutionComment(c) {
  const id = String(c?.id ?? '').trim()
  return Boolean(id) && id === resolvedActiveExecutionCommentId.value
}

function executionModeFor(c) {
  return resolveCommentExecutionMode(c)
}

const rootRef = ref(null)
const statusByUrl = ref({})

async function refreshGitPrStatuses() {
  const urls = collectGitPrHtmlUrls(props.comments)
  if (!urls.length || !String(props.tenantId || '').trim()) {
    statusByUrl.value = {}
    return
  }
  try {
    statusByUrl.value = await fetchGitPrStatuses(props.tenantId, urls)
  } catch {
    /* 状态刷新失败时卡片展示「状态未知」，不打断会话 */
  }
}

function onGitPrMerged(htmlUrl, meta) {
  const u = String(htmlUrl || '').trim()
  if (!u) return
  // 合并成功：徽章立即回显「已由 xxx 合并」（审计回显），并触发一次状态轮询
  // 与服务端审计数据（含 noop 场景）对齐
  statusByUrl.value = {
    ...statusByUrl.value,
    [u]: {
      ...(statusByUrl.value[u] || {}),
      state: 'merged',
      ...(meta?.merged_by ? { merged_by: String(meta.merged_by) } : {}),
      ...(meta?.merged_at ? { merged_at: String(meta.merged_at) } : {}),
    },
  }
  void refreshGitPrStatuses()
}

watch(
  () => `${props.tenantId}|${collectGitPrHtmlUrls(props.comments).join('|')}`,
  () => {
    void refreshGitPrStatuses()
  },
  { immediate: true },
)

const nestedStreamHostIds = computed(() => {
  const ids = new Set()
  for (const c of props.comments || []) {
    if (c?.id != null) ids.add(String(c.id))
    for (const child of Array.isArray(c?.children) ? c.children : []) {
      if (child?.id != null) ids.add(String(child.id))
    }
  }
  return ids
})

const showFallbackStream = computed(() => {
  if (!props.streamBusy && !props.streamText) return false
  const aid = String(props.activeContainerAgentId || '').trim()
  if (aid && nestedStreamHostIds.value.has(aid)) return false
  return true
})

function showStreamUnder(c) {
  if (!props.streamBusy && !props.streamText) return false
  const aid = String(props.activeContainerAgentId || '').trim()
  if (!aid) return false
  return String(c?.id || '') === aid
}

function authorLabel(c) {
  const fallback = c?.commentKind === 'container_agent' ? '容器 Agent' : ''
  return resolveCommentAuthorDisplayName(c?.created_by, props.collaboratorNameById, fallback) || fallback || '?'
}

function authorAvatar(c) {
  const real = resolveCommentAuthorAvatar(c?.created_by, props.collaboratorAvatarById)
  if (real) return real
  return initialsAvatarDataUri(authorLabel(c) || '?')
}

function formatDt(datetime) {
  if (!datetime) return ''
  const date = new Date(datetime)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function commentStartSkipParts(c) {
  return splitCommentStartSkipNotice(c?.content)
}

function commentDisplayBody(c) {
  return commentStartSkipParts(c).body
}

function startSkipNoticeOf(c) {
  return commentStartSkipParts(c).skipNotice
}

/**
 * 将评论正文中的 $镜像名 提及渲染为高亮 chip 样式。
 * 匹配 $ 后跟字母、数字、下划线、点或连字符的模式。
 * 使用 v-html 输出，内容经 escapeHtml 防 XSS。
 */
function renderMentionChips(text) {
  const raw = String(text || '')
  if (!raw) return ''
  const escaped = escapeHtml(raw)
  return escaped.replace(
    /\$([\w.\-]+)/g,
    '<span class="mention-chip" style="display:inline-block;padding:0 6px;border-radius:9999px;background:#dbeafe;color:#1e40af;font-size:inherit;line-height:1.6;white-space:nowrap;">$1</span>',
  )
}

function escapeHtml(str) {
  const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' }
  return String(str).replace(/[&<>"']/g, (ch) => map[ch])
}

function bubbleClass(c) {
  return c.commentKind === 'ai' || c.commentKind === 'container_agent'
    ? 'bg-violet-50/90 border-violet-100'
    : 'bg-white border-gray-100'
}

/** 评论 DOM 定位 id（OPT-20260817-013）：空串返回 undefined 省略属性，供 ?comment= 滚动定位。 */
function commentDomId(c) {
  const id = String(c?.id ?? '').trim()
  return id || undefined
}

function assistantLooksLikeMarkdown(t) {
  const s = String(t || '')
  if (/^#{1,6}\s/m.test(s)) return true
  if (/```[\s\S]/.test(s)) return true
  if (/^\s*[-*]\s/m.test(s)) return true
  if (/\[.+\]\(https?:\/\//.test(s)) return true
  return false
}

function scrollToBottom() {
  const el = rootRef.value
  if (!el) return
  let container = el.parentElement
  while (container) {
    const style = window.getComputedStyle(container)
    const canScrollY =
      /(auto|scroll)/.test(style.overflowY) && container.scrollHeight > container.clientHeight
    if (canScrollY) {
      container.scrollTo({ top: container.scrollHeight, behavior: 'smooth' })
      return
    }
    container = container.parentElement
  }
}

watch(
  [
    () => props.streamText,
    () => props.streamBusy,
    () => props.comments.length,
  ],
  ([nextStreamText, nextStreamBusy, nextCommentsLength], [prevStreamText, prevStreamBusy, prevCommentsLength]) => {
    const streamStateChanged =
      nextStreamText !== prevStreamText || nextStreamBusy !== prevStreamBusy
    const commentsAppended = Number(nextCommentsLength || 0) > Number(prevCommentsLength || 0)
    if (!streamStateChanged && !commentsAppended) return
    nextTick(() => scrollToBottom())
  }
)
</script>

<style scoped>
.border-l-3 {
  border-left-width: 3px;
}

/* OPT-20260817-013: 搜索命中评论定位后的高亮（scrollToCommentById 追加该 class） */
.comment-highlight {
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.35);
  border-color: #8b5cf6 !important;
}
</style>
