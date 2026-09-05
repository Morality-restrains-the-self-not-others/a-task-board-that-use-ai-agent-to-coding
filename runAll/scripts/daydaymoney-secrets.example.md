# 主机密钥（手工放置，不进 Git）

克隆 `daydaymoney-deploy` 之后，把**两棵**树拷到 clone 根：

1. `conf-local/`（YAML + 网关/OIDC PEM）— 与 `conf/` 同相对路径
2. `deploy-binaries/` → `./artifacts/`（Go ELF + `taskEvents-bin.tar.gz` + `taskFE-dist.tar.gz`）

源码仓用 `bash scripts/collect-deploy-binaries.sh` 归集第 2 棵树。`./scripts/up.sh` 从 `artifacts/` 安装，不再默认下载 GitHub Release。

`conf-local/` 必须包含：

- 各 `conf-local/<area>/<app>/config.yaml`（支付、SSO、SMS、OAuth…）
- `conf-local/gateway/task-gateway/dev-gateway.pem` 与 `dev-gateway-key.pem`
- `conf-local/auth/task-auth/oidc_signing_key.pem`
- `conf-local/infra-host.env`（`INFRA_HOST=<本机可达 IP>`；与源机同 IP 可原样拷）

`./scripts/up.sh` 会 overlay clone 根或 `secrets/conf-local/`。`DEPLOY_MODE=1` 时缺网关/OIDC PEM **不会**自签或 mint。

禁止再维护 HOST_SECRETS 登记册。禁止 rsync `secrets/taskGateway` 或 `secrets/db`。
