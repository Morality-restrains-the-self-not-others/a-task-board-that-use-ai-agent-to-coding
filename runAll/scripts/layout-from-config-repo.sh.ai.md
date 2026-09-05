# layout-from-config-repo.sh Companion

从 `daydaymoney-deploy` checkout 组装 `$DEPLOY_ROOT`。

- `DEPLOY_ROOT` 等于配置仓根：根目录对 `envs/<env>/` 做相对符号链接（clone 即部署根）。
- 否则 rsync 到独立部署根；**不**覆盖已有 `releases.yaml`、不碰 `dockerInfra/*/data`、不覆盖 `*.local.yaml`。
- 物化后若 `$DEPLOY_ROOT/dockerInfra/mysql/data` 是指向 `/tmp/ram-work` 或损坏空壳的符号链接，必须拆掉（OPT-20260830-025），禁止 P4 切流绑到源码树空壳。
- `taskGateway/logs` 必须 0777（APISIX uid 636）。
- clone-as-root **禁止** sed git 跟踪的 `envs/<env>/conf/runAll.yaml`。始终把 `logging.file_root` 写入 gitignored 的 `$DEPLOY_ROOT/conf-local/runAll.yaml`（深合并，保留已有密钥键）；独立部署根仍会 sed 拷贝后的 YAML。
- 测试必须 unset 继承的 `DEPLOY_ROOT`（例如本机 `cutover.env`），否则会 rsync --delete 进 live 部署根。
