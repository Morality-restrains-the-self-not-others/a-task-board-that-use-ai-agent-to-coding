# 测试意图：订单状态筛选与徽章统一中文

- **对应功能意图**: `system_admin_order_status_zh_display.intent.md`
- **日期**: 2026-08-23

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| S1 | `statusLabel('refunded')` | `已退款` |
| S2 | 超管筛选常量 | 含 `{ value: 'refunded', label: '已退款' }` |
| S3 | 租户筛选常量 | 与超管同一 SSOT，含已退款 |
| S4 | 订单详情 `status=refunded` | 文案含「已退款」，不含英文 `refunded` |

| S5 | 超管列表 `status=refunded` | 筛选栏含「已退款」，徽章文案为「已退款」 |

## 可执行测试

- `taskFE/app/src/utils/systemAdminOrderListDisplay.test.js`
- `taskFE/app/src/utils/billingOrderDisplay.test.js`
- `taskFE/app/src/views/OrderDetail.refund.test.js`
- `taskFE/app/src/components/system-admin/SystemAdminOrderListPanel.deeplink.test.js`
