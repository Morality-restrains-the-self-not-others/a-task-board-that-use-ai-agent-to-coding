// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useQueuedAutoRunPanel } from './useQueuedAutoRunPanel.js'

describe('useQueuedAutoRunPanel', () => {
  it('未入队时 statusText 为空；入队且有前方任务时展示等待数', () => {
    const modalOpen = ref(false)
    const patchTask = async () => true
    const propsOff = {
      task: { id: 'a', queued_auto_run: false },
      tenantId: 't',
      workspaceId: 'w',
    }
    const off = useQueuedAutoRunPanel(propsOff, { patchTask, modalOpen })
    expect(off.statusText.value).toBe('')

    const propsOn = {
      task: {
        id: 'b',
        queued_auto_run: true,
        queued_auto_run_status: 'queued',
        queued_ahead_count: 3,
      },
      tenantId: 't',
      workspaceId: 'w',
    }
    const on = useQueuedAutoRunPanel(propsOn, { patchTask, modalOpen })
    expect(on.statusText.value).toContain('前方还有 3 个任务在等待')
    expect(on.aheadCount.value).toBe(3)
  })
})
