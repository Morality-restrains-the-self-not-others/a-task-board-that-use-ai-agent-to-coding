# 实施计划：租户下单一律打微信分账标识

- **日期**: 2026-08-23
- **设计 / NFR / 价值流**: 同主题 `2026-08-23-order-always-profit-sharing-flag-*`

## Tasks

- [x] **T1** 改预下单测例：无推荐人 / 资格 inactive / 资格失败均断言 `SettleInfo.ProfitSharing==true`（Red）
- [x] **T2** `wechatPrepay` live 无条件设置 SettleInfo；删除或掏空 `shouldFlagWechatProfitSharing` 预下单门禁（Green）
- [x] **T3** 保持 `markOrderForProfitSharing` 资格测例：无推荐人不落行、有资格落行
- [x] **T4** 新增解冻测例：无佣金行调用 Unfreeze 且 `OutOrderNo=UF{order_id}`；有佣金行不调用
- [x] **T5** `markOrderPaid` 微信成功路径：无行则 `unfreezeProfitSharing`；mock 跳过
- [x] **T6** 结构化日志：unfreeze 成功/失败含 order_id，无密钥
- [x] **T7** 更新意图、价值流 VS-PS-4/5/7、wechatpay skill 预下单条款
- [x] **T8** 事件对照：无新 MQ；书面例外已在意图表

验证：

```bash
cd taskBill && go test ./src -count=1 -timeout 120s \
  -run 'TestWechatPrepay|TestMarkOrderForProfitSharing|TestUnfreezeRemaining'
```
