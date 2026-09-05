// @vitest-environment node
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import {
  updateServerStatus,
  LAYER_JOB_LIVE_OUTPUT_MAX_CHARS,
  resolveContainerJobStreamTerminalStatus,
  patchLayerGraphJobStatus,
} from './updateServerStatus.js'
import { mergeAgentStepsByNumber } from './fetchJobExecutionLogBySteps.js'
import { createCommentLayerPanelStore } from './commentLayerPanelStore.js'

describe('updateServerStatus — relay_to_trae_status', () => {
  let deps
  let serverConfigRef

  beforeEach(() => {
    serverConfigRef = ref({
      applyRelayToTraeStatusFromSse: vi.fn(),
    })

    deps = {
      serverConfigRef,
      activeAiInstructId: ref(null),
      aiStreamBuffer: ref(''),
      aiStreamBusy: ref(false),
      activeContainerAgentId: ref(null),
      containerBootstrapCloneLogFull: ref(''),
      containerCloneProgressByKey: ref({}),
      containerEndpointRegistered: ref(false),
      containerPageUrl: ref(''),
      containerVscodeUrl: ref(''),
      containerPageLinkPendingReveal: ref(false),
      layerGraphSnapshot: ref({ layers: [], jobs: [], layers_root: '', bootstrap_layer_id: '' }),
      containerBootstrapFailureMessage: ref(''),
      containerBootstrapFailureTraceId: ref(''),
      layerChangesByLayerId: ref({}),
      layerJobLiveOutputMap: ref({}),
      layerJobExecutionPayload: ref(null),
      isServerRunning: ref(false),
      isServerStarting: ref(false),
      serverUrl: ref(''),
      serverStatus: ref(''),
      statusMessage: ref(''),
      statusEventCommentId: ref(''),
      statusEventRuntimeStatus: ref(''),
      statusTraceId: ref(''),
      statusProgress: ref(0),
      statusLogs: ref([]),
      fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      markContainerTransportOk: vi.fn(),
      onBootstrapCloneLogUpdate: vi.fn(),
      refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
      bumpProjectFileTreeRefresh: vi.fn(),
      rescheduleContainerCloneProgressClearTimer: vi.fn(),
      normalizeLayerChangesPayload: vi.fn(),
      applyLiveLayerChangesToCurrentPayload: vi.fn(),
      refreshZTreeExecutionLog: vi.fn(),
      stopContainerHeartbeat: vi.fn(),
      fetchContainerTaskUiContext: vi.fn().mockResolvedValue(undefined),
      CONTAINER_CLONE_PROGRESS_GLOBAL_KEY: '__global__',
      containerCloneProgressTimer: null,
    }
  })

  describe('relay_to_trae_status event', () => {
    it('calls applyRelayToTraeStatusFromSse on serverConfigRef when status is relay_to_trae_status', () => {
      const statusData = {
        status: 'relay_to_trae_status',
        relay_payload: { running: true, online_service_up: true, ui_url: 'http://example.com' },
      }

      updateServerStatus(statusData, deps)

      expect(
        serverConfigRef.value.applyRelayToTraeStatusFromSse,
      ).toHaveBeenCalledTimes(1)
      expect(
        serverConfigRef.value.applyRelayToTraeStatusFromSse,
      ).toHaveBeenCalledWith(statusData)
    })

    it('returns early after handling relay_to_trae_status (does not update statusMessage)', () => {
      const statusData = {
        status: 'relay_to_trae_status',
        relay_payload: { running: false },
      }

      updateServerStatus(statusData, deps)

      // Should NOT fall through to the generic status handler
      expect(deps.statusMessage.value).toBe('')
      expect(deps.serverStatus.value).toBe('')
    })

    it('works when serverConfigRef value has no applyRelayToTraeStatusFromSse method', () => {
      serverConfigRef.value = { otherMethod: vi.fn() }
      const statusData = {
        status: 'relay_to_trae_status',
        relay_payload: { running: true },
      }

      // Should not throw
      expect(() => updateServerStatus(statusData, deps)).not.toThrow()
    })

    it('works when serverConfigRef value is null', () => {
      serverConfigRef.value = null
      const statusData = { status: 'relay_to_trae_status' }

      // Should not throw
      expect(() => updateServerStatus(statusData, deps)).not.toThrow()
    })
  })

  describe('container_bootstrap_failed / complete', () => {
    it('records BOOTSTRAP_FAILED message for task-association hint', () => {
      updateServerStatus(
        {
          status: 'container_bootstrap_failed',
          phase: 'task_detail_or_credentials',
          message:
            'repo-clone-credentials 未返回完整；缺失仓库(1): https://github.com/ruandao/somanyad',
        },
        deps,
      )
      expect(deps.containerBootstrapFailureMessage.value).toContain('somanyad')
      expect(deps.statusTraceId.value).toBe('')
    })

    it('records BOOTSTRAP_FAILED SSE trace_id without writing task-level statusTraceId', () => {
      updateServerStatus(
        {
          status: 'container_bootstrap_failed',
          phase: 'task_detail_or_credentials',
          message: 'repo-clone-credentials 未返回完整',
          trace_id: 'boot-sse-trace-1',
        },
        deps,
      )
      expect(deps.containerBootstrapFailureTraceId.value).toBe('boot-sse-trace-1')
      expect(deps.statusTraceId.value).toBe('')
    })

    it('omits bootstrap trace_id when SSE has none', () => {
      updateServerStatus(
        { status: 'container_bootstrap_failed', message: '引导失败' },
        deps,
      )
      expect(deps.containerBootstrapFailureTraceId.value).toBe('')
    })

    it('clears failure and refreshes layer graph on BOOTSTRAP_COMPLETE', () => {
      deps.containerBootstrapFailureMessage.value = 'old failure'
      deps.containerBootstrapFailureTraceId.value = 'old-trace'
      updateServerStatus({ status: 'container_bootstrap_complete', message: 'ok' }, deps)
      expect(deps.containerBootstrapFailureMessage.value).toBe('')
      expect(deps.containerBootstrapFailureTraceId.value).toBe('')
      expect(deps.refreshLayerGraphFromServer).toHaveBeenCalledWith(true)
      expect(deps.bumpProjectFileTreeRefresh).toHaveBeenCalled()
    })

    it('does not clear BOOTSTRAP_FAILED when layer-graph only has empty pending anchor', () => {
      deps.containerBootstrapFailureMessage.value =
        'repo-clone-credentials 未返回完整；缺失仓库(1): http://115.29.110.74/example-user/somanyad.git'
      deps.containerBootstrapFailureTraceId.value = 'boot-sse-trace-1'
      updateServerStatus(
        {
          status: 'container_layer_graph',
          layers: [
            {
              layer_id: '20260827_074114_cdea43',
              meta_kind: 'empty',
              bootstrap_pending: true,
              created_at: '2026-08-27T07:41:14Z',
            },
          ],
          jobs: [],
          layers_root: '/workspace/layers',
          bootstrap_layer_id: '',
        },
        deps,
      )
      expect(deps.containerBootstrapFailureMessage.value).toContain('somanyad')
      expect(deps.containerBootstrapFailureTraceId.value).toBe('boot-sse-trace-1')
    })
  })

  describe('container_layer_graph empty layers', () => {
    it('ignores empty layers when a non-empty snapshot already exists', () => {
      deps.layerGraphSnapshot.value = {
        layers: [{ layer_id: 'L1', created_at: '2026-01-01T00:00:00Z' }],
        jobs: [{ id: 'J1', layer_id: 'L1', status: 'running' }],
        layers_root: '/layers',
        bootstrap_layer_id: 'L1',
      }

      updateServerStatus(
        {
          status: 'container_layer_graph',
          layers: [],
          jobs: [],
          layers_root: '',
          bootstrap_layer_id: '',
        },
        deps,
      )

      expect(deps.markContainerTransportOk).toHaveBeenCalled()
      expect(deps.layerGraphSnapshot.value.layers).toHaveLength(1)
      expect(deps.layerGraphSnapshot.value.layers[0].layer_id).toBe('L1')
      expect(deps.layerGraphSnapshot.value.jobs).toHaveLength(1)
    })

    it('accepts empty layers when snapshot was already empty', () => {
      deps.layerGraphSnapshot.value = {
        layers: [],
        jobs: [],
        layers_root: '',
        bootstrap_layer_id: '',
      }

      updateServerStatus(
        {
          status: 'container_layer_graph',
          layers: [],
          jobs: [],
          layers_root: '/x',
          bootstrap_layer_id: '',
        },
        deps,
      )

      expect(deps.layerGraphSnapshot.value.layers_root).toBe('/x')
    })
  })

  describe('container_job_stream live output cap', () => {
    it('截断超长实时输出，避免 map 内字符串无限增长', () => {
      const jid = 'job-cap'
      deps.layerJobLiveOutputMap.value = {
        [jid]: 'x'.repeat(LAYER_JOB_LIVE_OUTPUT_MAX_CHARS - 10),
      }
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'chunk',
          message: 'y'.repeat(50),
        },
        deps,
      )
      const stored = deps.layerJobLiveOutputMap.value[jid]
      expect(stored.length).toBe(LAYER_JOB_LIVE_OUTPUT_MAX_CHARS)
      expect(stored.endsWith('y'.repeat(50))).toBe(true)
    })

    it('phase=step 时追加摘要并刷新执行日志（逐步推送）', () => {
      const jid = 'job-step'
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'step',
          message: 'step 1: bash ls',
        },
        deps,
      )
      expect(deps.layerJobLiveOutputMap.value[jid]).toContain('step 1: bash ls')
      expect(deps.refreshZTreeExecutionLog).toHaveBeenCalledTimes(1)
      expect(deps.refreshLayerGraphFromServer).not.toHaveBeenCalled()
    })

    it('phase=step 时即时合并 step_number/delivery_summary 进 steps，GET 迟到缺页不丢步（OPT-20260816-042）', () => {
      const jid = 'job-step-merge'
      deps.layerJobExecutionPayload.value = {
        job: { id: jid, status: 'running' },
        steps: { steps: [{ step_number: 1, delivery_summary: 'step 1' }] },
      }
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'step',
          message: 'step 2: bash ls',
          step_number: 2,
          delivery_summary: 'bash ls',
          state: 'completed',
        },
        deps,
      )
      let steps = deps.layerJobExecutionPayload.value.steps.steps
      expect(steps.map((s) => s.step_number)).toEqual([1, 2])
      const s2 = steps.find((s) => s.step_number === 2)
      expect(s2.delivery_summary).toBe('bash ls')
      expect(s2.state).toBe('completed')
      // persist 消费者异步：GET 迟到且缺 step 2 时，按 step_number 合并仍保留 step 2
      steps = mergeAgentStepsByNumber(steps, [{ step_number: 1, delivery_summary: 'step 1' }])
      expect(steps.map((s) => s.step_number)).toEqual([1, 2])
    })

    it('phase=step 无已有 payload 时不构造假 payload（交给 GET hydrate 兜底）', () => {
      const jid = 'job-step-null-payload'
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'step',
          message: 'step 1: bash ls',
          step_number: 1,
          delivery_summary: 'bash ls',
        },
        deps,
      )
      expect(deps.layerJobExecutionPayload.value).toBeNull()
      expect(deps.refreshZTreeExecutionLog).toHaveBeenCalledTimes(1)
    })

    it('phase=completed（events 终态）应乐观改 job.status 并强制刷层图', () => {
      const jid = '387c72c4-2974-4555-8524-cf700d679f38'
      deps.layerGraphSnapshot.value = {
        layers: [
          {
            layer_id: '20260718_052302_236610',
            job_status: 'running',
            mind_state: 'running',
            job_id: jid,
          },
        ],
        jobs: [
          {
            id: jid,
            layer_id: '20260718_052302_236610',
            status: 'running',
            command: '把项目运行起来',
          },
        ],
        layers_root: '',
        bootstrap_layer_id: '',
      }
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'completed',
          message: '',
        },
        deps,
      )
      expect(deps.layerGraphSnapshot.value.jobs[0].status).toBe('completed')
      expect(deps.layerGraphSnapshot.value.layers[0].job_status).toBe('completed')
      expect(deps.layerGraphSnapshot.value.layers[0].mind_state).toBe('idle_done')
      expect(deps.refreshLayerGraphFromServer).toHaveBeenCalledWith(true, expect.objectContaining({ commentId: expect.any(String) }))
      expect(deps.refreshZTreeExecutionLog).toHaveBeenCalled()
    })

    it('phase=done + job_status=failed 应写入 failed 而非默认 completed', () => {
      const jid = 'job-fail'
      deps.layerGraphSnapshot.value = {
        layers: [{ layer_id: 'L1', job_status: 'running', mind_state: 'running' }],
        jobs: [{ id: jid, layer_id: 'L1', status: 'running' }],
        layers_root: '',
        bootstrap_layer_id: '',
      }
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: jid,
          phase: 'done',
          job_status: 'failed',
          message: '',
        },
        deps,
      )
      expect(deps.layerGraphSnapshot.value.jobs[0].status).toBe('failed')
      expect(deps.refreshLayerGraphFromServer).toHaveBeenCalledWith(true, expect.objectContaining({ commentId: expect.any(String) }))
    })

    it('phase=running 不应落入启动状态分支改写 serverStatus', () => {
      deps.serverStatus.value = 'success'
      updateServerStatus(
        {
          status: 'container_job_stream',
          job_id: 'j-run',
          phase: 'running',
          message: '',
        },
        deps,
      )
      expect(deps.serverStatus.value).toBe('success')
      expect(deps.refreshLayerGraphFromServer).not.toHaveBeenCalled()
    })
  })

  describe('resolveContainerJobStreamTerminalStatus / patchLayerGraphJobStatus', () => {
    it('maps done/completed/failed and ignores bare error', () => {
      expect(resolveContainerJobStreamTerminalStatus('completed', {})).toBe('completed')
      expect(resolveContainerJobStreamTerminalStatus('done', { job_status: 'interrupted' })).toBe(
        'interrupted',
      )
      expect(resolveContainerJobStreamTerminalStatus('done', {})).toBe('completed')
      expect(resolveContainerJobStreamTerminalStatus('error', {})).toBe('')
    })

    it('patchLayerGraphJobStatus updates matching job and layer', () => {
      const snap = ref({
        layers: [{ layer_id: 'L1', job_status: 'running', mind_state: 'running' }],
        jobs: [{ id: 'J1', layer_id: 'L1', status: 'running' }],
      })
      patchLayerGraphJobStatus(snap, 'J1', 'completed')
      expect(snap.value.jobs[0].status).toBe('completed')
      expect(snap.value.layers[0].job_status).toBe('completed')
      expect(snap.value.layers[0].mind_state).toBe('idle_done')
    })
  })

  describe('container_agent_stream', () => {
    it('appends chunks and sets busy for matching agent_comment_id', () => {
      updateServerStatus(
        {
          status: 'container_agent_stream',
          agent_comment_id: 'ac1',
          phase: 'chunk',
          message: 'hello ',
        },
        deps,
      )
      updateServerStatus(
        {
          status: 'container_agent_stream',
          agent_comment_id: 'ac1',
          phase: 'chunk',
          message: 'world',
        },
        deps,
      )
      expect(deps.activeContainerAgentId.value).toBe('ac1')
      expect(deps.aiStreamBusy.value).toBe(true)
      expect(deps.aiStreamBuffer.value).toBe('hello world')
      expect(deps.fetchTaskDetail).not.toHaveBeenCalled()
    })

    it('ignores chunks for a different agent_comment_id once active', () => {
      deps.activeContainerAgentId.value = 'ac1'
      deps.aiStreamBuffer.value = 'keep'
      updateServerStatus(
        {
          status: 'container_agent_stream',
          agent_comment_id: 'ac2',
          phase: 'chunk',
          message: 'other',
        },
        deps,
      )
      expect(deps.aiStreamBuffer.value).toBe('keep')
    })

    it('on done: clears busy, refetches detail, then clears buffer', async () => {
      deps.activeContainerAgentId.value = 'ac1'
      deps.aiStreamBusy.value = true
      deps.aiStreamBuffer.value = 'partial'
      let resolveFetch
      deps.fetchTaskDetail = vi.fn(
        () =>
          new Promise((resolve) => {
            resolveFetch = resolve
          }),
      )
      updateServerStatus(
        {
          status: 'container_agent_stream',
          agent_comment_id: 'ac1',
          phase: 'done',
          message: '',
        },
        deps,
      )
      expect(deps.aiStreamBusy.value).toBe(false)
      expect(deps.activeContainerAgentId.value).toBe(null)
      expect(deps.aiStreamBuffer.value).toBe('partial')
      expect(deps.fetchTaskDetail).toHaveBeenCalledTimes(1)
      resolveFetch()
      await Promise.resolve()
      await Promise.resolve()
      expect(deps.aiStreamBuffer.value).toBe('')
    })

    it('on error: appends message, clears busy, refetches', () => {
      deps.activeContainerAgentId.value = 'ac1'
      deps.aiStreamBusy.value = true
      deps.aiStreamBuffer.value = 'partial'
      updateServerStatus(
        {
          status: 'container_agent_stream',
          agent_comment_id: 'ac1',
          phase: 'error',
          message: ' boom',
        },
        deps,
      )
      expect(deps.aiStreamBuffer.value).toBe('partial boom')
      expect(deps.aiStreamBusy.value).toBe(false)
      expect(deps.activeContainerAgentId.value).toBe(null)
      expect(deps.fetchTaskDetail).toHaveBeenCalledTimes(1)
    })

    it('ignores events without agent_comment_id', () => {
      updateServerStatus(
        { status: 'container_agent_stream', phase: 'chunk', message: 'x' },
        deps,
      )
      expect(deps.aiStreamBuffer.value).toBe('')
    })
  })

  describe('runtime_hydrate', () => {
    it('Running → isServerRunning，不写 statusLogs，拉 container UI', () => {
      deps.cloudRuntimeStatus = ref('')
      deps.markServerRuntimeServing = vi.fn()
      deps.serverRuntimeNotServing = ref(true)

      updateServerStatus(
        { status: 'runtime_hydrate', runtime_status: 'Running' },
        deps,
      )

      expect(deps.isServerRunning.value).toBe(true)
      expect(deps.isServerStarting.value).toBe(false)
      expect(deps.cloudRuntimeStatus.value).toBe('Running')
      expect(deps.markServerRuntimeServing).toHaveBeenCalledTimes(1)
      expect(deps.statusLogs.value).toEqual([])
      expect(deps.fetchContainerTaskUiContext).toHaveBeenCalledTimes(1)
    })

    it('已 Running 再 hydrate 不重复拉 container UI', () => {
      deps.isServerRunning.value = true
      updateServerStatus(
        { status: 'runtime_hydrate', runtime_status: 'Running' },
        deps,
      )
      expect(deps.fetchContainerTaskUiContext).not.toHaveBeenCalled()
    })

    it('Stopped → 清 running + pause 心跳', () => {
      deps.pauseContainerHeartbeatForRelayStop = vi.fn()
      deps.isServerRunning.value = true
      deps.serverRuntimeNotServing = ref(false)

      updateServerStatus(
        { status: 'runtime_hydrate', runtime_status: 'Stopped' },
        deps,
      )

      expect(deps.isServerRunning.value).toBe(false)
      expect(deps.isServerStarting.value).toBe(false)
      expect(deps.serverStatus.value).toBe('stopped')
      expect(deps.serverRuntimeNotServing.value).toBe(true)
      expect(deps.pauseContainerHeartbeatForRelayStop).toHaveBeenCalledTimes(1)
      expect(deps.statusLogs.value).toEqual([])
    })
  })

  describe('stopped / error — pause heartbeat (released-server UI)', () => {
    it('calls pauseContainerHeartbeatForRelayStop on stopped', () => {
      deps.pauseContainerHeartbeatForRelayStop = vi.fn()
      deps.isServerRunning.value = true
      deps.serverUrl.value = 'http://127.0.0.1:8765'

      updateServerStatus({ status: 'stopped', message: '服务器已停止' }, deps)

      expect(deps.isServerRunning.value).toBe(false)
      expect(deps.isServerStarting.value).toBe(false)
      expect(deps.serverUrl.value).toBe('')
      expect(deps.pauseContainerHeartbeatForRelayStop).toHaveBeenCalledTimes(1)
      expect(deps.stopContainerHeartbeat).not.toHaveBeenCalled()
      expect(deps.fetchContainerTaskUiContext).toHaveBeenCalled()
    })

    it('falls back to stopContainerHeartbeat when pause helper missing', () => {
      delete deps.pauseContainerHeartbeatForRelayStop
      updateServerStatus({ status: 'error', message: '启动失败' }, deps)
      expect(deps.stopContainerHeartbeat).toHaveBeenCalledTimes(1)
    })
  })

  describe('stop-vm success SSE (success status + stop copy)', () => {
    it('clears running on 停止虚拟机成功', () => {
      deps.pauseContainerHeartbeatForRelayStop = vi.fn()
      deps.isServerRunning.value = true
      deps.serverUrl.value = 'http://127.0.0.1:8765'

      updateServerStatus(
        { status: 'success', message: '停止虚拟机成功', progress: 100 },
        deps,
      )

      expect(deps.isServerRunning.value).toBe(false)
      expect(deps.isServerStarting.value).toBe(false)
      expect(deps.serverUrl.value).toBe('')
      expect(deps.pauseContainerHeartbeatForRelayStop).toHaveBeenCalledTimes(1)
    })

    it('clears running on Mock 实例已停止 (not treat as start success)', () => {
      deps.pauseContainerHeartbeatForRelayStop = vi.fn()
      deps.isServerRunning.value = true
      deps.serverUrl.value = 'http://mock.local'

      updateServerStatus(
        { status: 'success', message: 'Mock 实例已停止', progress: 100 },
        deps,
      )

      expect(deps.isServerRunning.value).toBe(false)
      expect(deps.isServerStarting.value).toBe(false)
      expect(deps.serverUrl.value).toBe('')
      expect(deps.pauseContainerHeartbeatForRelayStop).toHaveBeenCalledTimes(1)
      expect(deps.fetchContainerTaskUiContext).toHaveBeenCalled()
    })
  })

  describe('server scheduling → per-binding startup log bus', () => {
    it('fans out comment-scoped processing messages to latestServerStartupStatusForBinding', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'processing',
          comment_id: 'C1',
          message: '[-、task_t1_C1] 正在自动创建前置资源并启动服务器...',
          progress: 45,
          event_name: 'server_status_update',
        },
        deps,
      )
      expect(deps.statusLogs.value.some((l) => l.includes('正在自动创建前置资源'))).toBe(true)
      expect(latestServerStartupStatusForBinding.value?.comment_id).toBe('C1')
      expect(latestServerStartupStatusForBinding.value?.messages?.[0]).toContain('正在自动创建前置资源')
      expect(deps.statusEventCommentId.value).toBe('C1')
      expect(deps.statusEventRuntimeStatus.value).toBe('')
    })

    it('clears statusEventCommentId when a later status event has no comment_id', () => {
      deps.statusEventCommentId.value = 'STALE'
      updateServerStatus(
        {
          status: 'processing',
          message: '正在启动服务器...',
          progress: 10,
        },
        deps,
      )
      expect(deps.statusEventCommentId.value).toBe('')
    })

    it('records runtime_status from start-success push for snapshot apply', () => {
      updateServerStatus(
        {
          status: 'success',
          comment_id: 'C9',
          message: 'aliyun服务器启动成功！',
          progress: 100,
          runtime_status: 'Running',
          event_name: 'server_status_update',
        },
        deps,
      )
      expect(deps.statusEventCommentId.value).toBe('C9')
      expect(deps.statusEventRuntimeStatus.value).toBe('Running')
    })

    it('carries trace_id to binding bus for per-binding start TraceId caching', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'processing',
          comment_id: 'C1',
          message: '[-、task_t1_C1] 正在启动服务器...',
          progress: 45,
          trace_id: 'trace-parallel-c1',
          event_name: 'server_status_update',
        },
        deps,
      )
      expect(latestServerStartupStatusForBinding.value?.comment_id).toBe('C1')
      expect(latestServerStartupStatusForBinding.value?.trace_id).toBe('trace-parallel-c1')
    })

    it('fans out comment-scoped trace_id even when message is empty', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'processing',
          comment_id: 'C2',
          message: '',
          progress: 5,
          trace_id: 'trace-c2-http-ack',
          event_name: 'server_status_update',
        },
        deps,
      )
      expect(latestServerStartupStatusForBinding.value?.comment_id).toBe('C2')
      expect(latestServerStartupStatusForBinding.value?.trace_id).toBe('trace-c2-http-ack')
    })

    it('omits trace_id from binding bus when statusData has none', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'processing',
          comment_id: 'C1',
          message: '[-、task_t1_C1] 正在创建前置资源...',
          progress: 30,
        },
        deps,
      )
      expect(latestServerStartupStatusForBinding.value?.comment_id).toBe('C1')
      expect(latestServerStartupStatusForBinding.value?.trace_id).toBeUndefined()
    })

    it('does not fan out container_heartbeat to binding startup log bus', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'container_heartbeat',
          comment_id: 'C1',
          message: 'heartbeat',
        },
        deps,
      )
      expect(latestServerStartupStatusForBinding.value).toBeNull()
    })

    it('fans out userdata_boot (init_from_task2app.sh) progress to binding bus', async () => {
      const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
      latestServerStartupStatusForBinding.value = null
      updateServerStatus(
        {
          status: 'processing',
          phase: 'userdata_boot',
          comment_id: 'C1',
          message: '[-、task_t1_C1] 安装容器运行时...',
          progress: 35,
          event_name: 'server_status_update',
        },
        deps,
      )
      expect(latestServerStartupStatusForBinding.value?.comment_id).toBe('C1')
      expect(latestServerStartupStatusForBinding.value?.messages?.[0]).toContain('安装容器运行时')
    })
  })

  describe('start-vm submission ack', () => {
    it('treats success + 已提交 as starting, not running', () => {
      updateServerStatus(
        {
          status: 'success',
          message: '自动创建资源并启动虚拟机请求已提交',
          event_id: '862532588237045760',
        },
        deps,
      )

      expect(deps.isServerStarting.value).toBe(true)
      expect(deps.isServerRunning.value).toBe(false)
      expect(deps.statusProgress.value).toBe(5)
    })

    it('sets isServerStarting for starting status from SSE', () => {
      updateServerStatus(
        {
          status: 'starting',
          message: '正在启动服务器...',
          progress: 10,
        },
        deps,
      )

      expect(deps.isServerStarting.value).toBe(true)
      expect(deps.isServerRunning.value).toBe(false)
    })
  })

  describe('container_layer_graph comment_id isolation', () => {
    it('writes A snapshot only and leaves B and non-active page-level snapshot alone', () => {
      const store = createCommentLayerPanelStore()
      store.patch('cmt_b', {
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [], layers_root: '', bootstrap_layer_id: '' },
      })
      deps.layerPanelStore = store
      deps.displayComments = ref([{ id: 'cmt_b', commentKind: 'ai' }])
      deps.activeContainerAgentId = ref('')

      updateServerStatus(
        {
          status: 'container_layer_graph',
          comment_id: 'cmt_a',
          layers: [{ layer_id: 'layer-a', created_at: '2026-01-01T00:00:00Z' }],
          jobs: [],
          layers_root: '/layers-a',
          bootstrap_layer_id: 'layer-a',
        },
        deps,
      )

      expect(store.get('cmt_a').snapshot.layers[0].layer_id).toBe('layer-a')
      expect(store.get('cmt_b').snapshot.layers[0].layer_id).toBe('layer-b')
      expect(deps.layerGraphSnapshot.value.layers).toEqual([])
    })

    it('empty layers for A do not wipe B', () => {
      const store = createCommentLayerPanelStore()
      store.patch('cmt_b', {
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
        execLogTopError: '',
      })
      deps.layerPanelStore = store
      deps.displayComments = ref([{ id: 'cmt_b', commentKind: 'ai' }])

      updateServerStatus(
        {
          status: 'container_layer_graph',
          comment_id: 'cmt_a',
          layers: [],
          jobs: [],
        },
        deps,
      )

      expect(store.get('cmt_b').snapshot.layers[0].layer_id).toBe('layer-b')
      expect(store.get('cmt_b').execLogTopError).toBe('')
    })
  })

  describe('container_layer_changes and job-stream live output isolation', () => {
    it('writes layer_changes to A slot and leaves B unchanged', () => {
      const store = createCommentLayerPanelStore()
      store.patch('cmt_a', {
        snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [{ id: 'job-a', layer_id: 'layer-a' }] },
        layerChangesByLayerId: {},
      })
      store.patch('cmt_b', {
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [{ id: 'job-b', layer_id: 'layer-b' }] },
        layerChangesByLayerId: {
          'layer-b': { layer_id: 'layer-b', changes: [{ path: 'b.txt' }] },
        },
      })
      deps.layerPanelStore = store
      deps.displayComments = ref([{ id: 'cmt_b', commentKind: 'ai' }])
      deps.normalizeLayerChangesPayload = (src) => ({
        layer_id: String(src.layer_id),
        changes: src.changes,
        change_count: src.changes.length,
      })

      updateServerStatus(
        {
          status: 'container_layer_changes',
          comment_id: 'cmt_a',
          layer_id: 'layer-a',
          changes: [{ path: 'a.txt', kind: 'M' }],
        },
        deps,
      )

      expect(store.get('cmt_a').layerChangesByLayerId['layer-a'].changes[0].path).toBe('a.txt')
      expect(store.get('cmt_b').layerChangesByLayerId['layer-b'].changes[0].path).toBe('b.txt')
      expect(store.get('cmt_b').layerChangesByLayerId['layer-a']).toBeUndefined()
      expect(deps.layerChangesByLayerId.value['layer-a']).toBeUndefined()
    })

    it('job-stream chunk for A job does not land in B liveOutputMap', () => {
      const store = createCommentLayerPanelStore()
      store.patch('cmt_a', {
        snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [{ id: 'job-a', layer_id: 'layer-a' }] },
        liveOutputMap: {},
      })
      store.patch('cmt_b', {
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [{ id: 'job-b', layer_id: 'layer-b' }] },
        liveOutputMap: { 'job-b': 'keep B' },
      })
      deps.layerPanelStore = store
      deps.displayComments = ref([{ id: 'cmt_b', commentKind: 'ai' }])

      updateServerStatus(
        {
          status: 'container_job_stream',
          comment_id: 'cmt_a',
          job_id: 'job-a',
          phase: 'chunk',
          message: 'chunk from A',
        },
        deps,
      )

      expect(store.get('cmt_a').liveOutputMap['job-a']).toContain('chunk from A')
      expect(store.get('cmt_b').liveOutputMap['job-b']).toBe('keep B')
      expect(store.get('cmt_b').liveOutputMap['job-a']).toBeUndefined()
    })

    it('layer_changes without comment_id still routes by owning layer', () => {
      const store = createCommentLayerPanelStore()
      store.patch('cmt_a', {
        snapshot: { layers: [{ layer_id: 'layer-a' }], jobs: [] },
        layerChangesByLayerId: {},
      })
      store.patch('cmt_b', {
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
        layerChangesByLayerId: {},
      })
      deps.layerPanelStore = store
      deps.displayComments = ref([{ id: 'cmt_b', commentKind: 'ai' }])
      deps.normalizeLayerChangesPayload = (src) => ({
        layer_id: String(src.layer_id),
        changes: src.changes || [],
      })

      updateServerStatus(
        {
          status: 'container_layer_changes',
          layer_id: 'layer-a',
          changes: [{ path: 'owned-by-a.txt', kind: 'A' }],
        },
        deps,
      )

      expect(store.get('cmt_a').layerChangesByLayerId['layer-a'].changes[0].path).toBe('owned-by-a.txt')
      expect(store.get('cmt_b').layerChangesByLayerId['layer-a']).toBeUndefined()
    })
  })
})
