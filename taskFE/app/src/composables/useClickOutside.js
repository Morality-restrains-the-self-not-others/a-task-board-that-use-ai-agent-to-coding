import { onBeforeUnmount, onMounted, unref } from 'vue'

/**
 * 点击目标元素外部时调用 handler。
 * @param {import('vue').Ref<EventTarget|null|undefined>|(() => EventTarget|null|undefined)} targetRef
 * @param {() => void} handler
 * @param {{ enabled?: import('vue').Ref<boolean>|boolean|(() => boolean), capture?: boolean }} [opts]
 */
export function useClickOutside(targetRef, handler, opts = {}) {
  const capture = opts.capture !== false

  const isEnabled = () => {
    if (opts.enabled === undefined) return true
    if (typeof opts.enabled === 'function') return Boolean(opts.enabled())
    return Boolean(unref(opts.enabled))
  }

  const resolveTarget = () => {
    if (typeof targetRef === 'function') return targetRef()
    return unref(targetRef)
  }

  function onDocumentClick(event) {
    if (!isEnabled()) return
    const target = event?.target
    if (!(target instanceof Node)) return
    const root = resolveTarget()
    if (!root || !(root instanceof Node)) return
    if (root.contains(target)) return
    handler()
  }

  function bind() {
    if (typeof document === 'undefined') return
    document.addEventListener('click', onDocumentClick, capture)
  }

  function unbind() {
    if (typeof document === 'undefined') return
    document.removeEventListener('click', onDocumentClick, capture)
  }

  onMounted(bind)
  onBeforeUnmount(unbind)

  return { bind, unbind }
}
