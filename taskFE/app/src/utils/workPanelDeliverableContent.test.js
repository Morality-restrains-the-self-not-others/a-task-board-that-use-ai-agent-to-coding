import { describe, expect, it } from 'vitest'
import {
  deliverableContentOptionLabel,
  readTodoWorkspaceSeq,
  todoToDeliverableContent,
} from './workPanelDeliverableContent.js'

describe('workPanelDeliverableContent', () => {
  it('readTodoWorkspaceSeq: 正整数序号', () => {
    expect(readTodoWorkspaceSeq({ workspace_seq: 12 })).toBe(12)
    expect(readTodoWorkspaceSeq({ workspaceSeq: '7' })).toBe(7)
  })

  it('readTodoWorkspaceSeq: 无效序号为空', () => {
    expect(readTodoWorkspaceSeq({ workspace_seq: 0 })).toBeUndefined()
    expect(readTodoWorkspaceSeq({ title: 'x' })).toBeUndefined()
    expect(readTodoWorkspaceSeq(null)).toBeUndefined()
  })

  it('todoToDeliverableContent 携带 workspace_seq 与标题', () => {
    expect(
      todoToDeliverableContent({
        id: '850256677331014872',
        title: '写一个 hello world程序',
        workspace_seq: 12,
      }),
    ).toEqual({
      id: '850256677331014872',
      title: '写一个 hello world程序',
      workspace_seq: 12,
    })
  })

  it('T1 有序号时 option 为 #N 标题', () => {
    expect(
      deliverableContentOptionLabel({
        id: '850256677331014872',
        title: 'hello',
        workspace_seq: 12,
      }),
    ).toBe('#12 hello')
  })

  it('T2 无序号时 option 为标题', () => {
    expect(
      deliverableContentOptionLabel({
        id: '850256677331014872',
        title: 'hello',
      }),
    ).toBe('hello')
    expect(
      deliverableContentOptionLabel({
        id: '850256677331014872',
        title: 'hello',
      }),
    ).not.toMatch(/#/)
  })

  it('T3 同标题不同序号可区分', () => {
    const a = deliverableContentOptionLabel({
      id: 'id-a',
      title: '写一个 hello world程序',
      workspace_seq: 3,
    })
    const b = deliverableContentOptionLabel({
      id: 'id-b',
      title: '写一个 hello world程序',
      workspace_seq: 8,
    })
    expect(a).toBe('#3 写一个 hello world程序')
    expect(b).toBe('#8 写一个 hello world程序')
    expect(a).not.toBe(b)
  })
})
