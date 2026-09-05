# 测试意图：部署机 9999 编排源码编译 + 增量安装 + 精准重启

## 测试目标

验证 `DEPLOY_MODE=1` 时页头两个按钮走「源码编译 → 增量安装 →（精准）重启」，且失败/缺配置不静默成功。

## 测试分层

| 层 | 内容 |
|----|------|
| 单元 | Go：`DEPLOY_MODE` 下 PreciseRestart/BuildAll 调 source compile + install 钩子；缺 `SOURCE_ROOT` 失败；编译失败不 restart |
| 脚本 | 增量 install：同 size/mtime 跳过；变化文件 copy-then-mv；不碰 `conf-local/` |
| 范围外 E2E | 真 `go build` 与真 9999 点击（本机验收清单） |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `DEPLOY_MODE=1` + 登记 `task-auth` + 假 compile 成功 | 调用 source compile（参数含该服务）→ 增量 install → `restartService`；不跑部署树 `build.sh` |
| T2 | 假 compile 非零 | 不 restart、登记保留、进度 failed |
| T3 | `SOURCE_ROOT` 空 | PreciseRestart/BuildAll 失败，错误含 `SOURCE_ROOT` |
| T4 | 空登记 | 不 compile、不 install；`POST /api/precise-restart` 返回 200 `{status:empty}`（不是 400「失败」）；点击时 `GET ?fill_from_scan=1` 可从脏工作树补登记 |
| T5 | BuildAll `DEPLOY_MODE=1` | `--all` compile + install；**零**次 `restartService` |
| T6 | 产物未变 | install 日志 `unchanged`，不 `mv` ELF |
| T7 | 产物变了且 dest 为正在「执行」的假 ELF | copy-then-mv 成功（新 inode） |
| T8 | 编译成功 | dest `conf-local/` 与 `$SOURCE_ROOT/conf-local/` 一致（rsync --delete） |
| T8b | 编译失败 | dest `conf-local/` 内容不变 |
| T9 | SSE/进度 | 现有 precise-restart / build-all 事件仍可订阅；失败响应带 trace |

## 数据与环境

- 隔离临时 `SOURCE_ROOT` / `DEPLOY_ROOT`；假 `build.sh` 只 `touch` ELF。
- `t.Setenv("DEPLOY_MODE","1")` 与 `SOURCE_ROOT`；`TestMain` 仍须清泄漏的 `DEPLOY_MODE`（既有 `deploy_layout_test.go`）。

## 可执行

- `go test ./runAll/src -count=1 -timeout 120s -run 'SourceCompile|DeployModePrecise|DeployModeBuildAll'`
- `python3 -m pytest runAll/scripts/tests/test_install_local_artifacts.py -q`（增量 skip）
- 另测 conf-local rsync：成功对齐、失败不写

## 通过标准

上表 T1–T9 全绿；本机 `http://192.168.1.10:9999/` 对已登记服务点一次精准编译重启，只重启这些服务且 runAll 编排器 PID 不变。

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 空登记 POST 200 `{status:empty}`；点击 `fill_from_scan` 补脏树登记 |
