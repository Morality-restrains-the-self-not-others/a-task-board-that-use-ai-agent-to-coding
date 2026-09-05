# [运行时] 厂商门户保存运行环境后关联被清空 → 自动运行无法启动服务器

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

任务详情页显示「自动运行」为「是」，但「服务器启动状态」为未启动。典型 URL：`.../task-detail/task_*?relayToTrae=true`。

后端日志：

```text
task_auto_run_start_vm_error ... cloud start-vm-auto HTTP 400:
{"message":"镜像市场返回空的运行环境列表","status":"error"}
```

同镜像此前可成功 `start-vm-auto`，随后在厂商门户多次「设置区域运行环境」后，公开接口：

`GET /api/public/image-runtime-environments/?image_id=<external_image_id>` → `[]`。

## 环境与上下文

- 路径：创建/更新任务 `auto_run=true` → `scheduleTaskAutoRun` → `POST .../start-vm-auto/` → `resolveCloudServerImageID` → `fetchAIPublicImageRuntimeEnvironments`
- 厂商门户：`POST /api/vendor/container-images/{id}/cloud-server-image-association/`
- 相关：`taskAiProvider` Go 迁版后的关联 API 与 `VendorPortal.vue`

## 根因

1. **字段名不一致**：前端 `saveRegionEnv` 提交 `cloud_server_image`，后端 `SetAssociations` 只读 `cloud_server_image_id`。
2. **先删后插且静默跳过**：`SetAssociations` 先 `DELETE` 该容器全部关联，再对无效 item `continue` → 关联被清空仍返回 200。
3. **单区域保存误用全量替换**：单条 association POST 也走 `SetAssociations`（全删），即使字段正确也会抹掉其他地域。
4. **列表形状过简**：`ListAssociations` 曾把 `cloud_server_image` 序列化为纯 ID 字符串，前端按对象读 `.id` / `.image_name` 失败。

## 修复

1. 后端：接受 `cloud_server_image_id` 与遗留 `cloud_server_image`；无效 payload **先校验再删**；单条 POST 改 `UpsertAssociation`（按 platform+region）；列表返回嵌套 CSI 对象 + `cloud_server_image_id`。
2. 前端：`buildRegionEnvAssociationPayload` 只发 `cloud_server_image_id`；`VendorPortal` 改用该工具。
3. 数据：为受影响镜像恢复 `aliyun/cn-hongkong` 关联；重放 `start-vm-auto` 验证 200。
4. 测例：`store_associations_test.go`、`regionEnvAssociationPayload.unit.test.js`。

## 预防

- 关联/外键类 API：列表字段名与写入字段名做契约测试；禁止「DELETE ALL → 解析失败仍 200」。
- 单资源 PATCH/POST 不得实现为全量替换，除非文档与 UI 明确发送全集。
- `auto_run` 异步失败时，任务详情须能看到 SSE error（本例已有）；可选后续在 create 门禁预检运行环境非空。

## 验证

```bash
curl -sS "http://127.0.0.1:8010/api/public/image-runtime-environments/?image_id=<ext_id>"
# 非 []
cd taskAiProvider && go test ./infrastructure/ -run 'Association|CloudServerImageID|Upsert' -count=1
cd taskAiProvider/frontend && npm run test:unit -- ./tests/regionEnvAssociationPayload.unit.test.js
```
