<template>
  <div class="relative" data-testid="task-description-skill-field">
    <div
      v-if="list.skills.length"
      class="mb-1 flex flex-wrap gap-1"
      data-testid="task-description-skill-chips"
    >
      <button
        v-for="skill in list.skills"
        :key="skill.name"
        type="button"
        class="rounded-md border px-2 py-0.5 text-xs"
        :class="skill.isDefault ? 'border-indigo-300 bg-indigo-50 text-indigo-800' : 'border-gray-200 bg-white text-gray-700'"
        :title="skill.description || skill.name"
        @click="insertSkill(skill.name)"
      >
        <!-- Anti-Replay-OK: ui-only 本地写入描述 mention，无网络 -->
        {{ chipLabel(skill) }}
        <span v-if="skill.isDefault" class="ml-0.5 text-[10px]">默认</span>
      </button>
    </div>
    <div class="relative">
      <ul
        v-if="imageMenuOpen && filteredImages.length"
        class="absolute z-20 left-0 right-0 -top-1 -translate-y-full max-h-48 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg"
        data-testid="task-description-image-mention-picker"
        role="listbox"
      >
        <li
          v-for="(img, idx) in filteredImages"
          :key="img.id"
          role="option"
          :aria-selected="idx === imageHighlightIndex"
          class="px-3 py-1.5 text-sm cursor-pointer"
          :class="idx === imageHighlightIndex ? 'bg-primary/10 text-primary' : 'hover:bg-gray-50'"
          @mousedown.prevent="selectImage(img)"
        >
          <div class="flex items-center gap-2">
            <img
              v-if="imageGroupIconSrc(img)"
              data-testid="task-description-image-mention-icon"
              class="h-6 w-6 shrink-0 rounded object-cover ring-1 ring-gray-200"
              :src="imageGroupIconSrc(img)"
              :alt="String(img.name || img.image_name || '')"
            >
            <span
              v-else
              data-testid="task-description-image-mention-icon-placeholder"
              class="inline-block h-6 w-6 shrink-0 rounded bg-gray-200 ring-1 ring-gray-200"
              aria-hidden="true"
            />
            <div class="min-w-0">
              <div class="leading-tight">{{ formatInstalledImageRunLabel(img.name || img.image_name, img.version ?? img.tag) }}</div>
              <div v-if="imageMetaLine(img)" class="mt-0.5 line-clamp-1 text-xs text-gray-500">
                {{ imageMetaLine(img) }}
              </div>
            </div>
          </div>
        </li>
      </ul>
      <ul
        v-if="skillMenuOpen && filteredSkills.length"
        class="absolute z-20 left-0 right-0 -top-1 -translate-y-full max-h-48 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg"
        data-testid="task-description-skill-picker"
        role="listbox"
      >
        <li
          v-for="(skill, idx) in filteredSkills"
          :key="skill.name"
          role="option"
          :aria-selected="idx === skillHighlightIndex"
          class="px-3 py-1.5 text-sm cursor-pointer"
          :class="idx === skillHighlightIndex ? 'bg-indigo-50 text-indigo-800' : 'hover:bg-gray-50'"
          @mousedown.prevent="selectSkill(skill)"
        >
          /{{ skill.name }}{{ skill.isDefault ? '（默认）' : '' }}
          <span v-if="skill.description" class="ml-2 text-xs text-gray-500">{{ skill.description }}</span>
        </li>
      </ul>
      <div
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 overflow-hidden whitespace-pre-wrap break-words px-3 py-2 text-transparent"
        data-testid="task-description-skill-highlight"
      >
        <span
          v-for="(part, idx) in highlighted"
          :key="idx"
          :class="part.highlight ? 'rounded-sm bg-indigo-200/80 text-indigo-900' : ''"
        >{{ part.text }}</span>
      </div>
      <textarea
        id="task-description"
        ref="areaRef"
        v-model="model"
        rows="3"
        class="relative mt-0 block w-full bg-transparent px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
        placeholder="补充说明（可选）。输入 $ 选择镜像，选择镜像后可输入 /技能名"
        @input="onInput"
        @keydown="onKeydown"
        @blur="onBlur"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import {
  filterInstalledImages,
  findActiveAtQuery,
  replaceAtQueryWithMention,
} from '../composables/taskDetail/commentImageMentionComposer.js'
import {
  filterImageSkills,
  findActiveSlashSkillQuery,
  highlightSkillTokens,
  imageSkillsFromImage,
  insertSkillToken,
} from '../utils/imageSkills.js'
import { formatInstalledImageRunLabel, formatInstalledImageUpdateTime } from '../utils/installedImageLabel.js'
import { imageGroupIconSrc } from '../utils/imageGroupIcon.js'
import { applyImageMentionToDescription, parseImageMentionFromText } from '../utils/createTaskImageMention.js'

const model = defineModel({ type: String, default: '' })

const props = defineProps({
  image: { type: Object, default: null },
  installedImages: { type: Array, default: () => [] },
})

const emit = defineEmits(['mention-change'])

const areaRef = ref(null)
const list = computed(() => imageSkillsFromImage(props.image))
/**
 * chips 提示中的镜像 mention 名：剥离前导 $，
 * 标签渲染为 `$镜像名 /技能名` 完整 mention 语法提示。
 */
const imageMentionName = computed(() => String(props.image?.name || '').trim().replace(/^\$/, ''))

/**
 * chip 提示文本：完整 mention 语法 `$镜像名 /技能名`；
 * 镜像名缺失时回退 `/技能名`。单插值渲染，避免模板空白被编译器合并。
 */
function chipLabel(skill) {
  const name = imageMentionName.value
  return name ? `$${name} /${skill.name}` : `/${skill.name}`
}
const highlighted = computed(() =>
  highlightSkillTokens(model.value, list.value.skills.map((s) => s.name)),
)

// ---- @ 镜像 / 技能 弹层 ----
const imageMenuOpen = ref(false)
const activeQuery = ref(null)
const imageHighlightIndex = ref(0)
const skillMenuOpen = ref(false)
const activeSkillQuery = ref(null)
const skillHighlightIndex = ref(0)

const filteredImages = computed(() =>
  filterInstalledImages(props.installedImages, activeQuery.value?.query || ''),
)

/** 镜像选项元信息行：更新时间 + 说明，无内容时返回 ''（不渲染第二行） */
function imageMetaLine(img) {
  const time = formatInstalledImageUpdateTime(img)
  const desc = String(img?.description || '').trim()
  const parts = []
  if (time) parts.push(`更新时间 ${time}`)
  if (desc) parts.push(desc)
  return parts.join(' · ')
}

/**
 * 当前 mention 对应的镜像：已有绑定（props.image 来自 container_image.id）且名字匹配时
 * 以其为准（同名多变体如 trae-agent 两个版本，@name 文本反解会拿错变体），否则按 id 回退。
 */
const mentionedImage = computed(() => {
  const mention = parseImageMentionFromText(model.value, props.installedImages)
  if (!mention?.id) return null
  const cur = props.image
  if (
    cur &&
    String(cur.name || '').trim().toLowerCase().replace(/^\$/, '') ===
      String(mention.name || '').trim().toLowerCase().replace(/^\$/, '')
  ) {
    return cur
  }
  return props.installedImages.find((img) => String(img.id) === String(mention.id)) || null
})

const mentionedSkills = computed(() => imageSkillsFromImage(mentionedImage.value).skills)

const filteredSkills = computed(() =>
  filterImageSkills(mentionedSkills.value, activeSkillQuery.value?.query || ''),
)

function caretPos() {
  const el = areaRef.value
  if (!el) return String(model.value || '').length
  const raw = el.selectionStart
  // jsdom/异常时 selectionStart 为 0 且文本非空 → 兜底视为末尾（真实浏览器输入后光标即末尾）
  return raw > 0 ? raw : String(model.value || '').length
}

function refreshMenus() {
  const t = model.value
  const caret = caretPos()
  const mention = parseImageMentionFromText(t, props.installedImages)
  if (mention?.id) {
    // 已有镜像：仅允许技能弹层（/query 之后）
    imageMenuOpen.value = false
    activeQuery.value = null
    const sq = mentionedSkills.value.length ? findActiveSlashSkillQuery(t, caret) : null
    activeSkillQuery.value = sq
    skillMenuOpen.value = !!sq
    skillHighlightIndex.value = 0
    return
  }
  skillMenuOpen.value = false
  activeSkillQuery.value = null
  const q = findActiveAtQuery(t, caret)
  activeQuery.value = q
  imageMenuOpen.value = !!q
  imageHighlightIndex.value = 0
}

/** 文本变化 → 弹层刷新 + 通知父级同步 container_image 绑定 */
function onInput(e) {
  refreshMenus()
  // input 事件时 DOM 值即权威新值（defineModel prop 下行滞后，model.value 可能还是旧值）
  emitMentionChange(e?.target?.value)
}

function onBlur() {
  // 延迟关闭，允许 mousedown 选中菜单项
  setTimeout(() => {
    imageMenuOpen.value = false
    skillMenuOpen.value = false
  }, 120)
}

/**
 * 通知父级同步 container_image 绑定。
 * @param {string} [textOverride] 权威新文本——model.value 是 defineModel prop-backed，
 *   写入后需父组件重渲染 prop 才下行，同同步块内读回的是旧值（如 selectSkill 后 emit 截断文本）。
 *   显式传入本次写入的文本可避免该竞态。
 */
function emitMentionChange(textOverride) {
  emit('mention-change', {
    text: String(textOverride != null ? textOverride : model.value || ''),
    id: mentionedImage.value?.id != null ? String(mentionedImage.value.id) : '',
  })
}

function selectImage(img) {
  if (!img?.id) return
  const el = areaRef.value
  const t = model.value
  const caret = caretPos()
  const q = findActiveAtQuery(t, caret) || activeQuery.value
  const replaced = q ? replaceAtQueryWithMention(t, q.start, caret, img.name) : { text: t, caret }
  const withSpace = {
    text: `${replaced.text} `,
    caret: replaced.caret + 1,
  }
  model.value = withSpace.text
  imageMenuOpen.value = false
  activeQuery.value = null
  refreshMenus()
  emit('mention-change', { text: withSpace.text, id: String(img.id) })
  el?.focus()
}

function selectSkill(skill) {
  if (!skill?.name) return
  const el = areaRef.value
  const t = model.value
  const caret = caretPos()
  const replaced = insertSkillToken(t, caret, skill.name)
  model.value = replaced.text
  skillMenuOpen.value = false
  activeSkillQuery.value = null
  emitMentionChange(replaced.text)
  el?.focus()
}

/**
 * 技能 chip 点击：把标签上的完整 `$镜像 /技能` 写入描述自由段。
 * 不可只插 `/技能`——空描述（项目默认镜像已绑定）时会让 mention-change 带空 id，父级解绑镜像、chips 消失。
 */
function insertSkill(name) {
  const mention = chipLabel({ name }).trim()
  const next = applyImageMentionToDescription(model.value, mention, props.installedImages)
  model.value = next
  const id = props.image?.id != null ? String(props.image.id) : ''
  emit('mention-change', { text: next, id })
  areaRef.value?.focus()
}

function onKeydown(e) {
  if (skillMenuOpen.value && filteredSkills.value.length) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      skillHighlightIndex.value = (skillHighlightIndex.value + 1) % filteredSkills.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      skillHighlightIndex.value =
        (skillHighlightIndex.value - 1 + filteredSkills.value.length) % filteredSkills.value.length
      return
    }
    if ((e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey) || e.key === 'Tab') {
      e.preventDefault()
      selectSkill(filteredSkills.value[skillHighlightIndex.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      skillMenuOpen.value = false
      return
    }
  }

  if (imageMenuOpen.value && filteredImages.value.length) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      imageHighlightIndex.value = (imageHighlightIndex.value + 1) % filteredImages.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      imageHighlightIndex.value =
        (imageHighlightIndex.value - 1 + filteredImages.value.length) % filteredImages.value.length
      return
    }
    if ((e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey) || e.key === 'Tab') {
      e.preventDefault()
      selectImage(filteredImages.value[imageHighlightIndex.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      imageMenuOpen.value = false
      return
    }
  }
}
</script>
