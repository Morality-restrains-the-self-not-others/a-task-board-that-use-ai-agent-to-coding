# DDD：管理端租户下拉联系方式

无新聚合。`Company`（taskTenant）仍以 `tenant_company` 为 SSOT；`LoginMethod`（taskAuth）为创建者联系方式 SSOT。应用服务在查询用例内编排：搜公司 →（可选）按标识搜用户再反查公司 → 批量取联系方式 → 投影到选项 DTO。

端口：`FetchCreatorContacts`、`SearchCreatorIDs`（测试可替换）。适配器：HTTP + Internal-Secret。

不发布领域事件（只读查询例外，见意图对照表）。
