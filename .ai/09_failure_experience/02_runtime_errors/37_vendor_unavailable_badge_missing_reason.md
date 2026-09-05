# [运行时] 厂商门户「不可用」徽章无具体原因

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-18
- 最后修改：2026-07-18
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/`（标题「AI 容器镜像市场」）
- 元素：`span.badge.badge-unavailable`，可见文本仅「不可用」
- 无法得知需补齐区域运行环境还是 UserData 模板

## 根因

1. 公开目录 `ListApprovedCatalog` 会调用 `marketplaceAvailability` 填充 `is_marketplace_available` / `unavailable_reason`。
2. 厂商/管理端列表走 `scanContainerImageRows`，**未**附带上述字段。
3. 前端用 `!im.is_marketplace_available`：字段缺失时恒为 true → 恒显「不可用」；`unavailable_reason` 为空 → 原因区不渲染。

## 解决方案

1. `ListContainerImages` / `ListContainerImagesAdmin` 经 `attachMarketplaceAvailability` 补齐字段。
2. 前端改为 `isMarketplaceUnavailable`（`=== false`）+ `formatUnavailableBadge`（徽章内联原因）。
3. 测例：`TestVendorContainerImagesListHasImageGroupAndStatusDisplay`、`containerImageGroup.unit.test.js`。

## 预防

- 列表 API 与前端展示字段须同批契约测试；禁止对可选 bool 用 `!` 判断缺失。
- 公开目录已有的派生字段，厂商列表复用同一计算函数，避免双路径漂移。

## 验证

```bash
cd taskAiProvider && go test ./src/ -run TestVendorContainerImagesListHasImageGroupAndStatusDisplay -count=1
cd taskAiProvider/frontend && npm run test:unit -- ./tests/containerImageGroup.unit.test.js
TOKEN=$(bash taskAiProvider/scripts/mint_token.sh vendor <email> | tail -1)
curl -sS -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8010/api/vendor/container-images/ \
  | jq '.[0] | {is_marketplace_available, unavailable_reason}'
```

## 关联

- 意图：`docs/intents/backend/vendor_marketplace_unavailable_reason.intent.md`
- 字段契约类比：`25_provider_vendor_list_image_group_field_mismatch.md`
