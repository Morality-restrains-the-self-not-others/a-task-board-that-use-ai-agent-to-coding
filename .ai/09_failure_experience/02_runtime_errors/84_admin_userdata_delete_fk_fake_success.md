# [运行时] 平台审核删除 UserData 模板刷新后复现

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-24
- 最后修改：2026-07-24
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/admin` → UserData 模板 → 点「删除」
- 即时表现：列表行消失并提示成功
- 刷新后：同一模板重新出现

## 环境与上下文

- 前端：`AdminUserDataTemplates.vue` / `useUserDataTemplateCrud.handleDeleteTemplate`
- API：`DELETE /api/admin/userdata-templates/{id}/`
- Store：`DeleteUserDataTemplate`；SQLite 开启 `foreign_keys(1)`
- FK：`marketplace_vendorcloudserverimage.userdata_template_id` → `marketplace_userdatatemplate.id`

## 根因

1. 模板仍被云服务器镜像（CSI）引用时，硬 `DELETE` 触发 **FOREIGN KEY constraint failed**。
2. Handler 写为 `_ = a.DB.DeleteUserDataTemplate(id)` **忽略错误**，仍 `204`。
3. 前端按成功从本地数组移除 → 刷新重新拉列表 → 行「复活」。

## 修复

1. `DeleteUserDataTemplate`：事务内先 `UPDATE … userdata_template_id=NULL`，再 `DELETE`，`RowsAffected==0` 返回 not found。
2. Handler：删除失败返回 404/400，禁止假 204。
3. 前端：`String(id)` 过滤，避免类型不一致漏删本地行。
4. 测例：`TestAdminUserDataTemplateDeleteClearsCSIRefs`。

## 预防

- 写接口禁止忽略 `error` 后固定成功码。
- 有 FK 的资源删除须先断引用、软删，或明确 409；并有「引用存在仍可删」的契约测试。

## 验证

```bash
cd taskAiProvider && go test ./src/ -run 'AdminUserDataTemplateDeleteClearsCSIRefs' -count=1
# 管理端：删除被 CSI 引用的模板 → 204 → 刷新列表不再出现；CSI.userdata_template_id 为 NULL
```

## 相关案例

- [83_admin_userdata_created_at_column_empty.md](./83_admin_userdata_created_at_column_empty.md)
- [36_vendor_userdata_cell_dash_after_save.md](./36_vendor_userdata_cell_dash_after_save.md)
