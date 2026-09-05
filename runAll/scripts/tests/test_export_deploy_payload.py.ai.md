# test_export_deploy_payload.py Companion

断言 `export-deploy-payload.sh` 写出 `secrets.example/README.md`（conf-local 放置方法），且不再复制 HOST_SECRETS 登记册。`.gitignore` 含 `/conf-local/`、`/deploy-binaries/`、`docker/code-server/*.tar.gz`、`/taskAiProvider/frontend/dist/` 与 `/taskFE/app/public/`；导出树不含 code-server 官方 tarball。`scripts/install-local-artifacts.sh`、`up.sh` 与 `write-cutover-env.sh` 一并导出；`up.sh` 不得再内嵌 `cutover.env` heredoc。
