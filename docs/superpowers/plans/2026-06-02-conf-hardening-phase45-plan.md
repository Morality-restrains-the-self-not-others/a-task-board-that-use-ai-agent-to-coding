# Plan: conf 硬化 Phase 4.5

**Status:** 已实施（auto-flow 2026-06-02）

## 4.5a — CI + conf-read + write/sync + go

- [x] `scripts/ci/check_conf_sync.sh` + 接入 `check_ddd_bdd_compliance.py`
- [x] `scripts/conf-read.py`
- [x] `write_port_config` 末尾 conf-sync
- [x] `go_run_container` 删除死代码 `configFilePath`
- [x] `conf-sync.py` 内容未变时跳过写入（CI 稳定）

## 4.5b — Saas 直读

- [x] `settings.py` / `daydaymoney_nginx_conf` 直读 `load_app_config`
- [x] `port_config.read_port_config` 直读各 app（无 `assemble_legacy`）
- [x] 删 `load_django_fragments`；单一 `build_runtime_snapshot()`（`read_port_config` / conf-read 共用）
- [x] `settings` 去掉二次 `read_port_config`；`ai_provider` / `health` / redis publisher 窄读
- [x] `tests/test_conf_read.py`

## 4.5c — 前端/E2E

- [x] `loadConfYaml.mjs` + 全 Playwright 改用
- [x] Vite / taskSSE 用 `conf-read.py snapshot-json`
- [x] 删除 `conf_emit_json.py`、`loadPortConfig.mjs`

## 4.5d — 文档

- [x] `conf/core/django/config.example.yaml`；`conf/README.md`；`.gitignore` local
- [ ] `task2app-wt-relay-stop/` — 工作树目录当前不存在，D3 待目录恢复后 rsync `task2app/playwright` 等

## 4.5e — value-stream + 验证

- [x] `value-stream.yaml` 描述更新
- [x] pytest 27 passed；go test ok；`check_conf_sync` ok
