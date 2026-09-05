# 实施计划：二进制部署 P1–P3

- **Date:** 2026-08-30
- **Depends:** I1–I2 已合入（FindConfigRoot、SyncPinnedArtifacts）

> 本机 runAll 继续读现网 `conf/` 子仓（双写）。**禁止**删除或掏空该子仓。P3 = 源码仓 schema/`conf.example` + 门禁，不是停机切流。P4 无源码主机不在范围。

## 任务

- [x] P1a 红：`seed-daydaymoney-deploy` 拷贝 conf 时排除 `config.local.yaml` / `*.local.yaml` / `.git`
- [x] P1b 绿：私有 GitHub 仓 `task2money/daydaymoney-deploy`（失败则本地 seed + 脚本可重试）
- [x] P2a 红：`DEPLOY_MODE=1` 时 `resolveBuildCommand` 恒为空（含推断 `./build.sh`）
- [x] P2b 红：`file://` fetcher + `github://owner/repo/asset@sha` 解析；token 不进日志
- [x] P2c 绿：`runAll -command deploy-sync` 不跑 `go build`；`scripts/deploy-sync.sh` / 发布脚本
- [x] P3a 红：`check_no_prod_conf_in_source.py` — 源码树禁止 `config.local.yaml`；`conf.example` 禁止真实密钥；跳过根 `conf/` 子仓
- [x] P3b 绿：元规则 42/47、`conf/ai.md`、ADR-0027 部署机 download-then-swap；`conf.example` schema

## 事件任务

- [x] 无 MQ — 既有意图例外

## 验证

```bash
cd /tmp/ram-work/runAll && go test -count=1 ./src -run 'DeployMode|PackageRef|DeploySync|FileArtifact|GitHubRelease'
python3 db/scripts/ci/test_check_no_prod_conf_in_source.py
python3 db/scripts/ci/check_no_prod_conf_in_source.py
bash -n scripts/seed-daydaymoney-deploy.sh runAll/scripts/deploy-sync.sh
```
