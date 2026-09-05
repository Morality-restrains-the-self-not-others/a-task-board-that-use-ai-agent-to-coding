# 网络拓扑感知服务配置（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-29
- 最后修改：2026-07-29
- 维护者：Trae AI
- 约束索引编号：第 37 条
- Cursor 规则：`.cursor/rules/network-topology-aware-config.mdc`（alwaysApply）
- 关联规则：[服务监听禁止仅绑 127.0.0.1](./22_service_listen_host_not_localhost_only.md)、[服务仅读本目录配置](./29_service_own_conf_directory_only_via_sync.md)、[异域前端与 API 域名分离](./17_cross_domain_frontend_api.md)

## 动机

当前 monorepo 所有服务部署在单机（`127.0.0.1`），但随着业务增长，必然面临以下演进：

1. **内网服务独立**：Redis、Kafka、MySQL 等基础中间件需拆分到独立的内网节点（高可用集群）
2. **对外服务外迁**：GitLab 等重资源服务拆分到不同云服务商平台（如自建机房 → 云主机 / SaaS）
3. **业务服务横向扩展**：Go 业务服务按限界上下文拆分到不同机器节点

**当前反模式**：配置文件中大量硬编码 `127.0.0.1` / `localhost` 作为基础设施地址，导致未来拆分时需逐服务修改配置，极易遗漏。

## 核心规则

### 1. 网络拓扑分层分类（一级，禁止忽略）

所有服务必须按**网络可达性**显式分类为以下三个层级：

| 层级 | 说明 | 典型服务 | 寻址方式 |
|------|------|----------|----------|
| **内网基础设施层** (`internal-infra`) | 仅内网可达，不暴露公网入口 | Redis、Kafka、MySQL | `${INFRA_HOST}` 或内网 DNS 子域 |
| **外部服务平台层** (`external-platform`) | 部署在第三方平台 / 独立云主机 | GitLab、Portainer、Grafana | `${subdomains.xxx}` 或独立域名 |
| **业务网关层** (`business-gateway`) | 经 APISIX/nginx 对外暴露 API | taskAuth、taskGateway、Django | `${subdomains.xxx}`（已有模式） |

### 2. 配置文件中禁止硬编码基础设施地址（一级，禁止忽略）

**禁止**在以下场景中硬编码 `127.0.0.1` / `localhost` 作为基础设施服务的地址：

- ❌ `conf/infra/redis/config.yaml` → `host: 127.0.0.1`
- ❌ `conf/infra/kafka/config.yaml` → `host: 127.0.0.1`
- ❌ `conf/infra/mysql/config.yaml` → `host: 127.0.0.1`
- ❌ `conf/infra/docker-infra/config.yaml` → `redis.host: 127.0.0.1`、`kafka.host: 127.0.0.1`、`kafka.bootstrapServers: localhost:9093`
- ❌ 各 Go 服务配置中 `services.xxx.host: "127.0.0.1"`（指其他业务服务）
- ❌ Go 服务配置中 `kafkaBootstrapServers: localhost:9093`

**必须**使用以下寻址方式之一：

- ✅ `${INFRA_HOST}` — 基础设施宿主机 LAN/WAN IP（通过环境变量 `INFRA_HOST` 注入）
- ✅ `${subdomains.kafka}` / `${subdomains.redis}` / … — 经 `conf/base.yaml` 统一管理的 DNS 子域
- ✅ `host: 0.0.0.0` — **仅用于进程监听地址**（对外暴露端口，不表示连接目标）

### 3. 基础设施服务子域名注册（一级，必须）

`conf/base.yaml` 的 `subdomains` 节点**必须**为每类基础设施服务注册子域条目（即使当前指向同一主机）：

```yaml
subdomains:
  # ... 现有业务子域 ...
  # 基础设施内网子域（当前指向同一主机，未来拆分时仅修改此文件）
  infra:            infra.${baseDomain}        # 基础设施统一入口
  kafka:            kafka.${baseDomain}        # Kafka bootstrap（已存在但含义模糊）
  kafkaInternal:    kafka-internal.${baseDomain}  # Kafka 内网地址（broker 直连）
  redisInternal:    redis-internal.${baseDomain}  # Redis 内网地址
  mysqlInternal:    mysql-internal.${baseDomain}  # MySQL 内网地址
```

### 4. 同节点通信允许 loopback 优化（二级，推荐）

同一物理/虚拟节点上的**进程间通信**（如 Go 服务调用 Django internal API），仍允许使用 `http://127.0.0.1:PORT` 作为**客户端 URL**，但：

- 被调服务的 **listen host 必须为 `0.0.0.0`**（已有规则第 24 条）
- 配置中应使用 **`INTERNAL_LOOPBACK`** 常量或在注释中显式标注「同节点 loopback 调用」
- 当两个服务拆分到不同节点时，须将 loopback URL 改为 `${subdomains.xxx}` 或 `${INFRA_HOST}`

### 5. 新增服务配置检查清单

新增或修改服务配置时，必须逐项检查：

- [ ] 基础设施地址（Redis / Kafka / MySQL）未硬编码 `127.0.0.1`
- [ ] 使用 `${INFRA_HOST}` 或 `${subdomains.xxx}` 作为基础设施连接地址
- [ ] 监听地址为 `0.0.0.0`（非基础设施的客户端连接目标）
- [ ] 同节点 loopback URL 有注释标注，标注了拆分时需要改的变量
- [ ] `conf/base.yaml` 中已注册该服务所需的所有子域条目
- [ ] 跨服务配置通过 `sync.manifest.yaml` 同步片段，未直读他服务目录
- [ ] 未来拆分到不同节点的服务，其配置仅需修改 `base.yaml` 或环境变量即可切换

### 6. 存量配置迁移指引

现有配置文件中硬编码的 `127.0.0.1` 须在触及对应配置时逐步迁移（不强制一次性全部修改，但触及即改）：

| 现状 | 迁移方向 |
|------|----------|
| `redis.host: 127.0.0.1` | `${INFRA_HOST}` + 独立 port 或 `redisInternal` 子域 |
| `kafka.host: 127.0.0.1` | `${INFRA_HOST}` 或 `kafkaInternal` 子域 |
| `kafka.bootstrapServers: localhost:9093` | `${INFRA_HOST}:9093` 或 `${subdomains.kafkaInternal}:9093` |
| `mysql.host: 127.0.0.1` | `${INFRA_HOST}` 或 `mysqlInternal` 子域 |
| Go 服务 `services.xxx.host: "127.0.0.1"` | 同节点保持 loopback + 注释；异节点用子域 |
| `djangoInternalApi: http://127.0.0.1:8001` | 同节点保持 + 注释「拆分时改 `${subdomains.api}` 内部路由」 |

### 7. 基础设施宿主机地址注入

`${INFRA_HOST}` 变量由以下优先级解析（由 `confload` 统一处理）：

1. 环境变量 `INFRA_HOST`（最高优先级）
2. `conf/base.yaml` 的 `infraHost` 字段（fallback）
3. 当前默认值：`127.0.0.1`（仅开发环境，生产环境**必须**显式设置）

## 验收

```bash
# 1. 基础设施配置中不得硬编码 127.0.0.1（监听 host=0.0.0.0 除外）
grep -rn 'host:\s*"127.0.0.1"' conf/infra/ --include='*.yaml' && echo "FAIL: infra configs must use templates" || echo "PASS"

# 2. 服务间客户端地址若为 127.0.0.1 须有「同节点」注释
grep -rn '127.0.0.1' conf/ --include='*.yaml' | grep -v 'host: 0.0.0.0\|internalApi\|loopback\|同节点\|#\|_doc' && echo "WARN: undocumented 127.0.0.1 references" || echo "PASS"

# 3. base.yaml 须包含 infra 子域
grep -q 'infra:\|kafkaInternal:\|redisInternal:\|mysqlInternal:' conf/base.yaml && echo "PASS: infra subdomains registered" || echo "FAIL: missing infra subdomains"

# 4. 环境变量 INFRA_HOST 可正确解析（本地测试）
INFRA_HOST=10.0.0.1 python3 -c "
from shareLib.confload import resolve_template
print(resolve_template('\${INFRA_HOST}'))
" 2>/dev/null || echo "SKIP: confload not available in this context"
```

## 规则冲突处理

- 本规则与[服务监听禁止仅绑 127.0.0.1](./22_service_listen_host_not_localhost_only.md)互补：监听地址规则管**服务端 bind**，本规则管**客户端连接目标**
- 与[服务仅读本目录配置](./29_service_own_conf_directory_only_via_sync.md)联动：跨节点时服务发现也应通过本目录 conf 片段获取
- 与[应用启动禁止 Proxy](./23_app_startup_no_env_proxy.md)联动：出站直连不经过代理；`${INFRA_HOST}` 应直连可达

## 变更日志

- 2026-07-29：版本 1.0.0 — 初始创建；定义三层网络拓扑分类、禁止硬编码基础设施地址、基础设施子域名注册；存量迁移指引
