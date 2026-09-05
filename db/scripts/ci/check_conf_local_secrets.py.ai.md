# check_conf_local_secrets.py Companion

已跟踪 `conf/` YAML 不得含非空机密键（含 `internalSecret`、`host_password`、后缀 `*Secret`/`*PASSWORD`）。`${VAR:-}` 插值不算。禁止打印密钥值。
