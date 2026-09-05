<template>
  <div class="min-h-[12rem] rounded border border-gray-200 bg-white p-2">
    <p class="text-xs font-medium text-gray-700 mb-2 flex flex-wrap items-baseline gap-x-1.5 gap-y-0.5">
      <span>{{ previewTitle }}</span>
      <span
        v-if="previewKind === 'git' && isRepoRoot && currentBranch"
        class="font-normal text-gray-500"
        data-testid="git-log-current-branch"
      >· {{ currentBranch }}</span>
    </p>
    <p v-if="!selectedPath" class="text-xs text-gray-400">
      {{ previewKind === 'git' ? '点击左侧仓库目录可在此查看最近提交' : '点击左侧文件可在此查看内容' }}
    </p>
    <p v-else-if="loading" class="text-xs text-gray-500">加载中…</p>
    <p
      v-else-if="error"
      class="text-xs text-red-600 whitespace-pre-wrap break-words"
      data-testid="layer-change-preview-error"
      v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
    >{{ error }}</p>
    <template v-else-if="previewKind === 'git'">
      <p class="text-[11px] text-gray-500 mb-2 break-all">{{ selectedPath }}</p>
      <pre
        class="max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs leading-relaxed text-gray-800 font-mono"
      >{{ gitLogText || '' }}</pre>
    </template>
    <template v-else-if="isBinaryFilePreview">
      <dl
        class="space-y-1.5 text-xs text-gray-700"
        data-testid="layer-change-preview-binary-props"
      >
        <div class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">路径</dt>
          <dd class="min-w-0 break-all font-mono text-[11px]">{{ selectedPath }}</dd>
        </div>
        <div v-if="binaryBasename" class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">文件名</dt>
          <dd class="min-w-0 break-all">{{ binaryBasename }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">类型</dt>
          <dd>二进制（不展示内容）</dd>
        </div>
        <div v-if="binarySizeLabel" class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">大小</dt>
          <dd>{{ binarySizeLabel }}</dd>
        </div>
        <div v-if="binaryMtimeLabel" class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">修改时间</dt>
          <dd>{{ binaryMtimeLabel }}</dd>
        </div>
        <div v-if="binaryExt" class="flex gap-2">
          <dt class="shrink-0 text-gray-500 w-16">扩展名</dt>
          <dd>{{ binaryExt }}</dd>
        </div>
      </dl>
    </template>
    <template v-else-if="payload">
      <p class="text-[11px] text-gray-500 mb-2 break-all">{{ selectedPath }}</p>
      <pre
        v-if="isTextFilePreview"
        class="max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs leading-relaxed text-gray-800"
        data-testid="layer-change-preview-text"
      >{{ previewText }}</pre>
      <p v-else class="text-xs text-amber-700">
        当前文件为二进制内容，暂不支持文本预览。
      </p>
      <p v-if="payload.truncated" class="mt-2 text-[11px] text-amber-700">内容过长，已截断显示。</p>
    </template>
    <p v-else class="text-xs text-gray-400">暂无可展示内容</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  previewKind: { type: String, default: 'file' },
  selectedPath: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  /** 请求失败时的 traceId；空则不挂 data-traceId */
  errorTraceId: { type: String, default: '' },
  payload: { type: Object, default: null },
  gitLogText: { type: String, default: '' },
  /** 点击目录是否为 Git 仓库根 */
  isRepoRoot: { type: Boolean, default: false },
  /** 仓库根时的当前工作分支名 */
  currentBranch: { type: String, default: '' },
})

/** 容器 /api/layers/.../files/... 常返回 { path, content, truncated }，无 kind；旧 mock 使用 { kind, text } */
const isTextFilePreview = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return false
  if (p.kind === 'binary') return false
  if (p.kind === 'text') return true
  const text = typeof p.text === 'string' ? p.text : typeof p.content === 'string' ? p.content : ''
  if (!text) return false
  if (text.includes('\0')) return false
  return true
})

/**
 * 新 API：`kind=binary`。
 * 旧容器兼容：仍返回 content 字符串但含 NUL 时按二进制处理（不展示乱码正文）。
 */
const isBinaryFilePreview = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return false
  if (p.kind === 'binary') return true
  if (p.kind === 'text') return false
  const text = typeof p.content === 'string' ? p.content : typeof p.text === 'string' ? p.text : ''
  if (text && text.includes('\0')) return true
  return false
})

const previewTitle = computed(() => {
  if (props.previewKind === 'git') return '提交日志'
  if (isBinaryFilePreview.value) return '文件属性'
  return '文件内容预览'
})

const previewText = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return ''
  if (typeof p.text === 'string') return p.text
  if (typeof p.content === 'string') return p.content
  return ''
})

const binaryBasename = computed(() => {
  const p = props.payload
  if (p && typeof p.basename === 'string' && p.basename.trim()) return p.basename.trim()
  const path = String(props.selectedPath || '').replace(/\\/g, '/')
  if (!path) return ''
  const i = path.lastIndexOf('/')
  return i >= 0 ? path.slice(i + 1) : path
})

const binarySizeLabel = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return ''
  if (typeof p.size_human === 'string' && p.size_human.trim()) return p.size_human.trim()
  if (typeof p.size_bytes === 'number' && Number.isFinite(p.size_bytes)) {
    return `${p.size_bytes} B`
  }
  // 旧 API 无 size 字段时，用已读 content 字节数作近似（可能因截断偏小）
  const text = typeof p.content === 'string' ? p.content : typeof p.text === 'string' ? p.text : ''
  if (text && text.includes('\0')) {
    const approx = new TextEncoder().encode(text).length
    return `约 ${approx} B（容器未返回精确大小）`
  }
  return ''
})

const binaryMtimeLabel = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return ''
  if (typeof p.mtime_iso === 'string' && p.mtime_iso.trim()) {
    const d = new Date(p.mtime_iso)
    if (!Number.isNaN(d.getTime())) return d.toLocaleString()
    return p.mtime_iso
  }
  if (typeof p.mtime_ms === 'number' && Number.isFinite(p.mtime_ms)) {
    return new Date(p.mtime_ms).toLocaleString()
  }
  return ''
})

const binaryExt = computed(() => {
  const p = props.payload
  if (!p || typeof p !== 'object') return ''
  const e = typeof p.ext === 'string' ? p.ext.trim() : ''
  return e || ''
})
</script>
