# 角色权限分析：gitservice-durable-gitlab-home

- **设计**: `2026-07-22-gitservice-durable-gitlab-home-design.md`
- **结论**: **无新增 HTTP/RPC 接口**；无权限模型变更。

## 变更面

| 面 | 影响 | 权限含义 |
|----|------|----------|
| `run.sh` / compose 卷路径 | 运维启动路径 | 仍需本机 Docker 权限；数据目录属启动用户 `$HOME` |
| `conf/infra/git-service/config.yaml` `gitlabHome` | 可选覆盖 | 仅部署方可读改本服务 conf |
| PAT 文件位置随 `GITLAB_HOME` | 脚本读密钥文件 | 文件权限保持 0600；不扩大暴露面 |

## 风险

- 多用户共用同一 `GITLAB_HOME` 可能互相覆盖 → 文档约定每开发者独立 home 路径或显式 env。
- 不引入跨租户 API。
