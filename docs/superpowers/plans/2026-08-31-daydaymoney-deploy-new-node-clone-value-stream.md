# Value Stream: 新节点 clone-run

- **日期**: 2026-08-31
- **状态**: skipped_non_product（平台交付，非 `value-stream.yaml` 业务流）

## 平台增量（单切片）

1. 运维 clone `daydaymoney-deploy`（seed）
2. 手工 rsync 完整 `conf-local/`（YAML + 网关/OIDC PEM + 可选 `infra-host.env`）
3. `./scripts/up.sh` layout + overlay + 拉 Release
4. 起 Docker 基础设施；`./runAll/run.sh`（source `cutover.env`）；空库 9999 INIT_ALL

不改业务域 streams / fields / 测试点图。
