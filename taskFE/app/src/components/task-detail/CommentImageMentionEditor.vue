<template>
  <div class="relative" data-testid="comment-image-mention-editor">
    <div
      id="comment-content"
      ref="editorEl"
      class="comment-mention-editor w-full text-sm p-2 border border-gray-300 rounded-md resize-y min-h-[4rem] focus:outline-none focus:ring-primary focus:border-primary overflow-y-auto whitespace-pre-wrap break-words"
      contenteditable="true"
      role="textbox"
      aria-multiline="true"
      :aria-label="placeholder"
      :data-placeholder="placeholder"
      data-testid="comment-content-editor"
      @input="onInput"
      @keydown="onKeydown"
      @mouseup="onCaretMaybe"
      @keyup="onCaretMaybe"
      @blur="onBlur"
      @paste="onPaste"
    />
    <ul
      v-if="menuOpen && filteredImages.length"
      class="absolute z-20 left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg"
      data-testid="comment-image-mention-picker"
      role="listbox"
    >
      <li
        v-for="(img, idx) in filteredImages"
        :key="img.id"
        role="option"
        :aria-selected="idx === highlightIndex"
        class="px-3 py-1.5 text-sm cursor-pointer"
        :class="idx === highlightIndex ? 'bg-primary/10 text-primary' : 'hover:bg-gray-50'"
        @mousedown.prevent="selectImage(img)"
      >
        {{ img.name || img.id }}{{ img.version ? ':' + img.version : '' }}
      </li>
    </ul>
    <ul
      v-if="skillMenuOpen && filteredSkills.length"
      class="absolute z-20 left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg"
      data-testid="comment-image-skill-picker"
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
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import {
  extractMentionFromDom,
  extractMentionFromPlainText,
  filterInstalledImages,
  findActiveAtQuery,
  matchImageByExactName,
  replaceAtQueryWithMention,
  serializeComposerDom,
} from '../../composables/taskDetail/commentImageMentionComposer.js'
import { setPendingImageMention } from '../../composables/taskDetail/commentImageMentionState.js'
import {
  extractSkillFromPlainText,
  filterImageSkills,
  findActiveSlashSkillQuery,
  imageSkillsFromImage,
  insertSkillToken,
} from '../../utils/imageSkills.js'

const model = defineModel({ type: String, required: true })

const props = defineProps({
  installedImages: { type: Array, default: () => [] },
  placeholder: { type: String, default: '写下你的评论… 输入 $ 选择镜像后可用 /技能' },
})

const emit = defineEmits(['keydown', 'mention-change'])

const editorEl = ref(null)
const menuOpen = ref(false)
const activeQuery = ref(null)
const highlightIndex = ref(0)
const skillMenuOpen = ref(false)
const activeSkillQuery = ref(null)
const skillHighlightIndex = ref(0)
const suppressModelWatch = ref(false)

const filteredImages = computed(() =>
  filterInstalledImages(props.installedImages, activeQuery.value?.query || ''),
)

const mentionedImage = computed(() => {
  const fromDom = extractMentionFromDom(editorEl.value)
  const id = fromDom?.id
  if (!id) return null
  return (props.installedImages || []).find((img) => String(img.id) === String(id)) || null
})

const mentionedSkills = computed(() => imageSkillsFromImage(mentionedImage.value).skills)

const filteredSkills = computed(() =>
  filterImageSkills(mentionedSkills.value, activeSkillQuery.value?.query || ''),
)

function skillsForMention(mention) {
  if (!mention?.id) return []
  const img = (props.installedImages || []).find((item) => String(item.id) === String(mention.id))
  return imageSkillsFromImage(img).skills
}

function mentionWithSkill(mention, plain) {
  if (!mention?.id) return mention
  const skill = extractSkillFromPlainText(plain, skillsForMention(mention))
  return { ...mention, skill }
}

function syncMentionState(plain) {
  const fromDom = extractMentionFromDom(editorEl.value)
  const mention = mentionWithSkill(fromDom || extractMentionFromPlainText(plain, props.installedImages), plain)
  setPendingImageMention(mention)
  emit('mention-change', mention)
  return mention
}

function getCaretPlainOffset() {
  const root = editorEl.value
  const sel = window.getSelection()
  if (!root || !sel || !sel.rangeCount) return serializeComposerDom(root).length
  const range = sel.getRangeAt(0)
  if (!root.contains(range.startContainer)) return serializeComposerDom(root).length
  const pre = range.cloneRange()
  pre.selectNodeContents(root)
  pre.setEnd(range.startContainer, range.startOffset)
  const probe = document.createElement('div')
  probe.appendChild(pre.cloneContents())
  return serializeComposerDom(probe).length
}

function setCaretPlainOffset(offset) {
  const root = editorEl.value
  if (!root) return
  const target = Math.max(0, Number(offset) || 0)
  const sel = window.getSelection()
  if (!sel) return

  let walked = 0
  const walk = (node) => {
    if (node.nodeType === 3) {
      const len = (node.nodeValue || '').length
      if (walked + len >= target) {
        const range = document.createRange()
        range.setStart(node, target - walked)
        range.collapse(true)
        sel.removeAllRanges()
        sel.addRange(range)
        return true
      }
      walked += len
      return false
    }
    if (node.nodeType !== 1) return false
    const el = node
    if (el.dataset && el.dataset.mentionId) {
      const name = String(el.dataset.mentionName || '').replace(/^\$/, '')
      const token = `$${name}`
      if (walked + token.length >= target) {
        const range = document.createRange()
        range.setStartAfter(el)
        range.collapse(true)
        sel.removeAllRanges()
        sel.addRange(range)
        return true
      }
      walked += token.length
      return false
    }
    if (el.tagName === 'BR') {
      if (walked + 1 >= target) {
        const range = document.createRange()
        range.setStartBefore(el)
        range.collapse(true)
        sel.removeAllRanges()
        sel.addRange(range)
        return true
      }
      walked += 1
      return false
    }
    for (const child of el.childNodes) {
      if (walk(child)) return true
    }
    return false
  }

  for (const child of root.childNodes) {
    if (walk(child)) return
  }
  const range = document.createRange()
  range.selectNodeContents(root)
  range.collapse(false)
  sel.removeAllRanges()
  sel.addRange(range)
}

function renderPlainWithMention(plain, mention) {
  const root = editorEl.value
  if (!root) return
  const text = String(plain || '')
  root.innerHTML = ''
  if (!text) return

  if (!mention?.id || !mention?.name) {
    root.textContent = text
    return
  }

  const token = `$${String(mention.name).replace(/^\$/, '')}`
  const idx = text.indexOf(token)
  if (idx < 0) {
    root.textContent = text
    return
  }

  if (idx > 0) root.appendChild(document.createTextNode(text.slice(0, idx)))
  const chip = document.createElement('span')
  chip.contentEditable = 'false'
  chip.dataset.mentionId = String(mention.id)
  chip.dataset.mentionName = String(mention.name)
  chip.className = 'comment-mention-chip'
  chip.textContent = token
  root.appendChild(chip)
  const rest = text.slice(idx + token.length)
  if (rest) root.appendChild(document.createTextNode(rest))
}

function refreshMenuFromCaret() {
  const plain = serializeComposerDom(editorEl.value)
  const caret = getCaretPlainOffset()
  const existing = extractMentionFromDom(editorEl.value)
  if (existing) {
    menuOpen.value = false
    activeQuery.value = null
    const sq = mentionedSkills.value.length ? findActiveSlashSkillQuery(plain, caret) : null
    activeSkillQuery.value = sq
    skillMenuOpen.value = !!sq
    skillHighlightIndex.value = 0
    return
  }
  skillMenuOpen.value = false
  activeSkillQuery.value = null
  const q = findActiveAtQuery(plain, caret)
  activeQuery.value = q
  menuOpen.value = !!q
  highlightIndex.value = 0
}

function pushModel(plain) {
  suppressModelWatch.value = true
  model.value = plain
  nextTick(() => { suppressModelWatch.value = false })
}

function onInput() {
  const plain = serializeComposerDom(editorEl.value)
  pushModel(plain)
  syncMentionState(plain)
  refreshMenuFromCaret()
  // OPT-20260824-077: 手动粘贴（onPaste 走 execCommand insertText 仅触发 onInput）
  // 或纯文本输入路径不渲染 mention chip —— chip 只在 watch(model)/onMounted/
  // selectImage/selectSkill 渲染。若镜像下拉未开（无进行中的 $ 查询，例如 caret 已
  // 越过 $镜像 后的 / 或空格）但正文含完整 $镜像 且 DOM 无 chip，重建 chip，
  // 使 $镜像 后的 / 技能菜单（依赖 chip）可开启。
  if (!menuOpen.value) {
    const mention = extractMentionFromPlainText(plain, props.installedImages)
    if (mention && !extractMentionFromDom(editorEl.value)) {
      renderPlainWithMention(plain, mention)
    }
  }
}

function onCaretMaybe() {
  refreshMenuFromCaret()
}

function onBlur() {
  // 延迟关闭，允许 mousedown 选中菜单项
  setTimeout(() => { menuOpen.value = false }, 120)
}

function onPaste(e) {
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain') || ''
  document.execCommand('insertText', false, text)
}

function selectImage(img) {
  if (!img?.id) return
  const plain = serializeComposerDom(editorEl.value)
  const caret = getCaretPlainOffset()
  const q = findActiveAtQuery(plain, caret) || activeQuery.value
  if (!q) return
  const replaced = replaceAtQueryWithMention(plain, q.start, caret, img.name)
  // 选中后补一个空格，便于继续输入指令
  const withSpace = {
    text: `${replaced.text} `,
    caret: replaced.caret + 1,
  }
  const mention = mentionWithSkill({ id: String(img.id), name: String(img.name || '') }, withSpace.text)
  renderPlainWithMention(withSpace.text, mention)
  pushModel(withSpace.text)
  setPendingImageMention(mention)
  emit('mention-change', mention)
  menuOpen.value = false
  activeQuery.value = null
  nextTick(() => setCaretPlainOffset(withSpace.caret))
}

function selectSkill(skill) {
  if (!skill?.name) return
  const plain = serializeComposerDom(editorEl.value)
  const caret = getCaretPlainOffset()
  const replaced = insertSkillToken(plain, caret, skill.name)
  const mention = extractMentionFromDom(editorEl.value) || extractMentionFromPlainText(replaced.text, props.installedImages)
  const nextMention = mentionWithSkill(mention, replaced.text)
  if (nextMention) nextMention.skill = skill.name
  renderPlainWithMention(replaced.text, nextMention)
  pushModel(replaced.text)
  setPendingImageMention(nextMention)
  emit('mention-change', nextMention)
  skillMenuOpen.value = false
  activeSkillQuery.value = null
  nextTick(() => setCaretPlainOffset(replaced.caret))
}

function tryConfirmExactBySpace(e) {
  const plain = serializeComposerDom(editorEl.value)
  const caret = getCaretPlainOffset()
  const q = findActiveAtQuery(plain, caret)
  if (!q || !q.query) return false
  const hit = matchImageByExactName(props.installedImages, q.query)
  if (!hit) {
    menuOpen.value = false
    activeQuery.value = null
    return false
  }
  e.preventDefault()
  selectImage(hit)
  return true
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
    if ((e.key === 'Enter' && !e.metaKey && !e.ctrlKey) || e.key === 'Tab') {
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

  if (menuOpen.value && filteredImages.value.length) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      highlightIndex.value = (highlightIndex.value + 1) % filteredImages.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      highlightIndex.value =
        (highlightIndex.value - 1 + filteredImages.value.length) % filteredImages.value.length
      return
    }
    if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
      e.preventDefault()
      selectImage(filteredImages.value[highlightIndex.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      menuOpen.value = false
      return
    }
    if (e.key === 'Tab') {
      e.preventDefault()
      selectImage(filteredImages.value[highlightIndex.value])
      return
    }
  }

  if (e.key === ' ' || e.key === 'Spacebar') {
    if (tryConfirmExactBySpace(e)) return
  }

  emit('keydown', e)
}

watch(
  () => model.value,
  (v) => {
    if (suppressModelWatch.value) return
    const next = String(v || '')
    const current = serializeComposerDom(editorEl.value)
    if (next === current) return
    const mention = extractMentionFromPlainText(next, props.installedImages)
    renderPlainWithMention(next, mention)
    syncMentionState(next)
    if (!next) {
      menuOpen.value = false
      activeQuery.value = null
    }
  },
)

onMounted(() => {
  const next = String(model.value || '')
  if (next) {
    const mention = extractMentionFromPlainText(next, props.installedImages)
    renderPlainWithMention(next, mention)
    syncMentionState(next)
  }
})
</script>

<style scoped>
.comment-mention-editor:empty:before {
  content: attr(data-placeholder);
  color: #9ca3af;
  pointer-events: none;
}
.comment-mention-chip {
  display: inline;
  color: #4f46e5;
  background: #eef2ff;
  border-radius: 0.25rem;
  padding: 0 0.2rem;
  font-weight: 600;
}
</style>
