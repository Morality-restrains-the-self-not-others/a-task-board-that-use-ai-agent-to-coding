# 功能意图：编辑镜像组弹窗图标必填

- **日期**: 2026-08-28
- **页面**: `https://provider.daydaymoney.com/` 编辑/创建镜像组模态框
- **选择器**: `div.card.modal`（「编辑镜像组」）

## 验收

- 表单项「图标」带必填标记；`input[type=file]` accept PNG/JPEG/WEBP。
- 未选文件且无已保存图标时点保存：页面提示「镜像组图标必填」，不调用 PUT/POST。
- 保存按钮使用 clickGuard + Idempotency-Key；进行中 disabled + aria-busy。
- 请求失败错误节点带 `data-traceId`。
