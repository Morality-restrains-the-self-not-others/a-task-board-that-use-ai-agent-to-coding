# 实施计划：daydaymoney.yaml 元信息全链路

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-daydaymoney-yaml-metadata-design.md`

## 任务清单

### A. Schema 与仓库文件

- [x] A1. 共享 Go 包 `shareLib/daydaymoneymeta`：Parse/Validate + 单测 T1/T2
- [x] A2. 共享 JS：`task2app/.../daydaymoneyMeta.js` + `taskChromePlugin/lib/daydaymoney-meta.js`
- [x] A3. 为主要服务目录与根目录写入 `daydaymoney.yaml`
- [x] A4. CI 脚本 `db/scripts/ci/check_daydaymoney_yaml.py` + 清单

### B. taskProjectService API

- [x] B1. `POST .../daydaymoney/parse-yaml` + 测试
- [x] B2. `GET .../daydaymoney/resolve` 多匹配 + 测试 T3–T5
- [x] B3. `GET .../projects/?tag=` + 测试 T6
- [x] B4. openapi.yaml + api_route_ownership.yaml

### C. 前端项目页与 header

- [x] C1. 同步标签工具函数 + Vitest T7
- [x] C2. Create/Edit/Detail UI 按钮「从 daydaymoney.yaml 同步」
- [x] C3. task2app SPA head 注入 `daydaymoney-service-id` / `daydaymoney-tags`
- [x] C4. CI：`check_daydaymoney_yaml.py`（head 与 YAML 一致可后续增强，见 OPT）

### D. Chrome 插件

- [x] D1. `api.resolveDaydaymoneyMeta`
- [x] D2. 浮窗/panel：读 meta → resolve → 预选 T9

### E. 日志与 Grafana

- [x] E1. tracelog `SetDaydaymoneyMeta` / Init 注入 T10
- [x] E2. 观测规范文档补字段
- [x] E3. `projectMatcher` 精确匹配优先 T11/T12

### F. 文档收尾

- [x] F1. 架构 v38 + 意图/设计（价值流 wsd 未改业务状态机，跳过）
- [x] F2. OPT 落盘；Ship PR
