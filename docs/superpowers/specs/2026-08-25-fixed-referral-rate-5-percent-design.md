# 设计：推荐分成比例固定 5%

- **日期**: 2026-08-25
- **状态**: accepted（/goal 自动采用）
- **页面**: `/system-admin/referral-management`
- **架构变更**: 无（不新增服务/事件/表；无需 ArchiMate / ADR）

## 背景与目标

超管「推荐资格管理」分成比例卡片当前允许在 5%–30% 内设置单一数值，并从 `billing_referral_config` 读取/写入。产品要求：**分成比例固定为 5%，不可变，也不从配置读取**。点数计提与微信打款均使用该 5%（打款仍不超过微信商户 `max_ratio`）。

## 选定方案

把 `5` 作为领域常量（已有 `referralCommissionRateNum`），全链路只读该常量：

1. **管理页卡片**：只展示「5%」，无数字输入、无保存按钮、不把 GET config 的 `referral_rate_percent` 绑到 UI。
2. **taskBill**：`getReferralRatePercent()` 恒返回 5，不读库。计提、分账落库金额、打款 `min(5%, wechat max_ratio)` 同源。
3. **taskReferral**：`configuredReferralRateDisplay()` 恒返回 `5%`，不读 `billing_referral_config`、不回退微信 `max_ratio`。
4. **POST config**：仍可改 `settle_delay_days`；忽略 body 中的比例字段；响应里比例始终为 5。
5. **库列**：保留 `referral_rate_percent` / `profit_sharing_ratio_percent`（避免 DDL），写入时固定 5，业务路径禁止当政策源读取。

拒绝方案：前端只读但仍 GET 配置（违反「不从配置读取」）；仅禁用输入仍 POST（仍可变）。

## 范围

- 范围内：管理页比例卡、计提、微信打款比例、stats/status/performance/申请列表展示、个人推荐页「比例可能调整」说明。
- 范围外：入账天数、资格审批、微信 `max_ratio` 查询接口本身、删除 DB 列。

## 验收

1. 比例卡无 `referral-rate-input` / `referral-ratio-save`；可见固定 `5%`。
2. 即使 GET config 返回 12，管理页仍展示 5%。
3. 库中写成 12 时，计提仍按 5%；打款 `min(5, wechat_max)`。
4. 用户侧展示与说明不再暗示比例可由后台改动。
