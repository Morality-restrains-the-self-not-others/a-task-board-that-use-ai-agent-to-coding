# [运行时] 厂商门户「设置区域运行环境」保存 400 且模态内无提示

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/` → 厂商门户 → 设置区域运行环境 → 保存
- 请求：`POST /api/vendor/container-images/{id}/cloud-server-image-association/`
- 状态码：400
- 请求体示例：`{"platform_type":"aliyun","region":"cn-hongkong","cloud_server_image_id":null}`
- 响应头含 `x-trace-id`；**模态可见区域无任何错误文案**

## 环境与上下文

- 前端：`taskAiProvider/frontend/src/views/VendorPortal.vue`（`saveRegionEnv`）
- 后端：`UpsertAssociation`（`store_marketplace.go`）经 `handleVendorContainerAction`
- 与 FE-32（字段名误用清空关联）、FE-33（仅 X-Trace-Id）不同：本例请求已带 `X-Parent-Span-Id`，业务体为「不选」清空

## 根因

1. **业务**：下拉「— 不选 —」会经 `buildRegionEnvAssociationPayload` 发送 `cloud_server_image_id: null`；`UpsertAssociation` 将 `csiID==0` 一律判非法并 400，未支持「仅清除本区域关联」。
2. **UX**：`saveRegionEnv` 失败时调用页面级 `setMsgError` → `p.msg`；区域环境对话框为全屏遮罩，**错误被挡在模态后方**，用户视口内看不到。

## 解决方案

1. 后端：`UpsertAssociation` 在 `platform`+`region` 合法且 `csiID==0` 时只 DELETE 该区域绑定并提交（不清其他区域）。
2. 前端：模态内增加 `regionEnvError` / `regionEnvErrorTraceId`，失败时 `setRegionEnvError`，错误节点带 `data-traceId`；文案标明「不选」= 清除本区域关联。
3. 测例：`TestUpsertAssociationClearWithZeroCSIRemovesOnlyThatRegion`。

## 预防

- 模态内发起的请求失败，错误 UI **必须**挂在该模态（或 toast）内，禁止只写被遮罩挡住的页面级 `p.msg`。
- UI「清空/不选」选项与 API 语义必须对齐（清除 vs 必填）；若仅作占位，应禁用保存并做前端校验。
- 单区域 association POST：`csiID==0` = clear that region；全量 `items[]` 仍走 `SetAssociations` 校验门禁。

## 验证

```bash
cd taskAiProvider && go test ./infrastructure/ -run 'Association|Upsert' -count=1
cd taskAiProvider/frontend && npm run test:unit -- ./tests/regionEnvAssociationPayload.unit.test.js
```

部署：需发布 Go 与前端 dist 到 `provider.daydaymoney.com` 后，硬刷新再验：选「不选」保存应 200 并清除本区域；其它 400 应在模态内见 `p.err[data-traceId]`。

## 相关案例

- [32_vendor_runtime_env_association_field_wipes.md](./32_vendor_runtime_env_association_field_wipes.md)
- [33_provider_frontend_trace_id_only_400.md](./33_provider_frontend_trace_id_only_400.md)
