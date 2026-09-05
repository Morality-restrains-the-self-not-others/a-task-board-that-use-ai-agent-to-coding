import { ref } from 'vue'

/**
 * 判断当前值是否来自 datalist 候选（精确匹配）。
 * @param {unknown[]} candidates
 * @param {unknown} value
 * @returns {boolean}
 */
export function shouldDismissDatalistPick(candidates, value) {
  const trimmed = String(value ?? '').trim()
  if (!trimmed || !Array.isArray(candidates) || candidates.length === 0) return false
  return candidates.map((item) => String(item)).includes(trimmed)
}

/**
 * datalist 选中时的 input 事件：Chrome 等为 insertReplacementText。
 * 普通逐字输入为 insertText，不应在 input 阶段失焦（避免候选名是更长分支名前缀时误伤）。
 * @param {Event | undefined | null} event
 * @returns {boolean}
 */
export function isDatalistPickerInputEvent(event) {
  if (!event || event.type !== 'input') return false
  const inputType = event.inputType
  return inputType === 'insertReplacementText' || inputType === 'insertFromDrop'
}

/**
 * Vue 受控 input + 原生 datalist：选中候选后若仍保持焦点，
 * Chrome 会在补丁 value 后再次打开建议列表。仅 blur 不够——
 * 绑定的 :list 会在重渲染时写回。选中后抑制 list 直至下次 focus；
 * 并在 DOM 上同步 removeAttribute('list')，避免等 Vue 补丁前的一帧复现。
 */
export function createDatalistDismissController() {
  const suppressedIds = ref(new Set())

  const listAttr = (inputId, datalistId) => {
    const id = String(inputId || '')
    if (!id || suppressedIds.value.has(id)) return undefined
    return datalistId
  }

  const dismissIfPicked = (inputEl, candidates, value, event) => {
    if (!shouldDismissDatalistPick(candidates, value)) return false
    // input 阶段仅处理「从候选列表选中」；change 始终处理
    if (event?.type === 'input' && !isDatalistPickerInputEvent(event)) return false

    const id = String(inputEl?.id || '')
    if (id) {
      const next = new Set(suppressedIds.value)
      next.add(id)
      suppressedIds.value = next
    }
    // 同步摘掉 list，避免 Vue 下一帧补丁前 Chrome 再次拉起建议 UI
    if (inputEl && typeof inputEl.removeAttribute === 'function') {
      inputEl.removeAttribute('list')
    }
    inputEl?.blur?.()
    return true
  }

  const restoreOnFocus = (inputElOrEvent) => {
    const inputEl = inputElOrEvent?.target ?? inputElOrEvent
    const id = String(inputEl?.id || '')
    if (!id || !suppressedIds.value.has(id)) return
    const next = new Set(suppressedIds.value)
    next.delete(id)
    suppressedIds.value = next
  }

  return {
    listAttr,
    dismissIfPicked,
    restoreOnFocus,
  }
}
