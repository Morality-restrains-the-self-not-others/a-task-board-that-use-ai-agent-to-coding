# 功能意图：厂商申请证照与联系方式验证

## 意图

主站用户在厂商门户（`provider.*`）申请认证时，须上传身份证与营业执照，并完成手机短信联系方式验证后，方可提交申请供运营审核。申请入口不在租户镜像市场。

## 参与者

- 申请人（已绑定真实邮箱的主站用户）
- 运营（ai-provider staff）
- taskAiProvider（申请/证照存储）
- taskAuth（SMS 验证状态）
- taskBill（verify-phone-code 门面，复用）

## 成功路径

1. 申请人打开申请表单 → 填写公司名/联系人
2. 上传身份证、营业执照 → 获得 file_key（2026-08-14 起生产改为 COS 预签名直传，见 `vendor-docs-cos-presign.intent.md`）
3. 完成手机短信验证（SMS gate 置位）
4. 提交申请 → pending
5. 运营查看证照与脱敏手机号 → approve/reject

## 失败路径

- 缺邮箱 / 缺双证 / 未短信验证 / file_key 越权 → 4xx
- 已 pending/qualified → 409

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 提交/重提厂商申请 | — | — | — | 与 vendor-application-entry 一致，无 MQ 订阅方 |
| 证照直传确认 / 路径规则变更 | 见 `vendor-docs-cos-presign.intent.md` | — | — | 存储迁 COS 后拆出独立意图 |
