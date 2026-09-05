# 测试意图：推荐绩效抽屉微信分账 Tab

## 测试目标

Tab 切换、列表请求参数、同步按钮门闩与错误 `data-traceId`。

## 测试分层

Vue 组件单测（vitest + vue-test-utils）。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认 | 可见「支付明细」，微信表未请求或未展示分账表 |
| T2 | 点「微信分账」 | GET 含 `referrer_user_id` 与 `status=all` |
| T3 | 空 items | 「暂无微信分账记录」 |
| T4 | 有记录 | 被推荐人 ID、AppID、OpenID、订单号、金额、本地状态、微信订单号/分账单号可见 |
| T4g | app_id/openid 空 | 单元格「—」 |
| T4b | fail_reason=qualification_revoked | 失败原因列「推荐资格已撤销」，不展示英文码 |
| T4c | fail_reason 非空且 fail_trace_id 非空 | `[data-testid=referral-ps-fail-reason]` 的 `data-traceId` 等于该 id |
| T4d | fail_reason 非空但无 fail_trace_id | 失败原因列不挂 `data-traceId`（禁止占位符） |
| T4h | fail_reason 为微信 HTTP 400 JSON 长文案 | 单元格全文等于 fail_reason；无 `truncate` / `max-w-[8rem]`；有 `whitespace-normal` + `break-all` |
| T4e | wechat_state=尚未提交微信 | 列头「微信分账单状态」；单元格「未向微信发起分账」；title 说明是分账单而非绑定 |
| T4f | wechat_state=FINISHED | 单元格「微信已分账完成」，不展示英文码 |
| T5 | 同步成功 | 微信分账单状态列更新为中文；busy 结束后按钮可再点 |
| T6 | 同步失败 | 错误节点带 `data-traceId` |
| T7 | 同步进行中连点 | 第二次不发请求（guard） |
| T8 | 待分账点「分账」 | 缘由表单出现在表格上方；按钮不 disabled（aria-expanded）；不发 POST；确认后才 POST `/{id}/share/` 带 reason 与 Idempotency-Key |
| T9 | 缘由不足 8 字 | 不发 POST |
| T10 | 已完成 | 无「分账」按钮 |
| T11 | 分账失败 | 错误节点带 `data-traceId` |

## 通过标准

T1–T11、T4b–T4f、T4h 全绿。无新内部事件。
