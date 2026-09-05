/**
 * 支付状态轮询（单链 + 可取消）— OPT-20260808-015
 *
 * OrderCreate 支付二维码弹窗的状态轮询原实现是 fire-and-forget 的
 * `for (60) { sleep(2s); GET }`：用户关闭弹窗 / 离开页面后，轮询链仍每 2s
 * 打一次 GET 最多 2 分钟 —— 浏览器网络与 CPU 挂起的源头之一。
 *
 * 本 composable 承诺：
 *  - 单链：startPoll 先取消旧链（重复点「确认支付」不产生并发轮询链）
 *  - isActive(orderId) 返回 false 即停止（弹窗关闭 / 订单变化，不回写旧单状态）
 *  - stopPoll 立即取消（组件 onUnmounted 调用）
 *  - maxAttempts 硬上限兜底
 *  - setTimeout 链式调度：慢网下不会堆积并发 GET（setInterval 的 async 回调会重叠）
 *
 * 用法：
 *   const { startPoll, stopPoll } = usePaymentPoll({
 *     getOrderId: () => currentOrder.value?.id,
 *     isActive: (orderId) => wechatQrVisible.value && currentOrder.value?.id === orderId,
 *     onPoll: async (orderId) => { /* 查单；已支付时更新状态并返回 true *\/ },
 *   })
 */
export function usePaymentPoll({
  getOrderId,
  isActive,
  onPoll,
  intervalMs = 2000,
  maxAttempts = 60,
}) {
  let timerId = null
  let chainId = 0

  const stopPoll = () => {
    // chainId 递增使任何在途/已排程的旧链 tick 全部失效
    chainId += 1
    if (timerId !== null) {
      clearTimeout(timerId)
      timerId = null
    }
  }

  const startPoll = () => {
    // 捕获发起时的订单 id：轮询期间订单变化（新订单）即停止旧链
    const orderId = getOrderId()
    if (!orderId) return
    stopPoll() // 单链：先取消旧链
    const myChain = chainId
    let attempts = 0

    const tick = async () => {
      if (myChain !== chainId) return
      if (attempts >= maxAttempts || !isActive(orderId)) {
        stopPoll()
        return
      }
      attempts += 1
      try {
        const done = await onPoll(orderId)
        if (done) {
          stopPoll()
          return
        }
      } catch {
        /* 网络抖动：继续下一轮 */
      }
      // await 期间若被 stopPoll / 新链取代，不再调度
      if (myChain === chainId) {
        timerId = setTimeout(tick, intervalMs)
      }
    }

    timerId = setTimeout(tick, intervalMs)
  }

  return { startPoll, stopPoll }
}
