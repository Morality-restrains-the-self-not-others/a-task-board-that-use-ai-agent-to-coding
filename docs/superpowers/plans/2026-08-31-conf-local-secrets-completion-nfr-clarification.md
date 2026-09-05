# NFR: conf-local 唯一 overlay

- **日期**: 2026-08-31
- **默认档**: L2；本增量无资金写路径。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 档位 | 判定 |
|------|-----------------|------|------|
| 主机读 `conf/<app>/config.yaml` | 否 | L0 | 单机文件加载，非租户 API；升级触发：若改为多租户配置服务 |
| 主机读 `conf-local/<app>/config.yaml` | 否 | L0 | 同上 |
| CI 扫描已跟踪 `conf/` | 否 | L0 | 仓库静态检查 |

## 幂等性审视

| 路径 | 副作用 | 档位 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------|----------|--------------|--------|----------|
| `ReadAppConfig` / `load_app_config` | 无（纯读） | L0 | 进程启动多次 | — | — | 每次读磁盘当前文件 |
| `extract_conf_secrets_to_local.py` | 写 `conf-local/`、抽空 `conf/` 键 | L2 | 重复执行 | 相对路径文件 | 文件路径 | 空键保持骨架；不写 `.local.yaml` |
| `up-from-config-repo.sh` rsync `conf-local/` | 覆盖主机 overlay | L2 | 重复 rsync | 相对路径 | 路径 | 后写覆盖；禁止再拷 `*.local.yaml` 进 `conf/` |
| 云/GitLab 凭据轮换 | 有（控制台） | L3 | OPS 窗口手工 | 凭据 ID | 控制台轮换流程 | 本迭代不执行 live 轮换 |

## 其它

- 机密不进日志、example、commit message。
- 可用性：缺 `conf-local/` 时骨架空串，服务按现网降级（与 ADR-0054 一致）。
