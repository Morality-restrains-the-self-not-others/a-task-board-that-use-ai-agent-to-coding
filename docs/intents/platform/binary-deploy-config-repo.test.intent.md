# 测试意图: 二进制部署；运行时配置与源码仓隔离

## 测试目标

证明部署根在**没有源码树**时仍能定位配置、exec 已钉二进制，且源码仓不再作为运行时 conf SSOT。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | `confload`：`CONF_ROOT`/`DEPLOY_ROOT` 优先于 `FindMonorepoRoot`；仅有 `conf/base.yaml`、无 `.gitmodules`/`dataMigrate` 时仍成功 |
| 单元 | 版本钉解析：`releases.yaml` 映射服务 → 产物坐标；缺钉/缺产物失败且不编译 |
| 集成 | 最小 fixture 部署根（bin + conf + 无源码）启动 stub 二进制读本目录 conf |
| 门禁 | 源码仓 `conf/` 不得再含生产 `runAll.yaml`（过渡期后）；配置仓结构校验 |

## 用例矩阵

| # | 用例 | 对应验收 | 期望 |
|---|------|----------|------|
| T1 | `CONF_ROOT` 指向仅含 `conf/base.yaml` 的目录 | 验收 1、2 | `FindConfigRoot` 返回该目录 |
| T2 | 未设 env 时仍可从 cwd 向上找到 `conf/base.yaml`（兼容开发树） | 开发回退 | 行为与现 `FindMonorepoRoot` 一致 |
| T3 | 部署根无 `go.mod`、无 `*.go` 时 start 不调用 `go build` | 验收 1、3 | 缺二进制则失败并提示拉产物 |
| T4 | `releases.yaml` 钉 SHA 与磁盘 `bin/` 不一致 | 验收 3 | deploy-sync 下载或拒绝启动；不 exec 错版本 |
| T5 | last-good：新产物下载失败 | 验收 3 | 仍 exec 旧二进制 |
| T6 | Docker 配方 `run.sh` 从 `DEPLOY_ROOT/conf/infra/mysql` 读 conf | 验收 2 | 不再依赖源码仓相对 `../../conf` 之外的路径约定以外的源码文件 |
| T7 | `DEPLOY_MODE=1` 无 `taskAuth/` 源码目录时仍 exec `$DEPLOY_ROOT/bin/taskAuth` | 验收 1 | cwd=部署根；`check_p4_deploy_root.sh` 通过 |
| T8 | 仅有 `daydaymoney-deploy` checkout + `scripts/up.sh`，无 ram-work 配方 symlink | 验收 1 | layout 出 conf/dockerInfra/dataMigrate；`check_p4` 在放入 bin 后通过 |
| T9 | `github://…@<Release tag>` deploy-sync 拉 ELF | 验收 3 | 干净目录能从 tag `deploy-20260831` 拉下 `runAll`/`valueStream`；内容 sha256 与 pin 一致 |
| T10 | archive pin 解压到 dest | 验收 1、3 | `taskEvents-bin` → `taskEvents/bin` 可执行 worker；`taskFE-dist` → `taskFE/app/public/html/index.html`（`html` → `releases/<stamp>`） |
| T11 | 已跟踪 conf YAML 无非空机密键；机密在 conf-local | 验收密钥不进 Git | `python3 db/scripts/ci/check_conf_local_secrets.py` 退出 0；`secrets.example/README.md` 描述 `conf-local/` 且不含 HOST_SECRETS 登记册 |

## 数据与环境

- Fixture：临时目录模拟 `$DEPLOY_ROOT/{conf,bin}`，不含源码。
- 不连真实 MySQL/Kafka；Docker 配方测试可用 dry-run / 解析 compose。

## 通过标准

- T1–T5 单测全绿；T6 至少脚本路径解析断言。
- 设计落地后 CI 增加「源码仓 conf 仅为 example」检查（过渡开关）。

## 业务意图 → 事件对照（测试）

| 业务意图 | 事件断言 | 例外理由 |
|---------|----------|---------|
| 部署机仅用二进制 + 独立配置仓运行 | 无 MQ 断言 | 运维交付，无领域事件 |
