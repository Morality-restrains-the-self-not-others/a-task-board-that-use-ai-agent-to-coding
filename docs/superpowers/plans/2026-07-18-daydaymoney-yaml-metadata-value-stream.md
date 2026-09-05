# 价值流：daydaymoney.yaml 元信息全链路

**日期**: 2026-07-18  
**迭代**: daydaymoney-yaml-metadata

## 端到端价值流

```
[开发者提交 daydaymoney.yaml]
        ↓
[CI 校验 schema + head meta 一致]
        ↓
┌───────┴────────┬──────────────────┬─────────────────┐
↓                ↓                  ↓                 ↓
[项目页同步 tags] [SPA 注入 head]  [服务启动注入日志]  [静态托管 yaml]
        ↓                ↓                  ↓                 ↓
[projects.tags]   [Chrome 读 meta]   [Loki JSON]      [Chrome fetch]
        ↓                ↓                  ↓
        └────→ [GET daydaymoney/resolve] ←────────┘
                        ↓
              [多 WS/项目 matches]
                        ↓
        ┌───────────────┴───────────────┐
        ↓                               ↓
[插件自动勾选建任务]            [Grafana 精确匹配建任务]
```

## 最小可交付增量（MVP）

1. Schema + 批量 YAML 文件 + 共享解析库  
2. Go resolve / parse-yaml / `?tag=`  
3. 项目页同步按钮 + head meta（主 SPA）  
4. Chrome 浮窗自动填充  
5. tracelog 字段 + Grafana 精确匹配  

## 测试点映射

见 `docs/intents/platform/daydaymoney-yaml-metadata.test-intent.md` T1–T12。
