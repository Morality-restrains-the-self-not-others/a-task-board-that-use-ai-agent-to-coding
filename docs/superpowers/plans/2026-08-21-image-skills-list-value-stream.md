# 价值流：镜像容器技能列表

Mapping the approved design into a value stream.

## Related Value Streams

既有「镜像市场 · autoRunStep.md 抽取」：同一 OCI 抽取通道。本流**扩展** sidecar 文件种类，不替换 autoRun。

## 增量

### I1 — 约定与抽取绑定（厂商）
厂商保存镜像 → 解析 `/app/imageSkills.yaml` → 绑定 JSON。验收：T1–T3。

### I2 — 市场可见（租户）
catalog / 已安装列表返回 `image_skills`，ImageMarket 展示。验收：T4–T6。

### I3 — 创建任务高亮
选镜像后描述 `/skill` 高亮。验收：T7。

### I4 — 评论 @镜像 /skill
slash 补全 + mentions.skill + 默认技能。验收：T8–T9。

## 最小可交付

I1+文档即可形成规范；I2–I4 同批交付以满足用户 1–3 条 UI。
