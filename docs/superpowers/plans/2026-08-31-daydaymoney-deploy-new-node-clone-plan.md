# Plan: 新节点 clone + 只拷 conf-local 可装配

- **日期**: 2026-08-31
- **设计**: `docs/superpowers/specs/2026-08-31-daydaymoney-deploy-new-node-clone-design.md`

无 MQ 任务。无 Python HTTP 接口。

## Tasks

- [x] T1 红：`test_up_from_config_repo.py` — 只 overlay `conf-local/`；不 rsync `taskGateway`/`db`；clone 根已有 conf-local 时保留；缺 `bin/runAll` 且 deploy-sync 失败则非 0（或明确失败日志+exit）
- [x] T2 绿：改 `up-from-config-repo.sh`（header 注释、overlay、可选 `infra-host.env`、bootstrap 失败可见）
- [x] T3 红：taskAuth `TestOidcSigningKeyDefaultPath` 期望 conf-local 路径；新增 DEPLOY_MODE 缺钥失败、旧路径一次性迁入
- [x] T4 绿：`jwt.go` `defaultSigningKeyPath` + `loadOrGenerateKey`
- [x] T5 红：`taskGateway` `setup-tls` 测例（conf-local staging、DEPLOY_MODE 缺 PEM 失败、SAN 用 INFRA_HOST）
- [x] T6 绿：`taskGateway/run.sh` `setup_tls`
- [x] T7 `conf-local.example` 占位 PEM + `infra-host.env`；extract 脚本提示树外 PEM 迁 conf-local
- [x] T8 本机 gitignore 路径拷入现网 PEM（不进 git）
- [x] T9 更新 secrets.example README；re-seed `.daydaymoney-deploy-seed`；seed README clone 清单
- [x] T10 构建含 MergeConfLocal 的 ELF，打 GitHub Release 新 tag，改钉 `envs/current/releases.yaml` / seed

## 验收命令

```bash
python3 -m pytest runAll/scripts/tests/test_up_from_config_repo.py -q
python3 -m pytest taskGateway/scripts/test_setup_tls.py -q
cd taskAuth && go test ./src -count=1 -run 'OidcSigningKey'
bash scripts/seed-daydaymoney-deploy.sh
diff -q runAll/scripts/up-from-config-repo.sh .daydaymoney-deploy-seed/scripts/up.sh
```
