# Monorepo runtime configuration (`conf/`)

Each deployable app has `conf/<app>/config.yaml` (authoritative). Cross-app values are copied into **GENERATED** fragment files via `sync.manifest.yaml` + `sync.sh`.

## Workflow

1. Edit `conf/<owner>/config.yaml` for values you own.
2. Run `./runAll/scripts/conf-sync-all.sh` (or `conf/<app>/sync.sh`).
3. Do **not** hand-edit `*.yaml` fragments with a `# GENERATED` header.

## Layout

- `core/django/` — Saas_project (Django)
- `frontend/vue/` — taskFE (Vite)
- `infra/docker-infra/` — local machine Redis/Kafka SSOT; synced to domain-events / task-sse / django fragments
- `events/domain-events/` — transport/redis/kafka + `<event>/config.yaml` for taskEvents intents
- `auth/git-oauth/providers/*.yaml` — per Git site OAuth
- See design: `docs/superpowers/specs/2026-06-02-port-config-split-monorepo-conf-design.md`

## Tests

- SMS test numbers: `conf/core/django/config.test.yaml` (`test_phone_numbers`)

## Local overrides & secrets (D2=A)

- Copy `conf/core/django/config.example.yaml` → `conf/core/django/config.local.yaml` for machine-specific values.
- `conf/**/config.local.yaml` is gitignored; deep-merges over `config.yaml`.
- CI runs `python3 db/scripts/ci/check_conf_sync.sh` — entire `conf/` must match `runAll/scripts/conf-sync-all.sh` output.
- Tracked files in this repo must be YAML / companion / `sync.sh` / license / a thin `.githooks/pre-commit`. Gate: `python3 db/scripts/ci/check_conf_tracked_allowlist.py`.

## CLI

```bash
python3 runAll/scripts/conf-read.py django port
python3 runAll/scripts/conf-read.py snapshot-json   # Playwright/Vite tooling only
```

Legacy app aliases (`django`, `vue`, `aiProvider`, …) are resolved by `load_app_config()` to the paths above.

## Deprecated

- `task2app/conf/port_config.json` — removed after migration
- `conf/port_config.test.json` — replaced by `conf/core/django/config.test.yaml`
- `conf/django/` — moved to `conf/core/django/` (2026-06 conf restructure)

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。不附带 AGPL 义务的专有许可见 [COMMERCIAL.md](./COMMERCIAL.md)。
