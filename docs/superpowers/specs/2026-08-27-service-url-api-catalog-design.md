# Grafana 能否梳理全站网址 / 接口依赖？方案比较与落地

- **Status:** accepted
- **Date:** 2026-08-27
- **Decision:** Grafana `:3000` 作为**浏览面 + 运行时叠加**，不是网址/接口 SSOT。

## 问题

1. 能否用 `http://10.2.150.68:3000/` 梳理全站网址、接口关系，以便查看各服务职能、依赖、是否核心路径？
2. 这是不是最优？有没有更优方案？

## 结论

**能用 Grafana 看「实际怎么跑」，不能靠 Grafana 当「网站有哪些 URL」的目录。**

`:3000` 已是 Grafana 11，数据源 Prometheus / Loki / Tempo。现成能力：

| 已有面板 | 回答的问题 | 答不了的问题 |
|---|---|---|
| Distributed Trace View → Tempo service map | 最近一段时间**实际发生**的服务调用图 | 零流量的既定路由、SPA 页面、核心 vs 辅助的业务判定 |
| HTTP API Latency / Lightweight APM | 经验热路径（QPS、延迟、错误率） | 意图拓扑；未打点或未采样的路径会消失 |
| Explore / traces | 单次请求 span 树 | 全站网址清单 |

核心路径有两层，不能混：

- **Declared core** — 价值流上的主旅程（登录、计费、任务、项目、算力）。来自配置，与流量无关。
- **Empirical hot** — 现网 QPS 高的路径。来自 Prometheus。健康检查也可以很「热」。

Grafana 单独做目录会把「没流量 = 不存在」和「QPS 高 = 核心」混在一起。

## 方案比较

| 方案 | 优点 | 缺点 | 判定 |
|---|---|---|---|
| A. 只把 Grafana 当目录 | 运维已打开 `:3000` | 无既定 URL SSOT；冷路径消失；不能标业务核心 | 否决作 SSOT |
| B. 只写 Markdown / ArchiMate | 人读、可评审 | 易漂移；无运行时热度 | 必要但不充分 |
| C. 新服务画拓扑（Kiali / 自建 UI） | 交互强 | 新运行时、与仓内 SSOT 重复 | 过重 |
| D. **配置 SSOT 生成目录 + Grafana 叠加运行时** | 网关/owner/前端已是权威；`:3000` 继续当浏览面；CI `--check` 防漂移 | 声明式核心表需维护 | **采用** |

仓内已有碎片，缺的是拼起来的目录：

- `taskGateway/routes/routes.yaml` — 公网 URI → upstream
- `db/api_route_ownership.yaml` — 前缀 → owner
- taskFE `router/{public,tenant,admin}Routes.js` — SPA 路径
- `conf/value-stream.yaml` — 业务价值流（核心判定的语义来源）
- Tempo / `*_http_requests_total` — 运行时

## 决策（We will）

1. **SSOT**：生成 `docs/architecture/service-url-api-catalog.{md,json}`。
2. **浏览面**：Grafana dashboard uid `service-url-api-catalog`（Tempo service map + Prometheus 热路径 + 按服务汇总）。
3. **核心路径**：生成器按前缀声明 `core | supporting | ops | internal | orphaned`；Grafana 只回答 empirical hot。
4. **防漂移**：`python3 db/scripts/ci/build_service_url_api_catalog.py --check`。

不引入新的拓扑微服务，不把 Grafana JSON 手改成目录正文。

## 入口

- Grafana: http://10.2.150.68:3000/d/service-url-api-catalog/service-url-api-catalog
- 人读: `docs/architecture/service-url-api-catalog.md`
- 机器: `docs/architecture/service-url-api-catalog.json`
- 再生: `python3 db/scripts/ci/build_service_url_api_catalog.py`
