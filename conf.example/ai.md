# conf.example

源码仓内的配置 **schema / 示例**（ADR-0052 P3）。

- 运行时 SSOT：私有仓 `github.com/task2money/daydaymoney-deploy`（`envs/<env>/conf/`）。
- 本机仍可双写 monorepo `conf/` 子仓；**禁止**把 `config.local.yaml` 或生产密钥放进本目录。
- 部署机：`DEPLOY_MODE=1` + `CONF_ROOT` / `DEPLOY_ROOT`，`runAll -command deploy-sync`。
- 部署根还须有 `db/registry.yaml`（DSN SSOT；含口令时只放主机、不进配置仓 Git）和 `$DEPLOY_ROOT/bin/`（deploy-sync 产物）。
