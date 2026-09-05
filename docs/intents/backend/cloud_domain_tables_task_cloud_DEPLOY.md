# 部署清单：cloud_server_events / IAM 迁入 taskCloudService

> 关联：[`cloud_domain_tables_task_cloud_migration.intent.md`](./cloud_domain_tables_task_cloud_migration.intent.md)  
> 日期：2026-07-14  
> **执行环境**：本机 monorepo 全栈（`/tmp/ram-work`，等同生产路径约定）  
> **执行时间**：2026-07-14 01:31–01:33 CST  
> **证据目录**：[`_deploy_evidence_20260714/`](./_deploy_evidence_20260714/)  
> **Archi 校验**：`./docs/architecture/Archi/Archi … --loadModel docs/architecture/v21-application-integration-20260714-0130-claude.archimate` → `Loaded model … v21 Target`（exit 0）

按顺序执行，**不要跳步**。任一步失败先回滚/修复再继续。

## 前置

- [x] monorepo 已包含本轮代码（Cloud event/IAM store、taskEvents `doCloudJSON`、Django `cloud.0052`） — 证据 `00_pre.txt`
- [x] 已备份 `db/saas/saas.sqlite3` 与 `data/task_cloud.db`（或等价生产路径） — 证据 `backups/` + `01_backup.txt`
- [x] 确认 `shared.internalSecret` 在 Cloud 与 taskEvents 一致（`X-Internal-Secret`） — 本机 Cloud `shared` 为空（放行）；taskEvents Cloud 调用可带 secret，空配置下一致可用 — `02_secret.txt`

## 步骤

### 1. 先起 / 升级 taskCloudService

```bash
# 编译并启动（端口默认 8018）
cd taskCloudService && go build -o bin/taskCloudService ./src/
# 经 runAll 重启 task-cloud-service，或：
./bin/taskCloudService
curl -sf http://127.0.0.1:8018/api/health/
```

- [x] health `ok` — `{"service":"taskCloudService","status":"ok"}`（`10_cloud.txt`）
- [x] 日志出现 `migrations complete`（含 `cloud_server_events` / `access_key_iam_associations`） — `taskcloud.log` + sqlite 表存在

### 2. 数据迁移（二选一，推荐先 shell 再 Django）

**方案 A — shell（显式列、幂等）**

```bash
bash db/task_cloud/migrate_cloud_events_from_saas.sh
# 期望：cloud_server_events / access_key_iam_associations 行数 ≥ saas 旧表
```

**方案 B — 仅 Django 0052**（空库或已由 A 灌入后）

```bash
cd task2app/Saas_project
# 使用项目 venv
./requirements/.venvs/run/bin/python3 manage.py migrate cloud 0052 --noinput
```

`0052` 会：HTTP import → DROP saas `cloud_cloudserverevent` / `cloud_accesskeyiamidassociation`。

- [x] saas 中上述两表已不存在 — shell skip missing；migrate No migrations to apply（已应用）
- [x] `data/task_cloud.db` 中两表有数据（若生产原先有行） — events=26，iam=1（`20_migrate.txt`）
- [x] `GET http://127.0.0.1:8001/api/health/` → `migrations.ok=true`（无 pending `cloud.0052`）

### 3. 重启 taskEvents 消费者

使进程加载「事件/IAM 打 Cloud」的新二进制/配置：

```bash
# 经 runAll 重启 domain-events 组，或逐个重启
# cloud_server_started / cloud_server_start_auto / cloud_platform_authorization_created 等
```

- [x] 环境变量：`TASK_CLOUD_SERVICE_BASE_URL`（默认 `http://127.0.0.1:8018`） — 重启时已 export
- [x] `SHARED_INTERNAL_SECRET` 或与 Cloud 相同的 internal secret — Cloud 空 secret 放行；二进制含 `/api/internal/access-key-iam-associations/`
- [x] 日志无持续 `saas intent ... create-access-key-iam` / `latest-pending-start-event` 失败（应走 Cloud） — 新进程 health 正常；无 bind/fatal（`30_events.txt`）

### 4. 冒烟

```bash
bash scripts/smoke/internal-apis-live.sh
# 或 runAll UI → Internal API smoke「立即验证」
```

- [x] smoke `pass` 且 saas health 200 — `pass=8 fail=0`；ownership `0 known-debt`（`40_smoke.txt`）
- [x] 可选：起一台机，确认 `cloud_server_events` 写入 Cloud 且计费 `cloud_event_id` 为 Snowflake 字符串 — 以 internal import + `latest-pending-start` 探针代替全链路起机：写入 id=`9990000000000000001`（Snowflake 形态）后可读 pending（随后已清理探针行）

## 回滚要点

1. **勿**在未备份时 DROP saas 表。
2. 若 0052 已 DROP：从备份恢复 saas 表 + 回退 taskEvents/Cloud 二进制。
3. ownership CI：`python3 db/scripts/ci/check_single_service_db_ownership.py` 应保持 `0 known-debt warnings`。

本轮备份：`_deploy_evidence_20260714/backups/saas.sqlite3.20260714-013115`、`task_cloud.db.20260714-013115`。

## 验证命令速查

```bash
python3 db/scripts/ci/check_single_service_db_ownership.py
sqlite3 data/task_cloud.db "SELECT COUNT(*) FROM cloud_server_events; SELECT COUNT(*) FROM access_key_iam_associations;"
sqlite3 db/saas/saas.sqlite3 "SELECT name FROM sqlite_master WHERE name IN ('cloud_cloudserverevent','cloud_accesskeyiamidassociation');"
```

## Archi CLI（可视化确认）

```bash
xvfb-run -a ./docs/architecture/Archi/Archi -application com.archimatetool.commandline.app \
  -consoleLog -nosplash \
  --loadModel docs/architecture/v21-application-integration-20260714-0130-claude.archimate
```

- [x] 退出码 0，输出 `Loaded model: 'AI Dev Platform — Application Integration v21 Target'` — 证据 `05_archi.txt` / `archi-v21-load.log`
