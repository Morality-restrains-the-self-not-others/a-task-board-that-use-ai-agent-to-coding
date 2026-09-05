# load.go Companion

与目录 [ai.md](./ai.md) 相同约束：服务仅读本 `conf/<app>/`；跨服务配置须 sync 后 `ReadAppFragment`。

SSOT：`.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md`。

变更 `ReadAppConfig` / `ReadAppFragment` 时不得削弱「片段必须在本 app 目录内」的路径校验。读完 `conf/<rel>` 后须深合并 `conf-local/<rel>`（ADR-0054）。禁止再合并 `config.local.yaml` / `*.local.yaml`。
