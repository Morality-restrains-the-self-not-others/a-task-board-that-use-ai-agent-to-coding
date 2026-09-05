# [运行时] 厂商门户保存区域运行环境后 UserData 列仍显示「—」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/` → AI 容器镜像市场 → 厂商门户 → 展开「各区域运行环境」
- 元素：`span.col-userdata.userdata-cell` 可见文本为「—」
- 用户在「设置区域运行环境」中选择了服务器镜像与 UserData 模板并保存成功后，列表 UserData 列仍无对应显示

## 环境与上下文

- 前端：`taskAiProvider/frontend/src/views/VendorPortal.vue`（`runtimeEnvUserdataTemplateLabel` / `userdataTemplateLabel`）
- 关联 API：`POST/GET .../cloud-server-image-association(s)/`
- CSI API：`POST/PATCH /api/vendor/cloud-server-images/`
- 相关：FE-32（关联字段清空）、FE-34（区域环境模态错误）

## 根因

1. **列表契约不完整**：`ListAssociations` 仅返回 `userdata_template: {id}`，前端 `userdataTemplateLabel` 依赖 `name` → 有 id 时显示「—」（不是「未选择」）。
2. **CSI 写入空转**：`CreateCloudServerImage` 的 SQL 把 `userdata_template_id` 写死为 `NULL`；`PATCH` handler 直接 `200 {id}`，不落库。
3. **关联保存忽略模板**：区域环境 POST 虽带 `userdata_template_id`，但 `UpsertAssociation` 只改关联表，不更新 CSI 上的模板外键。
4. **列表形状不一致**：`scanCSI` 曾把 `userdata_template` 序列化为纯 ID 字符串，与门户按对象读 `.id` / `.name` 不一致。

## 修复

1. `ListAssociations` / `ListCloudServerImages`：`LEFT JOIN marketplace_userdatatemplate`，返回 `{id, name, version}`。
2. `CreateCloudServerImage` / `UpdateCloudServerImage`：持久化 `userdata_template_id`（及硬件等可选字段）；PATCH 真正更新并回读。
3. 关联 POST：在 Upsert 后若 body 含 `userdata_template_id`，调用 `SetCloudServerImageUserdataTemplate`。
4. 前端：`userdataTemplateDisplayLabel` + 展开运行环境时加载模板选项，bare id 可降级解析。
5. 测例：`store_associations_test.go`（enrich / persist / create-update）、`userdataTemplateDisplay.unit.test.js`。

## 预防

- 展示列依赖的嵌套对象须含 UI 字段（name/version），禁止只回 id 却用「有 name 才显示」。
- 表单可选字段与 CREATE/PATCH SQL 做契约测试，禁止 stub 200。
- 同一业务模态提交的关联字段（CSI + UserData）须在同一请求路径内全部落库。

## 验证

```bash
cd taskAiProvider && go test ./infrastructure/ -run 'Association|CloudServerImage|OptionalUserdata|CreateAndUpdate' -count=1
cd taskAiProvider/frontend && npm run test:unit -- ./tests/userdataTemplateDisplay.unit.test.js
# 部署后：设置区域运行环境选模板保存 → UserData 列应为「{name} v{version}」
```

## 相关案例

- [32_vendor_runtime_env_association_field_wipes.md](./32_vendor_runtime_env_association_field_wipes.md)
- [34_vendor_region_env_modal_error_hidden_null_csi.md](./34_vendor_region_env_modal_error_hidden_null_csi.md)
