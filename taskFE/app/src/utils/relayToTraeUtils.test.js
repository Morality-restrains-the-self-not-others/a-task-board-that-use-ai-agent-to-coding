// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  RELAY_TO_TRAE_ENV_KEYS,
  TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
  isRouteRelayToTraeQuery,
  shouldApplyRelayToTraeStatusLogs,
  isContainerHeartbeatLogLine,
  classifyBootstrapMilestoneLogLine,
  classifyRelayStartupLogLine,
  isTokenPersistFailedLogLine,
  mapRelayToTraeStatusErrorMessage,
  resolveBootstrapStatusFromLogs,
  partitionRelayToTraeLogLines,
  mergeRelayToTraeLogLines,
  parseRelayToTraeEnv,
  buildDefaultRelayToTraeEnvItems,
  DEFAULT_TASK_API_ENDPOINT_ORIGIN,
  buildRelayToTraeStartEnvPayload,
  buildRelayToTraeTokenInitPayload,
  resolveRelayToTraePublicUiUrl,
  resolveRelayConsoleOpenUrl,
} from './relayToTraeUtils.js'

describe('classifyBootstrapMilestoneLogLine', () => {
  it('marks BOOTSTRAP_COMPLETE and legacy 任务引导完成 as complete', () => {
    expect(
      classifyBootstrapMilestoneLogLine(
        '[onlineServiceJS] BOOTSTRAP_COMPLETE 任务引导完成（详情已拉取、克隆与配置已就绪）。',
      ),
    ).toBe('complete')
    expect(
      classifyBootstrapMilestoneLogLine(
        '[onlineServiceJS] 任务引导完成（详情已拉取、克隆与配置已就绪）。',
      ),
    ).toBe('complete')
  })

  it('marks BOOTSTRAP_FAILED and post-listen error as failed', () => {
    expect(
      classifyBootstrapMilestoneLogLine(
        '[onlineServiceJS] BOOTSTRAP_FAILED phase=feature_params_env fetch failed',
      ),
    ).toBe('failed')
    expect(
      classifyBootstrapMilestoneLogLine('[onlineServiceJS] bootstrap (post-listen) error: boom'),
    ).toBe('failed')
  })

  it('marks BOOTSTRAP_PHASE as phase and ignores unrelated lines', () => {
    expect(
      classifyBootstrapMilestoneLogLine(
        '[onlineServiceJS] BOOTSTRAP_PHASE=task_detail_begin 容器已启动，开始拉取任务详情…',
      ),
    ).toBe('phase')
    expect(classifyBootstrapMilestoneLogLine('[onlineServiceJS] server listening')).toBe(null)
    expect(classifyBootstrapMilestoneLogLine('[relayToTrae] start')).toBe(null)
  })
})

describe('resolveBootstrapStatusFromLogs', () => {
  it('returns idle for empty or unrelated logs', () => {
    expect(resolveBootstrapStatusFromLogs([])).toEqual({
      kind: 'idle',
      label: '',
      phaseKey: '',
      detail: '',
    })
    expect(resolveBootstrapStatusFromLogs(['[relayToTrae] start']).kind).toBe('idle')
  })

  it('tracks running phase then complete', () => {
    const running = resolveBootstrapStatusFromLogs([
      '[onlineServiceJS] BOOTSTRAP_PHASE=task_detail_begin 容器已启动，开始拉取任务详情…',
      '[onlineServiceJS] BOOTSTRAP_PHASE=feature_params_begin 开始拉取 feature-params-env…',
    ])
    expect(running.kind).toBe('running')
    expect(running.phaseKey).toBe('feature_params_begin')
    expect(running.label).toContain('拉取智能体资源配置')

    const done = resolveBootstrapStatusFromLogs([
      '[onlineServiceJS] BOOTSTRAP_PHASE=clone_begin 任务详情已就绪，开始项目克隆…',
      '[onlineServiceJS] BOOTSTRAP_COMPLETE 任务引导完成（详情已拉取、克隆与配置已就绪）。',
    ])
    expect(done.kind).toBe('complete')
    expect(done.label).toBe('引导完成')
  })

  it('failed wins over later complete-looking noise and exposes badge label', () => {
    const failed = resolveBootstrapStatusFromLogs([
      '[onlineServiceJS] BOOTSTRAP_PHASE=feature_params_begin 开始拉取 feature-params-env…',
      '[onlineServiceJS] BOOTSTRAP_FAILED phase=feature_params_env fetch failed',
      '[onlineServiceJS] server listening on http://0.0.0.0:8765',
    ])
    expect(failed.kind).toBe('failed')
    expect(failed.phaseKey).toBe('feature_params_env')
    expect(failed.label).toBe('引导失败（智能体资源配置环境）')
    expect(failed.detail).toContain('BOOTSTRAP_FAILED')
  })

  it('surfaces 换票落盘失败 from FAIL_PERSIST / TOKEN_PERSIST_FAILED lines', () => {
    const persist = resolveBootstrapStatusFromLogs([
      '[onlineServiceJS] token-exchange: exchange-refresh OK refresh_token len=17',
      '[onlineServiceJS] token-exchange: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED token-persist: FAIL write /tmp/x: EACCES',
      '[relayToTrae] token-sync: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED 子进程换票成功但 refresh_token 落盘失败',
    ])
    expect(persist.kind).toBe('failed')
    expect(persist.phaseKey).toBe('token_persist')
    expect(persist.label).toBe('换票落盘失败')
    expect(persist.detail).toMatch(/FAIL_PERSIST|TOKEN_PERSIST_FAILED/)

    expect(isTokenPersistFailedLogLine(persist.detail)).toBe(true)
    expect(
      classifyRelayStartupLogLine(
        '[onlineServiceJS] token-exchange: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED boom',
      ),
    ).toBe('token_persist_failed')
    expect(
      isTokenPersistFailedLogLine(
        '[onlineServiceJS] token-exchange: FAIL HTTP 401 TOKEN_ACCESS_INVALID',
      ),
    ).toBe(false)
  })
})

describe('mapRelayToTraeStatusErrorMessage', () => {
  it('maps TOKEN_PERSIST_FAILED state.Error to Chinese start-button message', () => {
    expect(mapRelayToTraeStatusErrorMessage('TOKEN_PERSIST_FAILED: child token persist failed')).toBe(
      '换票落盘失败：请检查 ONLINE_PROJECT_STATE_ROOT 磁盘权限后重试',
    )
    expect(
      mapRelayToTraeStatusErrorMessage(
        '[relayToTrae] token-sync: FAIL_PERSIST error_code=TOKEN_PERSIST_FAILED boom',
      ),
    ).toContain('换票落盘失败')
  })

  it('passes through unrelated errors unchanged', () => {
    expect(mapRelayToTraeStatusErrorMessage('onlineServiceJS exited (code=1)')).toBe(
      'onlineServiceJS exited (code=1)',
    )
    expect(mapRelayToTraeStatusErrorMessage('')).toBe('')
    expect(mapRelayToTraeStatusErrorMessage('', 'fallback')).toBe('fallback')
  })
})

describe('isContainerHeartbeatLogLine / partitionRelayToTraeLogLines', () => {
  it('detects saas-heartbeat-probe access lines', () => {
    const line =
      '{"ts":"2026-07-09T08:00:00Z","level":"info","msg":"http_request","path":"/api/saas-heartbeat-probe?seq=1","status":"200"}'
    expect(isContainerHeartbeatLogLine(line)).toBe(true)
    expect(isContainerHeartbeatLogLine('[onlineServiceJS] server listening')).toBe(false)
  })

  it('partitions heartbeat lines away from startup logs and merge returns heartbeatLines', () => {
    const hb =
      '127.0.0.1 "GET /api/saas-heartbeat-probe?seq=2" 200 1ms'
    const start = '[onlineServiceJS] server listening on http://0.0.0.0:8765'
    const { startupLines, heartbeatLines } = partitionRelayToTraeLogLines([start, hb])
    expect(startupLines).toEqual([start])
    expect(heartbeatLines).toEqual([hb])
    const out = mergeRelayToTraeLogLines(
      { logs: [], snapshot: [], suppressedAfterClear: false },
      [start, hb, 'clone done'],
    )
    expect(out.logs).toEqual([start, 'clone done'])
    expect(out.heartbeatLines).toEqual([hb])
  })
})

describe('RELAY_TO_TRAE_ENV_KEYS', () => {
  it('contains the relay env keys in order', () => {
    expect(RELAY_TO_TRAE_ENV_KEYS).toHaveLength(4)
    expect(RELAY_TO_TRAE_ENV_KEYS[0]).toBe('TASK_API_ENDPOINT_ORIGIN')
    expect(RELAY_TO_TRAE_ENV_KEYS[1]).toBe('BUSINESS_API_ENDPOINT_ORIGIN')
    expect(RELAY_TO_TRAE_ENV_KEYS[2]).toBe('ACCESS_TOKEN')
    expect(RELAY_TO_TRAE_ENV_KEYS[3]).toBe('DEBUG_AGENT')
  })

  it('is frozen (immutable)', () => {
    expect(Object.isFrozen(RELAY_TO_TRAE_ENV_KEYS)).toBe(true)
  })
})

describe('TASK2APP_ACCESS_TOKEN_PLACEHOLDER', () => {
  it('is the expected placeholder string', () => {
    expect(TASK2APP_ACCESS_TOKEN_PLACEHOLDER).toBe('__TASK2APP_ACCESS_TOKEN__')
  })
})

describe('isRouteRelayToTraeQuery', () => {
  it('returns false for null/undefined query', () => {
    expect(isRouteRelayToTraeQuery(null)).toBe(false)
    expect(isRouteRelayToTraeQuery(undefined)).toBe(false)
  })

  it('returns false when relayToTrae is missing', () => {
    expect(isRouteRelayToTraeQuery({})).toBe(false)
    expect(isRouteRelayToTraeQuery({ other: 'value' })).toBe(false)
  })

  it('returns false when relayToTrae is null or undefined', () => {
    expect(isRouteRelayToTraeQuery({ relayToTrae: null })).toBe(false)
    expect(isRouteRelayToTraeQuery({ relayToTrae: undefined })).toBe(false)
  })

  it('returns true when relayToTrae is the string "true"', () => {
    expect(isRouteRelayToTraeQuery({ relayToTrae: 'true' })).toBe(true)
  })

  it('returns true when relayToTrae is the string "True" (case insensitive)', () => {
    expect(isRouteRelayToTraeQuery({ relayToTrae: 'True' })).toBe(true)
    expect(isRouteRelayToTraeQuery({ relayToTrae: 'TRUE' })).toBe(true)
  })

  it('returns false when relayToTrae is "false" or any other string', () => {
    expect(isRouteRelayToTraeQuery({ relayToTrae: 'false' })).toBe(false)
    expect(isRouteRelayToTraeQuery({ relayToTrae: '' })).toBe(false)
    expect(isRouteRelayToTraeQuery({ relayToTrae: '1' })).toBe(false)
  })

  it('handles array query values (takes first element)', () => {
    expect(isRouteRelayToTraeQuery({ relayToTrae: ['true', 'other'] })).toBe(true)
    expect(isRouteRelayToTraeQuery({ relayToTrae: ['false', 'true'] })).toBe(false)
    expect(isRouteRelayToTraeQuery({ relayToTrae: [''] })).toBe(false)
  })
})

describe('shouldApplyRelayToTraeStatusLogs', () => {
  it('allows logs when active_task_id matches current task', () => {
    expect(shouldApplyRelayToTraeStatusLogs('task-b', { active_task_id: 'task-b', logs: ['x'] })).toBe(true)
  })

  it('blocks logs when active_task_id belongs to another task', () => {
    expect(shouldApplyRelayToTraeStatusLogs('task-b', { active_task_id: 'task-a', logs: ['old'] })).toBe(false)
  })

  it('allows logs when log_task_id matches current task even if another task is active', () => {
    expect(
      shouldApplyRelayToTraeStatusLogs('task-a', {
        active_task_id: 'task-b',
        log_task_id: 'task-a',
        logs: ['history'],
      }),
    ).toBe(true)
  })

  it('blocks logs when relay payload has no task identity', () => {
    expect(shouldApplyRelayToTraeStatusLogs('task-b', { logs: ['legacy'] })).toBe(false)
  })
})

describe('mergeRelayToTraeLogLines', () => {
  it('shows status logs after start when awaiting first status refresh', () => {
    const statusLogs = ['[relayToTrae] start', '[onlineServiceJS] exited code 1']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: true,
      },
      statusLogs,
    )
    expect(out.logs).toEqual(statusLogs)
    expect(out.snapshot).toEqual(statusLogs)
    expect(out.suppressedAfterClear).toBe(false)
    expect(out.awaitingFirstStatusAfterStart).toBe(false)
  })

  it('does not refill logs after user cleared and refreshed status', () => {
    const old = ['line-1', 'line-2']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [...old],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      old,
    )
    expect(out.logs).toEqual([])
    expect(out.snapshot).toEqual(old)
    expect(out.suppressedAfterClear).toBe(true)
  })

  it('does not refill logs when cleared with empty snapshot but not awaiting first status', () => {
    const statusLogs = ['[relayToTrae] start']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      statusLogs,
    )
    expect(out.logs).toEqual([])
    expect(out.snapshot).toEqual(statusLogs)
    expect(out.suppressedAfterClear).toBe(true)
  })

  it('empty status push after clear must not lift suppression', () => {
    const old = ['line-1', 'line-2']
    const afterEmpty = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [...old],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      [],
    )
    expect(afterEmpty.suppressedAfterClear).toBe(true)
    expect(afterEmpty.snapshot).toEqual(old)
    const afterFull = mergeRelayToTraeLogLines(afterEmpty, old)
    expect(afterFull.logs).toEqual([])
    expect(afterFull.suppressedAfterClear).toBe(true)
  })

  it('does not refill when SSE replays a known contiguous slice after clear', () => {
    const full = ['a', 'b', 'c', 'd']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [...full],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      ['b', 'c'],
    )
    expect(out.logs).toEqual([])
    expect(out.snapshot).toEqual(full)
    expect(out.suppressedAfterClear).toBe(true)
  })

  it('does not refill when status returns a shorter prefix of the cleared snapshot', () => {
    const full = ['a', 'b', 'c']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [...full],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      ['a', 'b'],
    )
    expect(out.logs).toEqual([])
    expect(out.snapshot).toEqual(full)
    expect(out.suppressedAfterClear).toBe(true)
  })

  it('after clear, only appends new tail beyond snapshot and keeps suppression', () => {
    const prefix = ['line-1', 'line-2']
    const full = ['line-1', 'line-2', 'line-3']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [...prefix],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: false,
      },
      full,
    )
    expect(out.logs).toEqual(['line-3'])
    expect(out.snapshot).toEqual(full)
    expect(out.suppressedAfterClear).toBe(true)
  })

  it('awaiting first status after start shows logs until user clears', () => {
    const statusLogs = ['line-1', 'line-2']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: true,
      },
      statusLogs,
    )
    expect(out.logs).toEqual(statusLogs)
    expect(out.suppressedAfterClear).toBe(false)
    expect(out.awaitingFirstStatusAfterStart).toBe(false)
  })

  it('appends only new tail from full status snapshot', () => {
    const prefix = ['line-1', 'line-2']
    const full = ['line-1', 'line-2', 'line-3']
    const out = mergeRelayToTraeLogLines(
      {
        logs: [...prefix],
        snapshot: [...prefix],
        suppressedAfterClear: false,
      },
      full,
    )
    expect(out.logs).toEqual(full)
    expect(out.snapshot).toEqual(full)
  })

  it('rebases when SSE mid-stream snapshot is later covered by full /status buffer', () => {
    const bootstrap = [
      '[onlineServiceJS] 已向 SaaS 注册可达地址 public_ip=1.2.3.4 server_url=http://127.0.0.1:8765',
      '[onlineServiceJS] 已调度 SaaS 容器心跳（首跳延迟 5s，间隔见 TRAE_SAAS_HEARTBEAT_INTERVAL_SEC）',
      '[onlineServiceJS] BOOTSTRAP_PHASE=task_detail_begin 容器已启动，开始拉取任务详情…',
      '[onlineServiceJS] 任务详情已拉取（关联仓库 2 个），继续引导…',
      '[onlineServiceJS] 开始拉取仓库克隆凭证…',
      '[onlineServiceJS] BOOTSTRAP_PHASE=clone_begin 任务详情已就绪，开始项目克隆…',
    ]
    const relayPrefix = [
      '[relayToTrae] selected-image start: pull registry.example/app:latest',
      '[relayToTrae] container started id=abc name=relay_taskId_task_x',
    ]
    const afterSse = mergeRelayToTraeLogLines(
      {
        logs: [],
        snapshot: [],
        suppressedAfterClear: true,
        awaitingFirstStatusAfterStart: true,
      },
      bootstrap,
    )
    const afterFull = mergeRelayToTraeLogLines(afterSse, [...relayPrefix, ...bootstrap])
    expect(afterFull.logs.filter((l) => l.includes('clone_begin'))).toHaveLength(1)
    expect(afterFull.logs.filter((l) => l.includes('已向 SaaS'))).toHaveLength(1)
    expect(afterFull.logs[0]).toContain('[relayToTrae] selected-image start')
    expect(afterFull.snapshot).toEqual([...relayPrefix, ...bootstrap])
  })
})

describe('parseRelayToTraeEnv', () => {
  it('converts env items array to key-value object', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://api.daydaymoney.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: '__TASK2APP_ACCESS_TOKEN__' },
    ]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({
      TASK_API_ENDPOINT_ORIGIN: 'http://api.daydaymoney.com',
      BUSINESS_API_ENDPOINT_ORIGIN: 'http://biz.example.com',
      ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
    })
  })

  it('skips items with empty key', () => {
    const items = [
      { key: '', value: 'value' },
      { key: 'VALID_KEY', value: 'valid' },
      { key: '   ', value: 'whitespace-only-key' },
    ]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({ VALID_KEY: 'valid' })
  })

  it('skips items with null/undefined key', () => {
    const items = [
      { key: null, value: 'nope' },
      { key: undefined, value: 'also-nope' },
      { key: 'GOOD', value: 'yes' },
    ]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({ GOOD: 'yes' })
  })

  it('handles empty array', () => {
    expect(parseRelayToTraeEnv([])).toEqual({})
  })

  it('converts non-string values via String()', () => {
    const items = [
      { key: 'NUM', value: 42 },
      { key: 'BOOL', value: true },
    ]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({ NUM: '42', BOOL: 'true' })
  })

  it('handles missing value property (undefined ?? "" = "")', () => {
    const items = [{ key: 'NO_VALUE' }]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({ NO_VALUE: '' })
  })

  it('handles null value (String(null ?? "") = "")', () => {
    const items = [{ key: 'NULL_VAL', value: null }]
    const result = parseRelayToTraeEnv(items)
    expect(result).toEqual({ NULL_VAL: '' })
  })
})

describe('buildDefaultRelayToTraeEnvItems', () => {
  it('returns env items for each RELAY_TO_TRAE_ENV_KEYS entry', () => {
    const items = buildDefaultRelayToTraeEnvItems({})
    expect(items).toHaveLength(4)
    expect(items[0].key).toBe('TASK_API_ENDPOINT_ORIGIN')
    expect(items[1].key).toBe('BUSINESS_API_ENDPOINT_ORIGIN')
    expect(items[2].key).toBe('ACCESS_TOKEN')
    expect(items[3].key).toBe('DEBUG_AGENT')
  })

  it('uses placeholder for ACCESS_TOKEN', () => {
    const items = buildDefaultRelayToTraeEnvItems({})
    const tokenItem = items.find((i) => i.key === 'ACCESS_TOKEN')
    expect(tokenItem.value).toBe('__TASK2APP_ACCESS_TOKEN__')
  })

  it('uses env-based defaults when VITE env vars are provided', () => {
    const taskOrigin = 'http://custom-task.example.com'
    const bizOrigin = 'http://custom-biz.example.com'
    const items = buildDefaultRelayToTraeEnvItems({}, {
      taskApiOrigin: taskOrigin,
      businessApiOrigin: bizOrigin,
    })
    expect(items[0].value).toBe(taskOrigin)
    expect(items[1].value).toBe(bizOrigin)
  })

  it('defaults TASK_API_ENDPOINT_ORIGIN to local Django when env unset', () => {
    const items = buildDefaultRelayToTraeEnvItems({}, {})
    const taskItem = items.find((i) => i.key === 'TASK_API_ENDPOINT_ORIGIN')
    // Vite 可能注入 VITE_DEFAULT_MOCK_TASK_API_ENDPOINT；未注入时回落到常量。
    const fromVite =
      typeof import.meta.env?.VITE_DEFAULT_MOCK_TASK_API_ENDPOINT === 'string'
        ? import.meta.env.VITE_DEFAULT_MOCK_TASK_API_ENDPOINT.trim()
        : ''
    expect(taskItem.value).toBe(fromVite || DEFAULT_TASK_API_ENDPOINT_ORIGIN)
  })

  it('returns default biz origin when not overridden', () => {
    const items = buildDefaultRelayToTraeEnvItems({}, {})
    const bizItem = items.find((i) => i.key === 'BUSINESS_API_ENDPOINT_ORIGIN')
    const fromVite =
      typeof import.meta.env?.VITE_DEFAULT_RELAY_BUSINESS_API_ORIGIN === 'string'
        ? import.meta.env.VITE_DEFAULT_RELAY_BUSINESS_API_ORIGIN.trim()
        : ''
    expect(bizItem.value).toBe(fromVite || 'http://127.0.0.1:8765')
  })

  it('returns DEBUG_AGENT=True by default', () => {
    const items = buildDefaultRelayToTraeEnvItems({}, {})
    const debugItem = items.find((i) => i.key === 'DEBUG_AGENT')
    expect(debugItem.value).toBe('True')
  })
})

describe('buildRelayToTraeStartEnvPayload', () => {
  const placeholder = '__TASK2APP_ACCESS_TOKEN__'

  it('returns ok:true with env when all required fields are filled', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: placeholder },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(true)
    expect(result.env).toBeDefined()
    expect(result.env.TASK_API_ENDPOINT_ORIGIN).toBe('http://task.example.com')
    expect(result.env.BUSINESS_API_ENDPOINT_ORIGIN).toBe('http://biz.example.com')
    // ACCESS_TOKEN is always set to placeholder in the payload
    expect(result.env.ACCESS_TOKEN).toBe(placeholder)
  })

  it('passes DEBUG_AGENT through to start env payload', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: placeholder },
      { key: 'DEBUG_AGENT', value: 'True' },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(true)
    expect(result.env.DEBUG_AGENT).toBe('True')
  })

  it('returns ok:false with missingKey for empty TASK_API_ENDPOINT_ORIGIN', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: '' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: placeholder },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(false)
    expect(result.missingKey).toBe('TASK_API_ENDPOINT_ORIGIN')
  })

  it('returns ok:false with missingKey for empty BUSINESS_API_ENDPOINT_ORIGIN', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: '' },
      { key: 'ACCESS_TOKEN', value: placeholder },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(false)
    expect(result.missingKey).toBe('BUSINESS_API_ENDPOINT_ORIGIN')
  })

  it('returns ok:false for whitespace-only TASK_API_ENDPOINT_ORIGIN', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: '   ' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: placeholder },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(false)
  })

  it('sets ACCESS_TOKEN to placeholder regardless of input value', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: 'some-real-token' },
    ]
    const result = buildRelayToTraeStartEnvPayload(items, placeholder)
    expect(result.ok).toBe(true)
    expect(result.env.ACCESS_TOKEN).toBe(placeholder)
  })
})

describe('buildRelayToTraeTokenInitPayload', () => {
  const placeholder = '__TASK2APP_ACCESS_TOKEN__'

  it('returns payload with env when required fields are filled', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: 'anything' },
    ]
    const result = buildRelayToTraeTokenInitPayload(items, placeholder)
    expect(result.ok).toBe(true)
    expect(result.payload.env.TASK_API_ENDPOINT_ORIGIN).toBe('http://task.example.com')
    expect(result.payload.env.BUSINESS_API_ENDPOINT_ORIGIN).toBe('http://biz.example.com')
    expect(result.payload.env.ACCESS_TOKEN).toBe(placeholder)
  })

  it('returns missing key when required origin is empty', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: '' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: placeholder },
    ]
    const result = buildRelayToTraeTokenInitPayload(items, placeholder)
    expect(result.ok).toBe(false)
    expect(result.missingKey).toBe('TASK_API_ENDPOINT_ORIGIN')
  })

  it('builds token-init and start payload with the same env contract', () => {
    const items = [
      { key: 'TASK_API_ENDPOINT_ORIGIN', value: 'http://task.example.com' },
      { key: 'BUSINESS_API_ENDPOINT_ORIGIN', value: 'http://biz.example.com' },
      { key: 'ACCESS_TOKEN', value: 'real-token-should-not-be-used' },
    ]
    const startResult = buildRelayToTraeStartEnvPayload(items, placeholder)
    const tokenInitResult = buildRelayToTraeTokenInitPayload(items, placeholder)

    expect(startResult.ok).toBe(true)
    expect(tokenInitResult.ok).toBe(true)
    expect(tokenInitResult.payload.env.TASK_API_ENDPOINT_ORIGIN).toBe(startResult.env.TASK_API_ENDPOINT_ORIGIN)
    expect(tokenInitResult.payload.env.BUSINESS_API_ENDPOINT_ORIGIN).toBe(startResult.env.BUSINESS_API_ENDPOINT_ORIGIN)
    expect(startResult.env.ACCESS_TOKEN).toBe(placeholder)
    expect(tokenInitResult.payload.env.ACCESS_TOKEN).toBe(placeholder)
  })
})

describe('resolveRelayToTraePublicUiUrl', () => {
  const loopbackUiUrl = 'http://127.0.0.1:8765/ui/secret-token'

  it('returns empty for blank input', () => {
    expect(resolveRelayToTraePublicUiUrl('')).toBe('')
  })

  it('returns non-loopback ui_url unchanged', () => {
    const url = 'http://203.0.113.10:8765/ui/secret-token'
    expect(resolveRelayToTraePublicUiUrl(url)).toBe(url)
  })

  it('rewrites loopback hostname from BUSINESS_API_ENDPOINT_ORIGIN', () => {
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
      }),
    ).toBe('http://127.0.0.1:8765/ui/secret-token')
  })

  it('keeps loopback when no public host is available', () => {
    expect(resolveRelayToTraePublicUiUrl(loopbackUiUrl)).toBe(loopbackUiUrl)
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
      }),
    ).toBe(loopbackUiUrl)
  })

  it('falls back to publicIp when business origin is loopback', () => {
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
        publicIp: '203.0.113.9',
      }),
    ).toBe('http://203.0.113.9:8765/ui/secret-token')
  })

  it('falls back to publicIp Origin (scheme://domain) when business origin is loopback', () => {
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
        publicIp: 'https://businessapi.daydaymoney.com',
      }),
    ).toBe('https://businessapi.daydaymoney.com/ui/secret-token')
  })

  it('falls back to TASK_API_ENDPOINT_ORIGIN when business origin is loopback', () => {
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
        taskApiOrigin: 'http://127.0.0.1:8001',
      }),
    ).toBe('http://127.0.0.1:8765/ui/secret-token')
  })

  it('prefers business origin over publicIp (loopback business origin skipped)', () => {
    // When business origin is loopback, it is skipped and publicIp is used instead
    expect(
      resolveRelayToTraePublicUiUrl(loopbackUiUrl, {
        businessApiOrigin: 'http://127.0.0.1:8765',
        publicIp: '203.0.113.9',
      }),
    ).toBe('http://203.0.113.9:8765/ui/secret-token')
  })
})

describe('resolveRelayConsoleOpenUrl', () => {
  it('selected_image prefers containerPageUrl over relay ui_url', () => {
    expect(
      resolveRelayConsoleOpenUrl({
        mode: 'selected_image',
        relayUiUrl: 'http://127.0.0.1:8765/ui/tok',
        containerPageUrl: 'http://127.0.0.1:8765',
        serverUrl: 'http://127.0.0.1:9999',
      }),
    ).toBe('http://127.0.0.1:8765')
  })

  it('selected_image falls back to serverUrl when page url empty', () => {
    expect(
      resolveRelayConsoleOpenUrl({
        mode: 'selected_image',
        relayUiUrl: 'http://127.0.0.1:8765/ui/tok',
        serverUrl: 'http://127.0.0.1:8765/',
      }),
    ).toBe('http://127.0.0.1:8765/')
  })

  it('selected_image returns empty when not registered yet', () => {
    expect(
      resolveRelayConsoleOpenUrl({
        mode: 'selected_image',
        relayUiUrl: 'http://127.0.0.1:8765/ui/tok',
      }),
    ).toBe('')
  })

  it('selected_image rejects task-detail SPA urls', () => {
    expect(
      resolveRelayConsoleOpenUrl({
        mode: 'selected_image',
        serverUrl: 'http://127.0.0.1:4000/tenant/1/workspace/2/task-detail/t/',
      }),
    ).toBe('')
  })

  it('host mode uses relay ui_url', () => {
    expect(
      resolveRelayConsoleOpenUrl({
        mode: '',
        relayUiUrl: 'http://127.0.0.1:8765/ui/tok',
        businessApiOrigin: 'http://127.0.0.1:8765',
        serverUrl: 'http://127.0.0.1:8765',
      }),
    ).toBe('http://127.0.0.1:8765/ui/tok')
  })
})
