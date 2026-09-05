export const AVAILABLE_INSTANCES_DEBOUNCE_MS = 300

/**
 * 合并/取消可用实例列表拉取意图：debounce 期间仅保留最新一代，in-flight 由调用方 AbortController 取消。
 */
export function createAvailableInstancesFetchScheduler({
  debounceMs = AVAILABLE_INSTANCES_DEBOUNCE_MS,
  onRun,
} = {}) {
  let debounceTimer = null
  let generation = 0

  function clearScheduled() {
    if (debounceTimer != null) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
  }

  function bumpGeneration() {
    generation += 1
    return generation
  }

  function schedule({ immediate = false, onAbort } = {}) {
    clearScheduled()
    onAbort?.()
    const intentGeneration = bumpGeneration()
    const run = () => {
      if (intentGeneration !== generation) {
        return
      }
      debounceTimer = null
      onRun?.(intentGeneration)
    }
    if (immediate) {
      run()
    } else {
      debounceTimer = setTimeout(run, debounceMs)
    }
    return intentGeneration
  }

  function beginFetch(intentGeneration) {
    const fetchGeneration = intentGeneration ?? bumpGeneration()
    return {
      fetchGeneration,
      isStale: () => fetchGeneration !== generation,
      currentGeneration: () => generation,
    }
  }

  return {
    schedule,
    beginFetch,
    clearScheduled,
    currentGeneration: () => generation,
  }
}
