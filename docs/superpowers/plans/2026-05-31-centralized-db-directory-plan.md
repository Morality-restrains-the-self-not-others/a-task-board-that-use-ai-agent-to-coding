# Implementation Plan: 集中式 SQLite 目录 `db/`

## Tasks

- [ ] **T1** 创建 `db/registry.yaml`、`README.md`、`.gitignore`、子目录 `.gitkeep`
- [ ] **T2** 实现 `db/load/registry_loader.py` + 单元测试
- [ ] **T3** 实现 `db/load/registry.go`（Go module `dbload`）+ 测试
- [ ] **T4** `paths_loader.db_sqlite_path()` → registry `saas`
- [ ] **T5** Django `settings.py` taskauth/taskbill 路径 → registry
- [ ] **T6** taskAuth/taskBill `config.go` → dbload
- [ ] **T7** gitOauth / Saas_email / Saas_Ai_Provider settings → registry
- [ ] **T8** `db/migrate-existing.sh` + 更新迁移脚本默认路径
- [ ] **T9** 移除 `port_config.json` databasePath、`paths.conf` DB_SQLITE_PATH
- [ ] **T10** 更新 `billing_bridge/referral_stats.py`、conftest、文档引用
- [ ] **T11** 运行 migrate + pytest 验证
