# 意图 publish 证据豁免（known-debt）

本目录 `publish_evidence_exempt.yaml` 登记第二层 CI 允许的「证据豁免」行。

## 改走 MQ 时的步骤

1. 在生产路径 `send_event` / `PublishEvent`（或等价）投递 `future_mq_contract`
2. 更新意图表：填写真实「MQ类型/契约」，删除例外理由中的「证据豁免」
3. 删除本 YAML 中对应 `id` 条目
4. 跑 `python3 task2app/scripts/ci/check_ddd_bdd_compliance.py --skip-bdd`

未登记的「证据豁免」或登记残留但意图已无豁免行，均会令 CI 失败。
