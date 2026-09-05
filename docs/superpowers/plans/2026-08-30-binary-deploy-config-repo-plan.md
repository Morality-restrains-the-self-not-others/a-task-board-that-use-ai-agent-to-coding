# 实施计划：二进制部署配置根（I1–I2）

- **Date:** 2026-08-30
- **Design / DDD / NFR / VS:** 同迭代 `binary-deploy-config-repo`

> 本会话只勾选 I1–I2（及 I3 骨架）。I4 迁出源码仓 conf、I5 无源码主机验收不在本次 ship 范围（现网双写）。

## 任务

- [x] T1 红：`CONF_ROOT` 指向含 `base.yaml` 的 conf 目录（无 `.gitmodules`）时 `FindConfigRoot` 返回其父目录
- [x] T2 红：`DEPLOY_ROOT/conf/base.yaml` 存在时返回 `DEPLOY_ROOT`；env 优先于 cwd 向上查找
- [x] T3 绿：`FindMonorepoRoot` 委托 `FindConfigRoot`；既有 marker 测例 unset env 后仍过
- [x] T4 红：`LoadReleases` 解析 `releases.yaml`；缺钉报错
- [x] T5 红：`SyncPinnedArtifacts` fake fetcher 失败时不覆盖已有 bin；成功则原子写入 + sha sidecar
- [x] T6 绿：sync 路径不调用 `go build`（字符串/接口断言）
- [x] T7 `shareLib/confload/ai.md` 记录 `CONF_ROOT`/`DEPLOY_ROOT`
- [x] T8 设计文档追加权限节；`conf.example/releases.yaml` 骨架

## 事件任务

- [x] 无 MQ — 意图例外已写

## 验证命令

```bash
cd /tmp/ram-work/shareLib/confload && go test -count=1 .
cd /tmp/ram-work/runAll && go test -count=1 ./src -run 'Pin|Sync|Releases|ConfigRoot'
```
