# [运行时] 镜像市场管理 SSO 跳转后 provider.daydaymoney.com 无法打开

## 基本信息

- 案例编号：RT-20260901-0126
- 录入日期：2026-09-01
- 最后更新：2026-09-01
- 关联服务：taskAuth（SSO bridge）、taskAiProvider :8010、SH 边缘 nginx、`ensure-edge-tunnels.sh`
- 关联页面：https://www.daydaymoney.com/system-admin/container-images 「镜像市场管理（SSO）」

## 失败现象

- 点击后浏览器停在 `https://www.daydaymoney.com/api/accounts/sso/ai-provider/admin/?accessCode=…` 或跟 302 到 `https://provider.daydaymoney.com/api/auth/sso/exchange/?bridge=…` 后一直转圈 / 「无法打开」。
- 无 Cookie 的 curl 该 SSO URL 会 **401** JSON「无法解析登录凭据」（forward-auth），那是探活噪音，不是本故障。
- 已登录用户（Loki：`bootstrap-admin`，`GET /api/accounts/sso/ai-provider/admin/` **302**，trace `e98850ef18ce92472b66a62985b810c7`）签发成功；下一跳公网 provider **TLS 通、HTTP 0 字节超时**。

## 环境与上下文

- SH（`1.117.67.121`）`20-https.conf` 已有 `server_name provider.daydaymoney.com` → `upstream 127.0.0.1:8010`。
- 本机 `*:8010` 的 taskAiProvider 正常 200；`autossh -R 8010:127.0.0.1:8010 sh` 进程也在。
- 在 SH 上 `ss` 显示 `127.0.0.1:8010` 由 **sshd** LISTEN，但 `curl -m 5 http://127.0.0.1:8010/` **超时 0 字节**（半开反向转发）。
- 同机 SH `curl http://127.0.0.1:18081/` 正常 → 不是整条隧道机挂了，是 **8010 这一条 channel 死了、LISTEN 还在**。

## 根因

1. SSH 反向隧道半开：远端 sshd 仍占着端口，nginx `proxy_pass` 一直等到超时，浏览器表现为「打不开」。
2. `ensure-edge-tunnels.sh` 只 `pgrep autossh.*-R <port>`：进程在就算健康，**不探活**。
3. 宽松 `pgrep -f` 会把 Agent `bash -c '…autossh.*-R 8010…'` 诊断命令误当成隧道，跳过重启。

`accessCode` 是 Cursor 浏览器会话参数，SSO 桥不读它；不是 401/打不开的原因。

## 修复

1. 杀掉 SH 上占用 8010 的僵 sshd 子进程 + 本机对应 autossh，重建 `-R 8010`。
2. 探活：SH `curl -m 2 http://127.0.0.1:<port>/`；超时/拒绝则杀远端监听并重建。
3. `pgrep` 锚定 `/usr/lib/autossh/autossh .*-R …`。
4. 谓词：`runAll/scripts/edge_tunnel_health.py` + `test_edge_tunnel_health.py`。

## 验收

```bash
ssh sh 'curl -sS -m 5 -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8010/'
# 200
curl -sS -m 8 -o /dev/null -w "%{http_code}\n" https://provider.daydaymoney.com/
# 200，且 HTML 含 trae-service taskAiProvider
curl -sS -m 8 -o /dev/null -w "%{http_code}\n" --max-redirs 0 \
  https://provider.daydaymoney.com/api/auth/sso/exchange/
# 302 Location=/?error=缺少+bridge
python3 runAll/scripts/tests/test_edge_tunnel_health.py
```

登录后从容器镜像列表再点「镜像市场管理（SSO）」应进入 provider admin，不再白屏超时。

## 防再发

- cron 已跑 `ensure-edge-tunnels.sh`：半开隧道须靠 **HTTP 探活** 而不能只看 autossh PID。
- 诊断「SSO 打不开」时先看 Loki 该 GET 是 401 还是 302；302 则查 `provider.*` 与 SH `:8010` 隧道，不要先改 taskFE href。
