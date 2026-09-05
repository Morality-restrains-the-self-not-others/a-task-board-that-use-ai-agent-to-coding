# task2app 元规则：monorepo SDK 路径更新

> 日期：2026-06-01  
> 状态：**已实施**  
> 触发：阿里云 Python SDK 由 `~/alibabacloud-python-sdk` 迁入 `~/gitClone/ramDisk/ram-mount/sdk/alibabacloud-python-sdk`

---

## 1. 决策

| 项 | 旧 | 新 |
|----|----|-----|
| Python SDK 根 | `~/alibabacloud-python-sdk` | `sdk/alibabacloud-python-sdk`（相对 monorepo 根） |
| ECS 云服务器包 | 同上 `/ecs-20140526` | `sdk/alibabacloud-python-sdk/ecs-20140526` |
| 配置真源 | 文档硬编码 home 路径 | `task2app/paths.conf` → `SDK_*` 键 |
| 范例引用 | 注释写 `~/…` | `07_aliyun_sdk_usage.md` + `cloud/providers/aliyun/` |

`REPO_ROOT` 仍为 `task2app/`，故 SDK 在 paths.conf 中使用 `../sdk/...`。

---

## 2. 已修改文件

| 文件 | 变更 |
|------|------|
| `task2app/paths.conf` | 新增 `SDK_DIR`、`SDK_ALIYUN_PYTHON`、`SDK_ECS_PYTHON`、`SDK_ECS_GO` |
| `.ai/03_technical_implementation/07_aliyun_sdk_usage.md` | v1.1.0，完整 SDK 布局与 ECS 范例索引 |
| `.ai/03_technical_implementation/03_cloud_sdk_specifications.md` | 修正错误链接 `05_` → `07_` |
| `.ai/01_project_constraints/04_paths_configuration.md` | 补充 SDK 路径键 |
| `Saas_project/cloud/providers/aliyun/{network/vswitch,image}.py` | 注释路径 |
| `Saas_project/scripts/auxiliary/create_aliyun_resources.py` | 注释路径 |

---

## 3. 价值流影响

无业务价值流变更；仅开发/AI 规则与本地依赖安装路径。`value-stream.yaml` 无需修改。

---

## 4. 后续（可选）

- `paths_loader` 导出 `sdk_ecs_python()` 便捷函数
- CI 校验 `pip install -e` 使用 paths.conf 路径
- 同步 `task2app-wt-relay-stop` worktree（若仍在使用）
