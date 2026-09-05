# 意图：评论 CSC 云平台元数据不变量（禁止 mock / 空 region / 空授权）

## 背景与目标

库内出现评论 CSC：`platform=mock`、`region=''`、`authorization_id=''`，同时任务级模板已是 `aliyun` + 地域 + 授权。Workbench / runtime-status 因此走 mock 或未授权降级。

上一轮自愈把「platform=mock 的评论行」整段覆盖成任务级模板。这不对：**任务级只是新建默认值；用户（或 start-vm / 闲置复用）仍可能把评论级改成与模板不同的合法 platform / region / authorization_id。** 合法覆盖不得被读路径或存量 SQL 抹掉。

目标：评论 CSC 永远持有真实云平台三元组；非法空值才从模板**按字段补齐**；已有合法值保持评论级权威。

## 范围与边界

- 范围内：`ensureCommentCloudServerConfig` 创建/复用；`resolveScoped` 读路径补齐；`020` 存量 SQL 改为按字段补齐；评论级 PATCH/start-vm 写入校验。
- 范围外：不改 Workbench URL 格式；不把任务级重新写成运行实例；不禁止评论级与模板不同。

## 约束与风险

- `platform=mock` / 空、`region` 空、`authorization_id` 空 对评论行非法。
- 补齐只填非法/空字段，不覆盖非空合法字段。
- 模板自身也非法时：拒绝新建评论行（400），不落 mock 行。

## 验收标准

1. 新建评论 CSC：模板合法 → 行上 platform≠mock，region/auth 非空；模板非法 → 不落库。
2. 评论已是 `aliyun` + 自定义 region/auth（≠模板）→ ensure / resolve / 020 均不改这些字段。
3. 评论 `platform=mock` 且 region/auth 空、模板合法 → 只补这三字段，不改 instance/server_url。
4. 评论 `platform=aliyun`、region 空、auth 已是用户值 → 只补 region，不改 auth。
5. Workbench：真实 `i-…` + 合法三元组 → 200；`mock-` 实例仍拒绝。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 评论 CSC 非法元数据按字段补齐 | — | — | — | — | 同服务同库投影，无跨边界副作用；登记证据豁免 |
| 拒绝落库非法评论 CSC | — | — | — | — | 校验失败无状态变更 |

## 实施计划

1. 抽出 `fillCommentCSCIllegalCloudMeta`（按字段补齐）替换整段覆盖。
2. `ensure` 禁止默认 `platform=mock`；补齐后仍非法则 error、不 INSERT。
3. `resolveScoped` 只对非法字段补齐并落库。
4. 修订 `020` 为 `CASE WHEN` 空/mock 才写模板值。
5. 单测：自定义覆盖不被抹；空字段才补；新建拒绝非法模板。

## 变更记录

- 2026-08-14：从「整段覆盖模板」改为「非法字段补齐 + 评论级可覆盖」。
