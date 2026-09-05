# DaydaymoneyGrafana 插件 404（plugin not found）修复设计

**日期：** 2026-07-05  
**状态：** 方案 B 已实施（插件降级适配 Grafana 11.5.2）  
**选定方案：** B — 降级插件以适配 Grafana 11.5.2（不升级 AiMonitor 镜像）
**触发页面：** [http://183.250.1.132:3000/plugins/daydaymoney-grafana-app?page=configuration](http://183.250.1.132:3000/plugins/daydaymoney-grafana-app?page=configuration)

## 问题现象

浏览器访问插件配置页时，Grafana 发起：

```http
GET /api/gnet/plugins/daydaymoney-grafana-app HTTP/1.1
Host: 183.250.1.132:3000
```

响应：

```json
{
  "code": "NotFound",
  "message": "plugin not found",
  "requestId": "10f48c79-466c-4d56-8601-1cf4bdaa3b55"
}
```

同时页面可能出现「Grafana has failed to load its application files」（静态资源/反向代理问题，与插件 404 可并存）。

## 根因分析

### RC-1：Grafana 主版本与插件声明不兼容（主因）

| 项 | 当前值 | 期望 |
|----|--------|------|
| AiMonitor 部署镜像 | `grafana/grafana:11.5.2` | 与插件 `grafanaDependency` 一致 |
| 插件 `plugin.json` | `grafanaDependency: ">=12.3.0"` | 须 ≤ 运行中 Grafana 主版本 |
| 插件构建依赖 | `@grafana/*: 12.4.2`（`package.json`） | 与运行时 Grafana 12.x API 对齐 |

Grafana 在扫描 `plugins/` 目录时，若 `grafanaDependency` 不满足，**不会注册该插件**，`/api/gnet/plugins/{id}` 即返回 404。这与 provisioning 中写了 `type: daydaymoney-grafana-app` 无关——插件本体未被加载。

### RC-2：部署链路未保证 dist 产物存在

`AiMonitor/docker-compose.yaml` 挂载：

```yaml
../DaydaymoneyGrafana/dist:/var/lib/grafana/plugins/daydaymoney-grafana-app:ro
```

但 `AiMonitor/run.sh` **未**在启动前执行 `npm run build`。若服务器上 `dist/` 为空、过期或缺失 `module.js`，同样导致插件扫描失败。

### RC-3：反向代理未配置 `root_url`（次要，解释静态资源失败）

响应头含 `Via: 1.1 grafana, 1.1 google`，说明 `:3000` 前可能有负载均衡/反向代理。未设置 `GF_SERVER_ROOT_URL` 时，Grafana 生成的静态资源 URL 可能错误，触发「failed to load its application files」。

### RC-4：CORS（已单独修复，非本 404 主因）

`:3000` → `:18081` 的跨域已在 `conf/gateway/task-gateway/config.yaml` 补充 `:3000` 源；与「plugin not found」无直接关系，但插件加载成功后配置页登录仍依赖该修复。

## 当前架构理解（基于 v5 shipped / v4 target）

> 根据 `docs/architecture/` 当前基线：
>
> - **可观测性栈**：Grafana (:3000) + Loki + Promtail + Tempo + Prometheus（AiMonitor）
> - **应用层**：task-gateway (:18081) → saas-backend、task-auth 等
> - **端点类组件**：架构 v1 已将「Grafana 插件」列为 Application Component（其他端点）
>
> 本次为 **AiMonitor 部署与插件版本对齐**，不改变应用集成拓扑；**不新增架构版本文件**（属配置/部署修复）。

📋 架构版本历史：当前 shipped 为 **v5**（relay lifecycle）；本次不 bump 架构版本。

## 价值流影响

| 问题 | 评估 |
|------|------|
| 影响现有 stream | `ai-monitor.runtime.lifecycle_status`、Grafana 相关 increment2/3 测试点 |
| 新 stream | 否 |
| 字段变更 | 无 |
| 测试影响 | 建议新增：插件注册 smoke（`GET /api/gnet/plugins/daydaymoney-grafana-app` → 200） |

## 方案对比

### 方案 A（推荐）：升级 AiMonitor Grafana 至 12.4.x

与插件 `@grafana/* 12.4.2` 及 `grafanaDependency >=12.3.0` 对齐。

**改动：**

```yaml
# AiMonitor/docker-compose.yaml
grafana:
  image: grafana/grafana:12.4.2   # 原 11.5.2
```

**优点：** 无需降级插件 API；与 create-plugin 模板一致。  
**风险：** 需验证现有 Dashboard JSON、数据源 provisioning 在 12.x 下兼容（通常向后兼容）。

### 方案 B：降级插件以适配 Grafana 11.5.2（**已选定并实施**）

将 `grafanaDependency` 改为 `>=11.5.0`，`@grafana/*` 依赖降至 11.5.x，重新 `npm run build`。

**优点：** 不动 AiMonitor 镜像。  
**缺点：** 工作量大；已用 12.x 组件 API 的代码可能需回退；长期技术债。

**已实施变更：**

| 文件 | 变更 |
|------|------|
| `DaydaymoneyGrafana/src/plugin.json` | `grafanaDependency: ">=11.5.0"` |
| `DaydaymoneyGrafana/package.json` | `@grafana/data/runtime/ui/schema` → `11.5.2`；移除 `@grafana/i18n` |
| `DaydaymoneyGrafana/.config/docker-compose-base.yaml` | 默认 `GRAFANA_VERSION` → `11.5.2` |
| `AiMonitor/run.sh` | 启动前 `build_daydaymoney_grafana_plugin`（有 `node_modules` 时仅 `npm run build`） |
| `DaydaymoneyGrafana/dist/plugin.json` | 已重建，`grafanaDependency: ">=11.5.0"` |

### 方案 C：仅修 dist 挂载、不升级 Grafana

**结论：** 单独执行无法解决 RC-1；**不可作为唯一方案**。

## 推荐实施方案（方案 B + 部署加固）

> **重要澄清**：`GET /api/gnet/plugins/{id}` 是 Grafana **Cloud 插件市场**代理，私有/未签名插件**始终可能 404**，不代表本地未注册。  
> 正确验收接口：`GET /api/plugins/daydaymoney-grafana-app/settings` → 200，且 Grafana 日志有 `Plugin registered pluginId=daydaymoney-grafana-app`。

### 1. 保持 Grafana 11.5.2 镜像（方案 B）

`AiMonitor/docker-compose.yaml` 维持 `grafana/grafana:11.5.2`。

保留：

```yaml
GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS: daydaymoney-grafana-app
volumes:
  - ../DaydaymoneyGrafana/dist:/var/lib/grafana/plugins/daydaymoney-grafana-app:ro
  - ./grafana/provisioning:/etc/grafana/provisioning:ro
```

### 2. 启动前构建插件 dist

在 `AiMonitor/run.sh` 的 `compose up` 之前增加（若 `dist/module.js` 不存在或源码新于 dist）：

```bash
build_daydaymoney_grafana_plugin() {
  local plugin_dir="$ROOT/../DaydaymoneyGrafana"
  if [[ ! -f "$plugin_dir/dist/module.js" ]] || [[ "$plugin_dir/src" -nt "$plugin_dir/dist/module.js" ]]; then
    echo "正在构建 DaydaymoneyGrafana 插件..."
    (cd "$plugin_dir" && npm ci && npm run build)
  fi
}
```

### 3. 反向代理环境变量（若经 LB 访问）

在 `docker-compose.yaml` 为 Grafana 增加（按实际公网 URL 填写）：

```yaml
environment:
  GF_SERVER_ROOT_URL: http://183.250.1.132:3000/
  GF_SERVER_SERVE_FROM_SUB_PATH: "false"
```

若未来挂子路径，再改 `root_url` 并设 `serve_from_sub_path: true`。

### 4. 验收清单

```bash
# 1) 插件目录在容器内可见
docker exec aimonitor-grafana ls -la /var/lib/grafana/plugins/daydaymoney-grafana-app/plugin.json

# 2) 本地插件 API（正确验收）
curl -s -u admin:admin http://localhost:3000/api/plugins/daydaymoney-grafana-app/settings | jq .id
# 期望: "daydaymoney-grafana-app"

# 3) Grafana 日志确认注册
docker logs aimonitor-grafana 2>&1 | grep "Plugin registered" | grep daydaymoney

# 4) gnet 市场 API（私有插件 404 可忽略）
curl -s -o /dev/null -w "%{http_code}" -u admin:admin \
  http://localhost:3000/api/gnet/plugins/daydaymoney-grafana-app
# 可能仍为 404 — 不影响本地插件使用

# 5) 配置页可打开
# 打开 /plugins/daydaymoney-grafana-app?page=configuration
```

### 5. 文档同步

- 更新 `DaydaymoneyGrafana/README.md`：注明 **Grafana >= 11.5** 要求
- 更新 `DaydaymoneyGrafana/USAGE.md`：补充「plugin not found」排查表（区分 gnet vs 本地 API）

## 实施任务（Checkbox）

- [x] `DaydaymoneyGrafana/src/plugin.json` `grafanaDependency` → `>=11.5.0`
- [x] `DaydaymoneyGrafana/package.json` `@grafana/*` → `11.5.2`
- [x] `AiMonitor/run.sh` 增加 `build_daydaymoney_grafana_plugin` 预构建
- [x] 本地 `npm run build` + `npm run test:ci` 通过
- [x] 本地验收 `GET /api/plugins/daydaymoney-grafana-app/settings` → 200
- [ ] **服务器**执行：`cd DaydaymoneyGrafana && npm run build`（或同步 dist）
- [ ] **服务器**执行：`cd AiMonitor && ./run.sh stop && ./run.sh start`
- [ ] **服务器**验收配置页可输入账号/令牌并登录

## 风险与回滚

| 风险 | 缓解 |
|------|------|
| Grafana 12 升级导致 Dashboard 异常 | 启动后抽查 provisioning dashboards |
| npm build 在 CI/服务器失败 | run.sh 构建失败则 exit 1，不启动半残栈 |
| 回滚 | 恢复 `11.5.2` 镜像 + 使用方案 B 降级插件（不推荐长期） |

## 🏛️ 架构变更影响

- **迭代版本**：无新版本（部署/配置修复）
- **已有文件（未修改）**：`docs/architecture/v5-*`（current/shipped）
- **变更明细**：无组件增删；仅 AiMonitor Grafana 镜像版本与插件部署流程说明补充
