import { describe, expect, it } from 'vitest'
import {
  collectServerStartupLogMessages,
  filterTaskStatusLogsForBinding,
  mergeBindingAndServerStartupLogs,
  parseBindingLogDate,
  resolveServerStartupBindingCommentId,
  startupLogDedupeKey,
} from './bindingServerStartupLogs.js'

describe('resolveServerStartupBindingCommentId', () => {
  it('returns comment_id for server scheduling statuses', () => {
    expect(
      resolveServerStartupBindingCommentId({
        comment_id: 'C1',
        status: 'processing',
        message: '正在准备启动服务器...',
      }),
    ).toBe('C1')
  })

  it('ignores container heartbeat and other non-scheduling events', () => {
    expect(
      resolveServerStartupBindingCommentId({
        comment_id: 'C1',
        status: 'container_heartbeat',
      }),
    ).toBe('')
    expect(
      resolveServerStartupBindingCommentId({
        comment_id: 'C1',
        status: 'runtime_hydrate',
      }),
    ).toBe('')
    expect(resolveServerStartupBindingCommentId({ status: 'processing' })).toBe('')
  })

  it('routes userdata_boot (UserData / init_from_task2app.sh) progress to binding', () => {
    expect(
      resolveServerStartupBindingCommentId({
        comment_id: 'C1',
        status: 'processing',
        phase: 'userdata_boot',
        message: '安装容器运行时...',
      }),
    ).toBe('C1')
  })
})

describe('collectServerStartupLogMessages', () => {
  it('collects display message and sdk_call detail lines', () => {
    const label = '[-、task_t1_C1]'
    const msgs = collectServerStartupLogMessages(
      {
        status: 'sdk_call',
        message: `${label} 云厂商SDK调用信息`,
        log_label: label,
        sdk_call: { method: 'RunInstances', request_ids: ['r1'] },
      },
      `${label} 云厂商SDK调用信息`,
    )
    expect(msgs).toEqual([
      `${label} 云厂商SDK调用信息`,
      `${label} SDK调用: RunInstances`,
      `${label} Request IDs: r1`,
    ])
  })

  it('skips blank messages', () => {
    expect(collectServerStartupLogMessages({ status: 'processing', message: '  ' }, ' ')).toEqual([])
  })
})

describe('filterTaskStatusLogsForBinding + mergeBindingAndServerStartupLogs', () => {
  it('filters task logs by container name and merges without duplicating binding lines', () => {
    const binding = [
      '[13:29:58] 容器调度排队中',
      '[13:29:58] 正在启动容器实例',
      '[13:29:58] 容器实例已分配，等待服务就绪',
    ]
    const task = [
      '[13:29:50] [-、task_t1_C1] 检测到自动创建资源，正在准备前置资源创建任务...',
      '[13:29:55] [-、task_t1_C1] 正在自动创建前置资源并启动服务器...',
      '[13:29:50] [-、task_t1_C2] 其他评论的服务器调度',
      '[13:29:58] 正在启动容器实例',
    ]
    const filtered = filterTaskStatusLogsForBinding(task, {
      commentId: 'C1',
      containerName: 'task_t1_C1',
    })
    expect(filtered).toHaveLength(2)
    expect(filtered[0]).toContain('准备前置资源')
    const merged = mergeBindingAndServerStartupLogs(binding, filtered)
    expect(merged.some((l) => l.includes('准备前置资源'))).toBe(true)
    expect(merged.some((l) => l.includes('其他评论'))).toBe(false)
    // 同正文「正在启动容器实例」不因任务级行重复追加
    expect(merged.filter((l) => l.includes('正在启动容器实例'))).toHaveLength(1)
  })

  it('matches UserData task2app-container label and sole-active cloud/UserData lines', () => {
    const task = [
      '[18:42:00] [i-abc、task2app-container] 开始容器初始化',
      '[18:44:51] [i-abc、task2app-container] 拉取容器镜像: registry.example/img:latest',
      '[18:41:24] 正在准备启动aliyun服务器...',
      '[18:41:32] [-、other] 其他无关',
    ]
    const byDockerName = filterTaskStatusLogsForBinding(task, {
      commentId: 'C1',
      containerName: 'task_t1_C1',
    })
    expect(byDockerName.some((l) => l.includes('开始容器初始化'))).toBe(true)
    expect(byDockerName.some((l) => l.includes('拉取容器镜像'))).toBe(true)

    const sole = filterTaskStatusLogsForBinding(task, {
      commentId: 'C1',
      soleActiveBinding: true,
    })
    expect(sole.some((l) => l.includes('准备启动aliyun'))).toBe(true)
    expect(sole.some((l) => l.includes('开始容器初始化'))).toBe(true)
    expect(sole.some((l) => l.includes('其他无关'))).toBe(false)
  })

  it('does not ingest unlabeled stop-vm lines into the sole active binding', () => {
    const task = [
      '[18:40:59] [i-new、task_t1_C1] 启动容器...',
      '[18:41:01] [i-new、task_t1_C1] 容器启动成功',
      '[18:37:16] 正在准备停止aliyun服务器...',
      '[18:37:18] 正在调用aliyunAPI停止服务器...',
      '[18:37:22] 停止虚拟机成功',
    ]
    const sole = filterTaskStatusLogsForBinding(task, {
      commentId: 'C1',
      soleActiveBinding: true,
    })
    expect(sole.some((l) => l.includes('启动容器'))).toBe(true)
    expect(sole.some((l) => l.includes('容器启动成功'))).toBe(true)
    expect(sole.some((l) => l.includes('准备停止aliyun'))).toBe(false)
    expect(sole.some((l) => l.includes('调用aliyunAPI停止'))).toBe(false)
    expect(sole.some((l) => l.includes('停止虚拟机成功'))).toBe(false)
  })

  it('dedupes the same UserData step when clocks differ by 8h and only one line has trace_id', () => {
    const binding = [
      '[23:30:54] [i-abc、task_t1_c1] 容器运行时安装完成',
    ]
    const server = [
      '[15:30:53] [i-abc、task_t1_c1] 容器运行时安装完成 trace_id=bb1157e1b936e3d8eb7e7b05',
    ]
    const merged = mergeBindingAndServerStartupLogs(binding, server)
    expect(merged).toHaveLength(1)
    expect(merged[0]).toContain('trace_id=bb1157e1b936e3d8eb7e7b05')
    expect(merged.filter((l) => l.includes('容器运行时安装完成'))).toHaveLength(1)
  })

  it('keeps distinct UserData steps', () => {
    const merged = mergeBindingAndServerStartupLogs(
      ['[23:30:54] [i-abc、c1] 容器运行时安装完成'],
      ['[23:30:59] [i-abc、c1] 容器运行时服务已就绪'],
    )
    expect(merged).toHaveLength(2)
  })
})

describe('startupLogDedupeKey + parseBindingLogDate', () => {
  it('strips clock prefix and trailing trace_id', () => {
    expect(startupLogDedupeKey('[15:30:53] [i-abc、c1] 容器运行时安装完成 trace_id=bb1157e1')).toBe(
      '[i-abc、c1] 容器运行时安装完成',
    )
    expect(startupLogDedupeKey('[23:30:54] [i-abc、c1] 容器运行时安装完成')).toBe(
      '[i-abc、c1] 容器运行时安装完成',
    )
  })

  it('treats naive mysql utc wall clock as the same instant as RFC3339 Z', () => {
    expect(parseBindingLogDate('2026-08-13 15:30:59').toISOString()).toBe('2026-08-13T15:30:59.000Z')
    expect(parseBindingLogDate('2026-08-13T15:30:59Z').toISOString()).toBe('2026-08-13T15:30:59.000Z')
  })
})
