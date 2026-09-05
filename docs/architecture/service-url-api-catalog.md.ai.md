# Companion — service-url-api-catalog.md

本文件与 `service-url-api-catalog.json`、Grafana dashboard `service-url-api-catalog` 均为 **生成物**。

- **禁止**手工编辑三份生成物来「补一条路由」。改 SSOT：`taskGateway/routes/routes.yaml`、`db/api_route_ownership.yaml`、`taskFE/app/src/router/*Routes.js`，或改 `db/scripts/ci/build_service_url_api_catalog.py` 的 `CORE_API_GLOBS`。
- 改完后执行：`python3 db/scripts/ci/build_service_url_api_catalog.py`
- CI：`python3 db/scripts/ci/build_service_url_api_catalog.py --check`
- Grafana `:3000` 只叠加运行时（Tempo / Prometheus），不是网址 SSOT。
