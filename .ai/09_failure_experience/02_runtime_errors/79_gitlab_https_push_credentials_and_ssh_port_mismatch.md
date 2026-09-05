# [运行时] gitlab.daydaymoney.com HTTPS 推送凭据失败 + SSH:2222 打到错误宿主

## 基本信息

- 版本：1.1.0
- 案例编号：RT-20260722-0079
- 录入日期：2026-07-22
- 最后更新：2026-07-22
- 录入人：Cursor Agent

## 现象

- `git push gitlab main` 失败：
  `fatal: could not read Username for 'https://gitlab.daydaymoney.com': 没有那个设备或地址`
- 或改用 SSH 后：`Permission denied (publickey)`（公钥已在 GitLab 用户下仍失败）
- 或认证通过后：`The project you were looking for could not be found`

## 环境与上下文

- 嵌套仓 `gitlab` remote 原为 HTTPS：`https://gitlab.daydaymoney.com/example-user/<repo>.git`
- 本机 `~/.ssh/config` 曾将 `Host gitlab.daydaymoney.com` 指向公网 DNS + `Port 2222`
- DNS：`gitlab.daydaymoney.com` → `47.86.27.42`（HK 边缘）
- 本机 Docker GitLab CE 监听 `0.0.0.0:2222`（gitlab-shell）与 `:8012`（HTTP）
- HK 运维 sshd 使用 **Port 2222**；公网 **:22** 可专供 GitLab SSH 反向隧道

## 根因

1. **HTTPS**：无 `credential.helper` / `.git-credentials`，非交互环境无法提示用户名密码 → push 失败。
2. **SSH 端口错宿主**：公网 `gitlab.daydaymoney.com:2222` 落到 **HK 主机 OpenSSH**，不是本机/CPU 上的 **GitLab gitlab-shell**；故即使用户已添加 `cpu_zerg_gitlab` 公钥，仍 `Permission denied (publickey)`。
3. **镜像项目缺失**：自建 GitLab 上 `example-user/<repo>` 可能不存在（数据曾丢后未重建）→ 鉴权成功仍报 project not found。

## 修复

### 临时（本机特例，已废弃）

1. ~~`~/.ssh/config`：`HostName 127.0.0.1` + `Port 2222`~~（仅本机可达，外网无效）

### 长期（已落地）

1. **HK sshd**（`/etc/ssh/sshd_config.d/99-gitlab-git-forward.conf`）：
   - `GatewayPorts clientspecified`
   - `AllowTcpForwarding yes`
   - **不要**用会挡住其它 `-R` 端口的 `PermitListen`（曾导致 HTTP 隧道 `remote port forwarding failed`）
2. **隧道脚本** `~/scripts/enable_daydaymoney_tunnel.sh`：增加  
   `-R 0.0.0.0:22:127.0.0.1:2222`（可调 `GITLAB_SSH_REMOTE_BIND` / `GITLAB_SSH_LOCAL`）
3. **本机 ssh config**：`Host gitlab.daydaymoney.com` 走公网 DNS、默认端口 **22**、`IdentityFile ~/.ssh/cpu_zerg_gitlab`；运维 SSH 仍用 `Host hk` / Port 2222
4. 嵌套仓 remote：`git@gitlab.daydaymoney.com:example-user/<repo>.git`
5. HTTPS 回退：PAT 存 `~/.config/git/gitlab-daydaymoney.credentials`（0600）+ credential helper
6. 辅助脚本：`db/scripts/ensure_gitlab_mirror_remotes.sh`

路径：

```
git@gitlab.daydaymoney.com:22 → HK 0.0.0.0:22 --autossh -R--> 本机 127.0.0.1:2222 (gitlab-shell)
HTTPS :443 → HK nginx → 127.0.0.1:8012 --autossh -R--> 本机 GitLab HTTP
```

## 验收

```bash
ssh -o BatchMode=yes -T git@gitlab.daydaymoney.com
# Welcome to GitLab, @example-user!

git -C gitService ls-remote git@gitlab.daydaymoney.com:example-user/gitService.git HEAD
git -C gitService push gitlab main

GIT_TERMINAL_PROMPT=0 git ls-remote https://gitlab.daydaymoney.com/example-user/gitService.git HEAD
```

## 预防

- 禁止假定公网 `:2222` 即 GitLab SSH（该口是 HK 运维 sshd）；Git SSH 用默认 **:22**（隧道）或本机 `:2222`。
- Agent/CI 推送优先 `git@` + 预置密钥，或 PAT helper；禁止依赖交互式 HTTPS 提示。
- 改 HK `PermitListen` 前先确认不会阻断现有 HTTP `-R` 端口列表。
