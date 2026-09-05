// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import {
  applyStartVmHttpResult,
  buildStartVmAcceptedStatusUpdate,
  isAsyncStartVmSubmissionAck,
} from './startVmHttpResult.js'

describe('startVmHttpResult', () => {
  describe('isAsyncStartVmSubmissionAck', () => {
    it('returns true when message contains 已提交', () => {
      expect(
        isAsyncStartVmSubmissionAck({
          status: 'success',
          message: '自动创建资源并启动虚拟机请求已提交',
          event_id: '862532588237045760',
        }),
      ).toBe(true)
    })

    it('returns true when event_id present without vm_info', () => {
      expect(
        isAsyncStartVmSubmissionAck({
          status: 'success',
          event_id: '123',
        }),
      ).toBe(true)
    })

    it('returns false when vm_info indicates completed start', () => {
      expect(
        isAsyncStartVmSubmissionAck({
          status: 'success',
          event_id: '123',
          vm_info: { server_url: 'http://example.com' },
        }),
      ).toBe(false)
    })
  })

  describe('buildStartVmAcceptedStatusUpdate', () => {
    it('maps submission ack to processing with initial progress', () => {
      expect(
        buildStartVmAcceptedStatusUpdate({
          message: '自动创建资源并启动虚拟机请求已提交',
          event_id: '862532588237045760',
        }),
      ).toEqual({
        status: 'processing',
        message: '自动创建资源并启动虚拟机请求已提交',
        progress: 5,
        event_id: '862532588237045760',
      })
    })

    it('forwards start request _traceId as trace_id for statusTraceId / container-meta', () => {
      expect(
        buildStartVmAcceptedStatusUpdate({
          message: '启动请求已提交',
          event_id: 'ev-1',
          _traceId: 'web-start-abc12345',
        }),
      ).toEqual({
        status: 'processing',
        message: '启动请求已提交',
        progress: 5,
        event_id: 'ev-1',
        trace_id: 'web-start-abc12345',
      })
    })

    it('prefers response body trace_id over inbound _traceId', () => {
      expect(
        buildStartVmAcceptedStatusUpdate({
          message: '启动请求已提交',
          event_id: 'ev-1',
          _traceId: 'task_15742467311115154867',
          trace_id: 'cmt-start-independent',
        }),
      ).toEqual({
        status: 'processing',
        message: '启动请求已提交',
        progress: 5,
        event_id: 'ev-1',
        trace_id: 'cmt-start-independent',
      })
    })
  })

  describe('applyStartVmHttpResult', () => {
    it('updates processing state for async submission ack', () => {
      const updateServerStatus = vi.fn()
      const result = applyStartVmHttpResult(
        {
          status: 'success',
          message: '自动创建资源并启动虚拟机请求已提交',
          event_id: '862532588237045760',
        },
        updateServerStatus,
      )

      expect(result).toBe('accepted')
      expect(updateServerStatus).toHaveBeenCalledWith({
        status: 'processing',
        message: '自动创建资源并启动虚拟机请求已提交',
        progress: 5,
        event_id: '862532588237045760',
      })
    })

    it('propagates _traceId into status update for container-meta start TraceId', () => {
      const updateServerStatus = vi.fn()
      applyStartVmHttpResult(
        {
          status: 'success',
          message: '启动请求已提交',
          event_id: 'ev-9',
          _traceId: 'web-start-xyz',
        },
        updateServerStatus,
      )
      expect(updateServerStatus).toHaveBeenCalledWith(
        expect.objectContaining({
          status: 'processing',
          event_id: 'ev-9',
          trace_id: 'web-start-xyz',
        }),
      )
    })

    it('maps API error payload to error status', () => {
      const updateServerStatus = vi.fn()
      const result = applyStartVmHttpResult(
        { status: 'error', message: '配额不足' },
        updateServerStatus,
      )

      expect(result).toBe('error')
      expect(updateServerStatus).toHaveBeenCalledWith(
        expect.objectContaining({
          status: 'error',
          message: '配额不足',
          progress: 0,
          error_title: '服务器启动失败',
          error_code: 'STARTUP_ERROR',
          error_hint: '配额不足',
        }),
      )
    })
  })
})
