# conf-local.example

本目录是 **`conf-local/` 键名骨架**，不含真实密钥。加载顺序只有：

1. `conf/<app>/config.yaml`（及同目录 GENERATED 片段）
2. `conf-local/<app>/config.yaml`（同相对路径）

加载器**不读** `config.local.yaml`。

1. 把同相对路径文件拷到仓库根（或 `$DEPLOY_ROOT`）的 `conf-local/`。
2. 网关 TLS：`conf-local/gateway/task-gateway/dev-gateway.pem` 与 `dev-gateway-key.pem`。
3. OIDC 签名钥：`conf-local/auth/task-auth/oidc_signing_key.pem`。
4. 本机 IP：`conf-local/infra-host.env` 设 `INFRA_HOST=`。
5. `conf-local/` 已在根 `.gitignore`，禁止提交。
6. 本机 GitLab：在 `conf-local/infra/git-service/config.yaml` 设 `runAllStartEnabled: true`。

抽取本机已跟踪密钥：`python3 runAll/scripts/extract_conf_secrets_to_local.py`
