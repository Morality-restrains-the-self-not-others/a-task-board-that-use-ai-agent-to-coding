# 功能意图：源码机精准编译写入 deploy-binaries

## 用户故事

作为在源码仓工作的开发者，我需要一条命令，按「精准编译」登记（或指定服务）在**源码树**执行 `build_command`，把 ELF / 前端包写入 gitignored 的 `$META_ROOT/deploy-binaries`（本机即 `/tmp/ram-work/deploy-binaries`），供 clone-run 手拷或 `install-local-artifacts.sh` 使用。部署机（`DEPLOY_MODE=1`）仍禁止编译。

## 背景

ADR-0052 把源码与部署分开：部署根只有 `bin/` 与配方。`collect-deploy-binaries.sh` **只拷已有产物**，不编译。9999「精准编译重启」在部署机上因 `resolveBuildCommand` 清空而无编译。源码机缺少「只编译、写入 deploy-binaries、不重启」的入口。

本地新编 ELF 的 SHA 必然与 `conf.example/releases.yaml` 的 GitHub pin 不同；编译路径必须跳过该校验（`COLLECT_SKIP_SHA=1`）。

## 验收标准

1. `bash scripts/precise-compile.sh` 默认读 `.runall/precise_restart_services.txt`；可传服务 `name` 或 `working_dir` 别名；`--all` 编译全部带 `build_command` 的服务。
2. 在源码 `working_dir` 执行 `build_command`；无 `build_command` 的项跳过不失败。
3. 编译成功后归集到 `$META_ROOT/deploy-binaries`（可用 `--dest` 覆盖），写出 `MANIFEST.txt`。
4. 不调用 stop/start，不改登记文件（ADR-0027：本脚本只负责产物面）。
5. 即使 `releases.yaml` pin 与新 ELF 不一致，编译路径归集仍成功。
6. 无登记且未传名字、未 `--all` 时非零退出。

## 范围

- `runAll/scripts/precise-compile.sh`、`precise_compile.py`、meta wrapper
- `collect-deploy-binaries.sh` 的 `COLLECT_SKIP_SHA`
- 范围外：部署机 `go build`、GitHub Release 上传
- 部署 9999 按钮编排本脚本：见 `deploy_9999_source_compile_restart.intent.md`（本意图仍只覆盖源码机 CLI）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 例外理由 |
|---------|--------|--------|----------|
| 源码机编译产物落入 deploy-binaries | — | — | 运维交付，不产生业务领域事实 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
| 2026-09-02 | 交叉引用：9999 部署模式编排见 `deploy_9999_source_compile_restart` |
