/**
 * 模态框回车键确认 composable。
 * 在弹窗内容根元素上绑定 @keydown.enter="onModalEnter"，回车时触发主操作。
 * 自动跳过 textarea（回车应插入换行）。
 *
 * 用法：
 *   const { onModalEnter } = useModalEnterConfirm(() => emit('confirm'))
 *   // 模板: <div @keydown.enter="onModalEnter"> ... </div>
 */
export function useModalEnterConfirm(confirmFn) {
  function onModalEnter(event) {
    const tag = (event.target?.tagName || '').toLowerCase()
    if (tag === 'textarea') return
    if (typeof confirmFn === 'function') confirmFn()
  }

  return { onModalEnter }
}
