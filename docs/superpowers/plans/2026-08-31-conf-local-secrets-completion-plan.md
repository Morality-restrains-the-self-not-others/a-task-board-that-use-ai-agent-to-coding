# Plan: conf-local 唯一 overlay

- **日期**: 2026-08-31
- **设计**: `docs/superpowers/specs/2026-08-31-conf-local-secrets-completion-design.md`

## Tasks

- [x] T1 Go：`ReadAppConfig` 旁路 `config.local.yaml` 测例（空串不得抹 conf-local）
- [x] T2 Go：`ReadAppFragment` 旁路 `sms.local.yaml`
- [x] T3 实现：删除 Go merge `*.local.yaml`
- [x] T4 Python `load_app_config` 同等测例 + 删除 merge
- [x] T5 gitService YAML 兜底只合 conf + conf-local；`runAllStartEnabled` 改测 conf-local
- [x] T6 抽取脚本：不写/不保留 `.local.yaml`；conf-sync 不再生成 dest `.local.yaml`
- [x] T7 `up-from-config-repo.sh` / `prepare-ram-deploy.sh` 只 overlay `conf-local/`
- [x] T8 CI 拒绝跟踪 `*Pwd.md`；从 conf 子仓删除 `gitLabRootPwd.md`（历史泄露仍须 OPS 轮换）
- [x] T9 `conf-local.example` 键名骨架；文档/ADR 去掉「本机覆盖进 config.local.yaml」
- [x] T10 本机残留 `*.local.yaml` 非机密键迁 conf-local 后删除

无 MQ 任务。
