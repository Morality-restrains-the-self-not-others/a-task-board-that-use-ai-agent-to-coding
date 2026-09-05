/**
 * 看板拖拽结束时，为「被拖动的那张卡」构建 PATCH body。
 */
import { UNCATEGORIZED_DELIVERABLE_ID } from './workPanelDeliverableAggregation.js'

/**
 * - 同分栏：只写 order + progress_column_id
 * - 跨交付物分栏：额外写 deliverable_obj_id（未分类 → null 以清空）
 * - 非被拖动卡：仅写 order
 *
 * @param {{
 *   draggedTaskId: string,
 *   cardTaskId: string,
 *   order: number,
 *   targetProgressColumnId: string|null|undefined,
 *   sourceDeliverableColumnId: string|null|undefined,
 *   targetDeliverableColumnId: string|null|undefined,
 * }} args
 * @returns {Record<string, unknown>}
 */
export function buildDraggedTaskPatchPayload(args) {
  if (!args || typeof args !== 'object') {
    throw new Error('buildDraggedTaskPatchPayload: args required')
  }
  const {
    draggedTaskId,
    cardTaskId,
    order,
    targetProgressColumnId,
    sourceDeliverableColumnId,
    targetDeliverableColumnId,
  } = args
  if (draggedTaskId == null || draggedTaskId === '') {
    throw new Error('buildDraggedTaskPatchPayload: draggedTaskId required')
  }
  if (cardTaskId == null || cardTaskId === '') {
    throw new Error('buildDraggedTaskPatchPayload: cardTaskId required')
  }
  const orderNum = Number(order)
  if (!Number.isFinite(orderNum)) {
    throw new Error('buildDraggedTaskPatchPayload: order must be a number')
  }

  /** @type {Record<string, unknown>} */
  const payload = { order: orderNum }
  if (String(cardTaskId) !== String(draggedTaskId)) {
    return payload
  }

  if (targetProgressColumnId != null && targetProgressColumnId !== '') {
    payload.progress_column_id = String(targetProgressColumnId)
  }

  const src =
    sourceDeliverableColumnId == null || sourceDeliverableColumnId === ''
      ? null
      : String(sourceDeliverableColumnId)
  const dst =
    targetDeliverableColumnId == null || targetDeliverableColumnId === ''
      ? null
      : String(targetDeliverableColumnId)

  if (dst != null && dst !== src) {
    if (dst === UNCATEGORIZED_DELIVERABLE_ID) {
      payload.deliverable_obj_id = null
    } else {
      payload.deliverable_obj_id = dst
    }
  }

  return payload
}
