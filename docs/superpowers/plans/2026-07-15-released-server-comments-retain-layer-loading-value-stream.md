# 价值流：released-server-comments-retain-layer-loading

- **日期**: 2026-07-15
- **关联设计**: `docs/superpowers/specs/2026-07-15-released-server-comments-retain-layer-loading-design.md`

## 受影响流（`conf/value-stream.yaml`）

| Stream / Step | 影响 |
|---------------|------|
| `task-detail-runtime-relay` / container-runtime-context | 释放后 UI 收敛为非服务态，不再假 connecting |
| `task-comments-api` / `container-image-at-mention` | 只读：释放后 Feed 仍展示历史评论（含 AI / Agent） |

## 增量切片

1. **MVP**：非服务态隐藏 connecting spinner + 静态空态 + stop SSE pause 对称
2. **冷打开**：runtime=`released` 进入详情即空态
3. **回归**：运行中 connecting 文案仍可用

## 字段

无新 `<service>.<table>.<column>`。评论表只读。

## 测试期望

- 前端单测覆盖 S1/S2/S4/S5/S6
- 可选 Playwright：released + 预制评论
