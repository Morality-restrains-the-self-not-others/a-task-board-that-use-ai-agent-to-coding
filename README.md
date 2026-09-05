# 云端 Coding（Daydaymoney）

Copyright (c) 2025～2026 ljy124818167@qq.com

面向团队的 **AI 云端研发工作台**：在真实网页捕获问题 → 工作面板任务 → 云端智能体按评论执行 → 结果回到 Git。

- 产品介绍：[docs/product-introduction.md](./docs/product-introduction.md)
- 站点：[https://www.daydaymoney.com](https://www.daydaymoney.com)
- 免责声明：[DISCLAIMER.md](./DISCLAIMER.md)

---

## 使用说明

本仓库是 **源码 monorepo**（含多个独立子仓）。日常开发可在本树编译运行；生产推荐按 ADR-0052 走独立部署仓 + 二进制产物，不必在部署机放整棵源码树。

### 1. 环境要求

- Linux 开发机（推荐），Docker，Go 1.24+，Node.js（前端构建）
- 可用的本机 IP（基础设施与网关寻址用 `INFRA_HOST`）

### 2. 获取代码

```bash
git clone <本仓库 URL>
cd ram-mount-AGPL   # 或你的本地目录名
# 子仓按 .gitmodules 独立 clone；可用项目既有脚本批量拉取，勿假设单次 --recurse-submodules 即可齐套
```

### 3. 配置（机密只放 conf-local）

已跟踪的 `conf/` 只含非机密骨架。本机密钥与覆盖必须放在仓库根 `conf-local/`（已 gitignore）：

```bash
# 从示例骨架拷贝相对路径，再填入真实值
cp -a conf-local.example/. conf-local/
# 至少配置：
#   conf-local/infra-host.env          → INFRA_HOST=<本机 IP>
#   conf-local/gateway/task-gateway/   → TLS PEM（见 conf-local.example 说明）
#   conf-local/auth/task-auth/         → OIDC 签名钥等
```

细则见 [conf-local.example/README.md](./conf-local.example/README.md)。**禁止**把密钥写进已跟踪的 `conf/` 或提交 `conf-local/`。

### 4. 启动基础设施与编排

```bash
# 可选：手工起 Redis / Kafka / MySQL（也可由 runAll 编排）
bash dockerInfra/redis/run.sh start
bash dockerInfra/kafka/run.sh start
# MySQL 见 dockerInfra/mysql/

# 编译并启动编排器（Web UI 默认 :9999）
cd runAll && ./run.sh
```

打开 `http://<本机 IP>:9999/`：

1. 按依赖启动 infrastructure / platform 等服务组  
2. 空库时点击 **「初始化全部数据库」**（确认码按页面提示）  
3. 再启动业务服务；数据库迁移**不要**指望业务进程启动时自动执行

更细的编排参数见 [runAll/README.md](./runAll/README.md)、主配置 `conf/runAll.yaml`。

### 5. 二进制部署（推荐生产路径）

源码机编译产物后，在部署机使用 `daydaymoney-deploy`（见 `.daydaymoney-deploy-seed/README.md`）：

1. 源码侧：`bash scripts/precise-compile.sh` / `collect-deploy-binaries.sh` 产出 ELF 与前端包  
2. 部署机：`rsync` `conf-local/` 与 `deploy-binaries/` → `./scripts/up.sh` → `./runAll/run.sh`  
3. 同样经 `:9999` **初始化全部数据库** 后再启业务进程  

### 6. 最终用户（不自建）

若只使用托管站点，无需克隆本仓库：注册、入驻、购买配额、安装 Chrome 扩展等步骤见 [docs/product-introduction.md](./docs/product-introduction.md)「如何开始」。

---

## 许可

本仓库以 **GNU Affero General Public License v3.0 (AGPL-3.0)** 授权。完整条款见 [LICENSE](./LICENSE)。

- 您可以在遵守 AGPL-3.0 的前提下使用、修改、分发本软件（含商业使用）。
- 若通过网络提供本软件的修改版本，必须向用户提供对应完整源代码（AGPL 第 13 条）。
- 若您需要一份**不附带 AGPL copyleft 义务**的专有许可，请向版权方购买商业授权，见 [COMMERCIAL.md](./COMMERCIAL.md)。

商务联系：ljy124818167@qq.com
