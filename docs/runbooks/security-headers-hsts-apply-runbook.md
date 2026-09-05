# 安全响应头 + HSTS 分阶段落地 runbook（公网主站边缘 nginx）

> 设计：`docs/superpowers/specs/2026-08-24-security-headers-hsts-risk-analysis-design.md`（v106）
> 目标态示例配置：`docs/examples/edge-nginx-main-site-daydaymoney.com.conf.example`
> 适用主机：公网宿主机 `1.117.67.121`（腾讯云），边缘 nginx 配置在 `/etc/nginx/conf.d/`（repo 外）
> ⚠️ 执行前先备份：`tar czf /root/backups/nginx-conf-$(date +%Y%m%d-%H%M).tar.gz /etc/nginx/conf.d/`

## 前置盘点结论（2026-08-24 已做，复用时仅需复核 DNS）

| 项 | 结论 |
|----|------|
| 子域 A 记录 | 仅 `www`、`api`（CNAME→apex）+ apex，全部指向 1.117.67.121 |
| TLS 证书 | wildcard `*.daydaymoney.com` + apex（SAN 全覆盖），2026-10-17 过期 |
| :80 现状 | 返回 200 内容（**无跳转**，风险确认） |
| :443 现状 | 无 HSTS / CSP / XFO / nosniff / Referrer-Policy（实测确认） |

## Phase 1 — 低风险头立即生效（建议当天完成）

在 `20-https.conf`（或新建 `30-security-headers.conf`，放 :443 server 块内）：

```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header X-Frame-Options $html_doc always;                 # $html_doc 见下
add_header Strict-Transport-Security "max-age=300" always;   # 仅 :443
```

`$html_doc` map（conf.d 顶层，http 上下文；nginx ≥ 1.11）：

```nginx
map $upstream_http_content_type $html_doc {
    default     "";
    ~*text/html "SAMEORIGIN";
}
```

> ⚠️ nginx 陷阱：同一层级存在任意 `add_header` 时整层覆盖继承——XFO/CSP 必须走
> `$html_doc` 变量，不能拆成两个裸 add_header 期望「只对 HTML 生效」。
> Referrer-Policy 禁止换 `no-referrer`/更严值（taskGitOauth `febFromRequest`
> 依赖 origin 级 Referer 解析 OAuth 回跳源，见设计 §3 风险表）。

验证：

```bash
nginx -t && systemctl reload nginx
curl -sI https://www.daydaymoney.com/ | grep -iE 'strict-transport|x-content-type|referrer-policy|x-frame'
# 期望：HSTS max-age=300；nosniff；strict-origin-when-cross-origin；HTML 页有 XFO: SAMEORIGIN
curl -sI https://www.daydaymoney.com/api/health/ | grep -i x-frame   # API 响应不应有 XFO
```

前端冒烟：登录、OAuth（GitHub/GitLab/微信）回跳、SSE 执行日志、富文本日志 iframe、容器页跳转。

## Phase 2 — CSP 灰度（1–2 周后启动，另立任务）

1. `Content-Security-Policy-Report-Only`（仅 HTML，用 `$html_doc`）收集违规 ≥1 周
2. 策略骨架：`default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-src 'self'; img-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'`（以 audit 结果为准）
3. 前置已就绪：taskFE 内联脚本已外置 `/static/assets/api-base.js`（v106 Phase 0a 完成，构建后 `index.html` 无内联 `<script>`）
4. 零生产违规后转 `Content-Security-Policy` enforce

## Phase 3 — HSTS 阶梯升级

| 步 | max-age | 观察期 | 条件 |
|----|---------|--------|------|
| 1 | 300 | ≥1 周 | 无 http 直连客户端报障 |
| 2 | 86400 | ≥1 周 | 同上 |
| 3 | 31536000 | 长期 | 证书续期自动化确认（**certbot renew --dry-run 跑通**）后再评估 includeSubDomains/preload |

> 证书为单点（wildcard 2026-10-17 过期，验证走腾讯云 DNS ohttps.com）。
> 续期自动化确认前不加 includeSubDomains，防止子域证书缺失时浏览器强 HTTPS 全断。

## Phase 4 — HTTP→HTTPS 强制跳转

```nginx
# :80 server 块
return 302 https://$host$request_uri;   # 先 302 过渡 ≥1 周
# 确认无 http 客户端断裂后切 308：
# return 308 https://$host$request_uri;
```

> 302→308 切换后验证：`curl -sI http://www.daydaymoney.com/` 首行应为 `308` 且
> `Location: https://www.daydaymoney.com/`。

## 复扫验收（每阶段）

```bash
curl -sI https://www.daydaymoney.com/          # 头齐全
curl -sI http://www.daydaymoney.com/           # 308
curl -sI https://api.daydaymoney.com/          # 头齐全（如该子域复用主站 server 块）
```

内网面（10.2.150.68:9999/:3000/:4000/:18081）缺 HSTS 项 → 声明 **out of scope**（内网全 HTTP，
无 HTTPS 端点可挂 HSTS）。

## 回滚

```bash
tar xzf /root/backups/nginx-conf-<时间戳>.tar.gz -C /etc/nginx/ && nginx -t && systemctl reload nginx
```
