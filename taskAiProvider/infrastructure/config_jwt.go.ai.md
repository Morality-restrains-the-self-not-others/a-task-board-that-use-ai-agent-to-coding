# config_jwt.go Companion

SSO JWT 从 `confload.ReadAppConfig("core/sso")` 读取（含 conf-local 合并）。禁止 `os.ReadFile` 直读 `conf/core/sso/config.yaml` 取密钥。
