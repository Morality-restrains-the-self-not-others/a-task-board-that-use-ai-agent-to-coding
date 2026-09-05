# NFR 澄清: relay 本地 GitLab 推送卡住

> 输入: `2026-05-27-relay-localhost-gitlab-push-stuck` design + value-stream

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 单次 push 容器 git 命令默认超时 90s；SaaS 读超时沿用 120s |
| 可靠性 | L2 | push 失败必须释放 `layerGraphBusyActionKey`（现有 finally） |
| 安全性 | L3 | token 仅经 `oauth_auth_by_repo` 注入 GIT_ASKPASS，不落日志 |
| 可维护性 | L2 | canonical repo key 与 SaaS `forward_container_layer_git_push` 一致 |
| 可观测性 | L2 | `git-push.log` 记录 skip/push ok/fail/timeout |

## 质量场景

### QS-01: localhost GitLab 不 skip
| 要素 | 内容 |
|------|------|
| 刺激 | `oauth-access-push` + origin `http://localhost:8012/ljy/somanyad.git` |
| 响应 | 执行 git push，日志非 `skip=non_github_remote` |
| 度量 | `layerGitOauthPush.test.mjs` 通过 |

### QS-02: 推送超时可感知
| 要素 | 内容 |
|------|------|
| 刺激 | git 远端无响应 |
| 响应 | 90s 内 reject，detail 含「超时」 |
| 度量 | 单测或手工 `GIT_PUSH_TIMEOUT_MS=5000` |

### QS-03: relay E2E 推送 SaaS 200
| 要素 | 内容 |
|------|------|
| 刺激 | Playwright 全流程至点击推送 |
| 响应 | `container-layer-git-push` HTTP 200 |
| 度量 | Playwright（凭据齐全环境） |

## 领域模型影响

| NFR | DDD 动作 |
|-----|---------|
| canonical key 对齐 | 复用 VO `LayerPushOauthReadiness`；容器侧逻辑视为应用层适配，不新增聚合 |
| L3 安全 | `oauth_auth_by_repo` 红acted 日志保持不变 |

## 权衡与边界

- 不扩展 `prefer_container_remote` 至 relay 单仓（避免无凭据的 `git/push`）
- 不实现 GitLab MR API
