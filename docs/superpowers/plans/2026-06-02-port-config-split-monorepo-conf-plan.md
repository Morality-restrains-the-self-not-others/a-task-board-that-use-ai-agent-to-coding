# Implementation Plan: port_config → monorepo conf/

**Goal:** 用 `<monorepo>/conf/<app>/` 替代 `task2app/conf/port_config.json`，直读 + manifest sync。

**Status:** Phase 1–4 已完成（2026-06-02）

---

### Task 1: 迁移脚本与 conf 树

- [x] **1.1** `scripts/migrate_port_config_to_conf.py` 从 JSON 生成各 app `config.yaml` 与 `domain-events/<event>/config.yaml`
- [x] **1.2** `scripts/conf-sync.py` + `conf-sync-all.sh` + 各 app `sync.sh` / `sync.manifest.yaml`
- [x] **1.3** `conf/core/django/config.test.yaml`；`conf/README.md`

**Verify:** `python3 scripts/migrate_port_config_to_conf.py && ./scripts/conf-sync-all.sh`

---

### Task 2: Python 加载器（Saas_project）

- [x] **2.1** `task2app/Saas_project/config/conf_loader.py`（monorepo root、`load_app_config`、碎片 camelCase）
- [x] **2.2** 重写 `port_config.py` 使用 conf 树（删除 JSON 路径）
- [x] **2.3** `test_phone_config.py` → `conf/core/django/config.test.yaml`
- [x] **2.4** `test_port_config_merge.py` / `test_domain_events_port_config.py` 更新路径

**Verify:** `cd task2app/Saas_project && pytest tests/test_phone_config.py tests/test_port_config_merge.py tests/test_domain_events_port_config.py -q`

---

### Task 3: taskEvents Go

- [x] **3.1** `FindMonorepoRoot` → `conf/core/django/config.yaml`
- [x] **3.2** `Load()` / `intentFromPortConfig` → `conf/domain-events/`
- [x] **3.3** `config_test.go` 更新 fixture 路径

**Verify:** `cd taskEvents && go test ./config/... -count=1`

---

### Task 4: 其他消费者与清理

- [x] **4.1** gitOauth、vite、taskAuth Go、shell 脚本关键路径
- [x] **4.2** 删除 `task2app/conf/port_config.json*`、`conf/port_config.test.json`
- [x] **4.3** 更新文档指向 `conf/README.md`

**Verify:** `rg 'port_config\\.json' --glob '*.{py,go,js,mjs,sh}' --glob '!task2app-wt-*' --glob '!docs/**' --glob '!scripts/migrate*'` 仅剩 docstring / 迁移脚本
