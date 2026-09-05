# [运行时] 厂商门户添加镜像后版本列表恒空（image_group vs image_group_id）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-16
- 最后修改：2026-07-16
- 维护者：Trae AI 团队

## 现象

- 页面 `https://provider.daydaymoney.com/`（厂商门户）已成功添加容器镜像版本（DB / `GET /api/vendor/container-images/` 有记录）。
- 展开镜像组后显示「暂无版本」；组头 `versions_count` 为 `undefined`。
- 公开目录 `/catalog` 表格为空（若镜像仍为 `draft`，此为预期，不属本 bug）。

## 根因

1. 列表 API `scanContainerImageRows` 将组外键序列化为 **`image_group`**（字符串 ID）。
2. 前端 `getGroupVersions` 误用 **`im.image_group_id`** 过滤 → 恒为 `undefined` → 版本列表恒空。
3. `ListImageGroups` 不返回 `versions_count` / `latest_version`，前端直接渲染导致「undefined 个版本」；`removeGroup` 用 `versions_count > 0` 守卫时也会失效（有版本仍可删组）。

## 解决方案

1. 抽出 `frontend/src/utils/containerImageGroup.js`：`containerImageGroupId` / `getGroupVersions` / `enrichImageGroups` / `statusDisplay`。
2. `VendorPortal.vue` 改用上述工具；`refreshGroups`/`refreshImages` 后富化组列表；`removeGroup` 以真实版本数守卫。
3. 后端列表补充 `status_display`（`domain.StatusDisplay`）。
4. 单元测试：`frontend/tests/containerImageGroup.unit.test.js`；Go：`TestVendorContainerImagesListHasImageGroupAndStatusDisplay`。
5. `npm run build` 后重启 `taskAiProvider`（静态 `dist` + 二进制）。

## 预防

- 前后端 FK 字段名以 API 实际 JSON 为准做契约测试；禁止仅凭 ORM 列名在前端臆测。
- 列表过滤/展示字段变更时，用「真实响应样例 → 纯函数过滤」单测锁住（本例故意断言旧 `image_group_id` 过滤结果为 0）。

## 验证

```bash
TOKEN=$(bash taskAiProvider/scripts/mint_token.sh vendor contact@daydaymoney.com | tail -1)
curl -sS -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8010/api/vendor/container-images/ | jq '.[0] | {image_group, status_display}'
cd taskAiProvider/frontend && npm run test:unit -- ./tests/containerImageGroup.unit.test.js
```

登录厂商门户 → 展开镜像组 → 应看到已添加版本（含草稿）。

## 关联

- API 嵌套/字段契约：`17_billing_transactions_unit_id_not_nested.md`
- SSO string sub：`23_sso_bridge_missing_sub_string_claim.md`
