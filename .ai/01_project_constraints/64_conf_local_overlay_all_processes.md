# 所有进程加载 conf 必须叠加 conf-local（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-09-01
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo 全部业务/基础设施进程及其配置加载脚本
- 架构决策：ADR-0054（机密落点与加载器契约；本条强制所有进程走该契约）
- 约束索引：第 59 条
- No-ADR: covered by existing ADR-0054

## 核心原则

**凡运行时读取 `conf/**/*.yaml`（含 `base.yaml`、本目录片段、docker-infra 等），必须经 SSOT 加载器深合并同相对路径的 `conf-local/<rel>`。禁止只读已跟踪 `conf/`。禁止加载 `config.local.yaml`。**

第 58 条（`63_conf_local_secrets_only.md`）规定机密只放 `conf-local/`、加载器须深合并。本条补齐执行缺口：进程不得绕过加载器直接 `os.ReadFile` / `yaml.safe_load` / `readFileSync` 打开 tracked YAML，否则 `conf-local` 中的密钥（如 SMTP `host_password`）不会生效。

## 硬约束（一级，禁止忽略）

- 生产路径加载 `conf/<rel>` 后必须深合并 `conf-local/<rel>`（文件缺失则跳过，不得报错覆盖）
- SSOT API：Go `shareLib/confload` 的 `ReadAppConfig` / `ReadAppFragment` / `UnmarshalYAMLMerged` / `ReadOverlaidYAML` / `ReadYAMLMerged` / `MergeYAMLAtPath` / `MergeConfLocal` / `ResolveBaseYaml`；Python `runAll/scripts/conf_local.py` 的 `overlay_conf_file` / `merge_conf_local`
- **禁止**生产路径加载同目录 `config.local.yaml` / `*.local.yaml`（ADR-0054）
- 跨服务配置仍须 sync 到本 app 目录后再 overlay（第 30 条）
- 自定义合并（未迁到 SSOT）须在源码标注 `Conf-Local-Overlay-OK: <理由>`，并登记 OPT 限期迁到 SSOT；浅 `dict.update` / 按 struct Unmarshal 不算深合并达标

## 触发

- 新增/修改 `loadConfig`、`config.go`、`settings.py`、进程启动读 YAML
- 改 `shareLib/confload`、`runAll/scripts/conf_loader.py`、`conf_local.py`
- 评审发现 `os.ReadFile` / `yaml.safe_load` / `readFileSync` 指向 `conf/**/*.yaml`

## 验收标准

```bash
python3 db/scripts/ci/test_check_conf_local_overlay.py
python3 db/scripts/ci/check_conf_local_overlay.py
cd shareLib/confload && go test -count=1 -run 'TestReadAppConfigMergesConfLocal|TestResolveBaseYamlMergesConfLocal' .
```

## 实现位置

| 组件 | 路径 |
|------|------|
| 约束专文 | `.ai/01_project_constraints/64_conf_local_overlay_all_processes.md` |
| Cursor 元规则 | `.cursor/rules/conf-local-overlay-all-processes.mdc` |
| ADR | `docs/adr/0054-conf-local-secrets-only.md` |
| Go 合并 | `shareLib/confload/overlay.go` |
| Python 合并 | `runAll/scripts/conf_local.py` |
| 门禁 | `db/scripts/ci/check_conf_local_overlay.py` |
| 自测 | `db/scripts/ci/test_check_conf_local_overlay.py` |
| 意图 | `docs/intents/platform/conf_local_overlay_all_processes.intent.md` |

## 关联

- 第 58 条：机密只放 conf-local；加载器契约
- 第 30 条：运行时只读本服务 conf 目录；跨服务须 sync
- 第 42 条 / ADR-0052：人工可改配置编辑落点
- 第 57 条：禁止源码硬编码密钥
