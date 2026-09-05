# NFR: 新节点 clone-run

- **日期**: 2026-08-31
- **默认档**: L2；PEM/OIDC 密钥路径 L3。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 档位 | 判定 |
|------|-----------------|------|------|
| clone + `up.sh` 装配 | 否 | L0 | 单机运维控制面；升级触发：多区域配置服务 |
| 进程读 `conf-local/**` | 否 | L0 | 主机文件，非租户 API |
| GitHub Release `github://…@tag` | 否 | L0 | 全局产物钉；非租户分片 |

## 幂等性审视

| 路径 | 副作用 | 档位 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------|----------|--------------|--------|----------|
| `up.sh` rsync conf-local | 覆盖 overlay | L2 | 重复 up | 相对路径 | 路径 | 后写覆盖；不碰 `conf/` |
| `setup_tls` staging PEM | 写 `certs/` | L2 | 重复 start | 证书文件对 | 文件路径 | conf-local 存在则覆盖 staging |
| `initOidcSigningKey` | 读/可能 mint PEM | L3 | 进程启动 | 签名钥身份 | 文件路径 | DEPLOY_MODE 禁止 mint；旧路径一次性拷入 conf-local |
| `deploy-sync` | 写 `bin/` | L2 | 重复 sync | asset+sha | pin sha | last-good 保留；新机无 last-good 时失败须可见 |
| 9999 INIT_ALL | 写库 schema | L3 | 运维确认 | 库名 | 迁移 step_key | 既有幂等；本迭代不改 |

## 其它

- 机密不进日志/README/commit。
- 可用性：新节点缺 PEM 或空 `bin/` 必须失败，禁止静默自签/空跑。
