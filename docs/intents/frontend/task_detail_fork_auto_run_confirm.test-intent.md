# 测试意图：任务详情 Fork 确认是否自动运行

| # | 场景 | 期望 |
|---|------|------|
| T1 | 点击 Fork | 显示确认模态，未立即 POST |
| T2 | 模态点取消 | 不调用创建 API，`isForking` 仍为 false |
| T3 | 选择「不自动运行，仅派生」并确认 | POST body `auto_run === false`，含 `fork_from`，无 `agent_models` |
| T4 | 选择「自动运行并派生」并确认（已选 ≥1 模型） | POST `auto_run === true`；若查到公网 IP 则含 `client_public_ip`；含 `agent_models` |
| T5 | 派生成功 | 新标签打开新任务详情，关闭模态 |
| T6 | 关联仓库且身份未选 | 出现 Git 身份选择；模式为自动运行时「确认派生」disabled；切到仅派生后可确认 |
| T7 | 关联仓库且已选当前用户身份后自动运行 | POST `repo_identities` 为当前用户身份，不得用源任务他人身份 |
| T8 | 无关联仓库 | 不展示 Git 身份，不因身份拦截自动运行 |
| T9 | 打开弹窗 | 无副本数量框；自动运行未选模型前不可确认 |
| T10 | 仅派生确认 | 1 次 POST，无 `fork_count`，无 `agent_models` |
| T11 | （废止）仅派生数量 100 | 本迭代删除数字框，不再适用 |
| T12 | 第 1 份失败 | 不打开标签，模态保持 |
| T13 | 第 2 份失败 | 保留第 1 份并打开，提示部分失败 |
| T14 | 选自动运行但未选智能体（模型） | 确认 disabled |
| T15 | 同一智能体资源勾选两个模型后确认自动运行 | 2 次 POST，`agent_models[0].model` 不同，不用 `fork_count`，Idempotency-Key `${batch}:1` 与 `:2` |
| T16 | 仅派生时即使先前勾过模型 | 仍只创建 1 份同构任务，不按模型分叉 |
| T17 | 切换智能体资源 | 已选模型清空（可预勾新配置默认模型） |
| T18 | auto_run 创建 pending agent | job/context 含该份 `agent_models` |
| T19 | 自动运行拉模型时上游持续 502（APISIX HTML） | 先有限退避重试同一 GET；耗尽后 `fork-agent-models-error` 文案为「服务暂时不可用，请稍后重试」，带 `data-traceId`；可点 `fork-agent-models-retry` 重拉 |
| T20 | 自动运行拉模型首包 502、随后 200 | 不展示 error；列出模型选项 |
