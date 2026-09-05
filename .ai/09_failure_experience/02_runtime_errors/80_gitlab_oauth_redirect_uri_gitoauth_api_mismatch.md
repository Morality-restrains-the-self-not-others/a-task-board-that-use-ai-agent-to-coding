# [运行时] GitLab OAuth → The redirect URI included is not valid（gitoauth_api vs gitoauth.api）

## 基本信息

- 案例编号：FE-20260723-GITOAUTH-REDIRECT-URI-MISMATCH
- 录入日期：2026-07-23
- 最后更新：2026-07-23
- 关联服务：taskGitOauth（:8002）、GitLab CE Doorkeeper、边缘 nginx、DNSPod

## 失败现象

- 页面：`/tenant/{id}/projects/{proj}/`（Git 仓行「授权异常」→ 重试）
- GitLab 提示：**The redirect URI included is not valid.**
- 常见于 authorize 的 `redirect_uri` 与 Doorkeeper Application 白名单不一致。

## 根因（历史漂移）

| 组件 | 曾用主机 | 说明 |
|------|---------|------|
| `conf/base.yaml` `subdomains.gitoauth` | `gitoauth_api.*` | SSOT |
| 旧 nginx / Doorkeeper / DNS | `gitoauth.api.*` | 多层子域；`*.daydaymoney.com` 证书不覆盖 |

两端不一致时，authorize 被 GitLab 拒绝。

## 现行标准（2026-07-23）

域名硬编码**仅**在 `conf/base.yaml`；其余一律模板引用：

1. Provider：`${scheme}://${subdomains.gitoauth}/api/accounts/gitlab/oauth/callback/`
2. GitLab Doorkeeper：由 sync 脚本按 base.yaml 展开后写入
3. 边缘 nginx：源文件写 `server_name ${subdomains.gitoauth}`，`deploy_daydaymoney_sh_nginx.sh` render 后部署
4. DNS：为 `subdomains.gitoauth` 展开主机补 A/AAAA → 边缘机

## 验收

```bash
bash gitService/scripts/test_gitlab_daydaymoney_redirect_uri_ssot.sh
# → OK redirect_uri=https://gitoauth_api.daydaymoney.com/api/accounts/gitlab/oauth/callback/

curl -sS -H 'X-User-Id: 1' -H 'Accept: application/json' \
  'http://127.0.0.1:8002/api/accounts/gitlab/oauth/start-from-gateway/?next=/x&return_key=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&repo_url=https%3A%2F%2Fgitlab.daydaymoney.com%2Fg%2Fr.git&service_provider=daydaymoney-gitlab' \
  | python3 -c 'import json,sys,urllib.parse; u=json.load(sys.stdin)["authorize_url"]; print(urllib.parse.parse_qs(urllib.parse.urlparse(u).query)["redirect_uri"][0])'
# → https://gitoauth_api.daydaymoney.com/api/accounts/gitlab/oauth/callback/

docker exec gitlab gitlab-rails runner '
puts Doorkeeper::Application.find_by(uid: "0293ce8bb6c195169fcc585fa741851c431b98b7cfa162ce5edcb3290ad2f9e1").redirect_uri
'
getent hosts gitoauth_api.daydaymoney.com
curl -sSI https://gitoauth_api.daydaymoney.com/api/health/ | head -5
```

## 防再发

- 伴读：`conf/auth/git-oauth/ai.md`
- 禁止把 provider 改回 `gitoauth.api.${baseDomain}`
