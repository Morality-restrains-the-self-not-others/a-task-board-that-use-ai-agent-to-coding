# 测试意图: conf-local 机密收口

## 测试目标

证明加载器只合 `conf/<rel>` 与 `conf-local/<rel>`；`config.local.yaml` / `*.local.yaml` 即使存在也不参与 merge；门禁拒绝跟踪的 `*Pwd.md`；example 骨架无真值。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | Go `confload`：旁路 `config.local.yaml` 有冲突键时结果仍等于 conf+conf-local |
| 单元 | Python `conf_loader`；`gitService load_gitservice_config` 同等 |
| 单元 | 抽取脚本只写 conf-local，不写 `.local.yaml` |
| 门禁 | 跟踪 `*Pwd.md` 失败；`conf/` YAML 非空机密键仍失败 |
| OPS | 轮换表手工勾选 |

## 用例矩阵

| # | 用例 | 对应验收 | 期望 |
|---|------|----------|------|
| T1 | conf-local 有 secret，`config.local.yaml` 同键为 `""` 或其它值 | 验收 1 | 结果为 conf-local 值 |
| T2 | 仅 `config.local.yaml` 有 `runAllStartEnabled`，conf 与 conf-local 皆无 | 验收 1–2 | 合并结果**无**该键（须改写到 conf-local 才生效） |
| T3 | `ReadAppFragment` 旁路 `sms.local.yaml` | 验收 1 | 只用来自 conf + conf-local 的 fragment |
| T4 | 抽取脚本不创建 `.local.yaml` | 验收 2 | 输出目录无新 `.local.yaml` |
| T5 | 跟踪 `gitLabRootPwd.md` | 验收 3 | 门禁非 0 |
| T6 | `conf-local.example` 与已知 relpath 对齐 | 验收 4 | 骨架无真密钥模式 |
| T7 | 轮换后旧凭据失败 | 验收 5 | OPS 手工 |

## 数据与环境

- Fixture 用假密钥。禁止拷贝本机 `conf-local/` 真值。

## 通过标准

- T1–T6 自动化全绿；T7 在 OPS 窗口勾选。

## 业务意图 → 事件对照（测试）

| 业务意图 | 事件断言 | 例外理由 |
|---------|----------|---------|
| 只从 conf + conf-local 加载 | 无 MQ 断言 | 配置加载 |
