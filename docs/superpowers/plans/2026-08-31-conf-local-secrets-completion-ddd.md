# DDD: conf-local 唯一 overlay

- **日期**: 2026-08-31
- **Bounded context**: 配置加载（confload）/ 部署主机 overlay。非业务域。

无新聚合、实体、领域事件。业务意图「只从 conf + conf-local 加载」为配置面，书面例外不投 MQ（见 intent 对照表）。

加载契约（实现必须遵守）：

```
conf/<rel>  →  conf-local/<rel>
```

禁止第三文件 `config.local.yaml` / `*.local.yaml`。GENERATED 同目录片段（`docker-infra.yaml`）仍从 `conf/<app>/` 合入，不是 `.local.yaml` overlay。
