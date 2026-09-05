# 约束 43 — Kafka Topic 命名规范与测试污染防治（一级）

## 目标

Kafka 集群 topic 必须使用**有业务语义的名称**，禁止以随机 ID/hash 命名；防止测试代码向生产/dev broker 自动创建无意义 topics 污染 metadata。

## 背景（2026-08-10 事故）

dev 集群积累 **3071 个 `kafka-go-<16hex>` topics**（metadata 共 3131 个）。根因：

1. `taskAuth/third_party/kafka-go/`（vendored 上游库）的集成测试套件（80 处 `makeTopic()` → `kafka-go-%016x`）被**随机单测抽测**（pre-commit 门禁 + 夜间巡检 nightly_test_sweep.py）误当本仓单测资产收集运行
2. 测试连 `localhost:9092`（docker kafka 可达），`DialLeader` 的 `AllowAutoTopicCreation: true` 自动创建 topic
3. 测试无 teardown 清理 → topic 永久残留

## 规则

1. **业务 topic 命名单一来源**：`taskEvents/config/config.go` 的 EventTopic/DeadLetterTopic 映射（28 业务 + 28 DLT，语义命名如 `task-created`）。新增业务事件 topic 必须在此定义，并同步 `db/_infra/kafka_recreate.py` 的 KAFKA_TOPICS/DLT_TOPICS
2. **禁止自动创建**：任何代码/测试不得依赖 broker 的 auto-topic-creation（`AllowAutoTopicCreation`）创建业务 topic；业务 topic 一律显式创建（kafka_recreate.py / 初始化流程）
3. **测试不得连真实 broker**：随机单测抽测的 `rt_go_collect_dirs` **排除 `third_party/`**（vendored 第三方库测试属上游集成测试，需真实外部依赖）；`rt_go_run_dir` 注入 `KAFKA_SKIP_NETTEST=1` 双保险。修改后须同步 `scripts/hooks/templates/lib/random_test_runner.sh` 并执行 `deploy_repo_random_precommit.sh` 重新分发
4. **残留治理**：若集群出现 `^kafka-go-[0-9a-f]{16}$` 模式 topics，运行 `python3 db/_infra/kafka_cleanup_junk_topics.py`（dry-run 默认；`--confirm DELETE_JUNK` 执行）清理

## 验收标准

- `bash scripts/lib/random_test_runner.sh collect-go taskAuth` 输出不含 `./third_party/` 目录
- 集群 metadata 中无 `kafka-go-*` 前缀 topics（业务 topics + DLT + `__consumer_offsets` 之外无其他）
- 新增业务事件时：EventTopic 映射 + kafka_recreate.py 同步，不出现硬编码 topic 名

## 关联

- 设计文档：`docs/superpowers/specs/2026-08-10-kafka-go-topic-pollution-fix-design.md`
- 随机单测抽测规则：`.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`
- 清理脚本：`db/_infra/kafka_cleanup_junk_topics.py`（含单测 `test_kafka_cleanup_junk_topics.py`）
