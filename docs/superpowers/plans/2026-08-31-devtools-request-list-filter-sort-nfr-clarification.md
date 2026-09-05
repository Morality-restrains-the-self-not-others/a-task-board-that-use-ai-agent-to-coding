# NFR 澄清：DevTools 请求列表过滤与排序

- **Date:** 2026-08-31
- **Level default:** L2（客户端视图）；资金/鉴权路径不触及

## 路径分片键审视

| 路径 | 分片 ID | 可伸缩性 | 结论 |
|------|---------|----------|------|
| DevTools panel 本地 `queryRequestList` | 无 | L0 | 单 tab、内存 ≤200 条；升级触发：缓冲 >2000 再考虑虚拟列表 |
| 既有 `createTask` HTTP | 既有 workspace/company | 不在本增量 | 不改 |

## 幂等性审视

| 路径 | 副作用 | 级别 | 说明 |
|------|--------|------|------|
| 过滤/排序/点表头 | 无 | L0 | 纯视图；重复点击只改本地 sort state |
| 刷新列表 `getRecentRequests` | 无写 | L0 | 只读 SW |
| `createTask` | 有 | 既有 | 本增量不改；前端按钮仍走既有 create 防重放 |

## 类别

| 类别 | 等级 | 场景 |
|------|------|------|
| 性能 | L2 | 200 条 filter+sort < 16ms（同步主线程可接受） |
| 可用性/无障碍 | L2 | 表头 `aria-sort`；芯片 `aria-pressed` |
| 安全 | L2 | 输出编码沿用 `escHtml`；不日志 URL/body |
| 可观测性 | L1 | 过滤结果条数可用既有 `#requestCount`；不新增生产 console |
| 一致性 | L0 | 无跨端状态 |

## 领域模型影响

查询对象为无副作用值对象；无需聚合或事件。
