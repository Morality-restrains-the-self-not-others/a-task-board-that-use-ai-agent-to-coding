#!/usr/bin/env bash
# 应用 GitLab 注册/账密登录策略（OPT-20260807-034）。
# SSOT: conf/infra/git-service/config.yaml → signupEnabled / passwordAuthWeb / passwordAuthGit
# run.sh 已将这些键导出为 GITLAB_SIGNUP_ENABLED / GITLAB_PASSWORD_AUTH_WEB / GITLAB_PASSWORD_AUTH_GIT。
#
# GitLab 的 signup_enabled / password_authentication_enabled_for_web|git 是 DB 级
# application_setting，gitlab.rb 的 gitlab_rails['signup_enabled'] 只在 DB 值为空时生效，
# 对既有 DB 值不会覆盖。因此启动/bootstrap 时显式按 conf 强制写回，保证「conf 即最终态」。
set -euo pipefail

CONTAINER="${GITLAB_CONTAINER:-${GITLAB_CONTAINER_NAME:-gitlab}}"
SIGNUP="${GITLAB_SIGNUP_ENABLED:-false}"
PW_WEB="${GITLAB_PASSWORD_AUTH_WEB:-false}"
PW_GIT="${GITLAB_PASSWORD_AUTH_GIT:-false}"

# 仅接受 true/false，防 conf 误填其它值注入 Ruby
case "$SIGNUP" in true|false) ;; *) echo "非法 GITLAB_SIGNUP_ENABLED=$SIGNUP" >&2; exit 1 ;; esac
case "$PW_WEB" in true|false) ;; *) echo "非法 GITLAB_PASSWORD_AUTH_WEB=$PW_WEB" >&2; exit 1 ;; esac
case "$PW_GIT" in true|false) ;; *) echo "非法 GITLAB_PASSWORD_AUTH_GIT=$PW_GIT" >&2; exit 1 ;; esac

echo "应用注册/账密策略: signup_enabled=$SIGNUP password_auth_web=$PW_WEB password_auth_git=$PW_GIT"
docker exec "$CONTAINER" gitlab-rails runner "
s = ApplicationSetting.current
s.update!(
  signup_enabled: $SIGNUP,
  password_authentication_enabled_for_web: $PW_WEB,
  password_authentication_enabled_for_git: $PW_GIT,
)
puts 'auth_policy applied: signup_enabled=' + ApplicationSetting.current.signup_enabled.to_s + \
  ', password_auth_web=' + ApplicationSetting.current.password_authentication_enabled_for_web.to_s + \
  ', password_auth_git=' + ApplicationSetting.current.password_authentication_enabled_for_git.to_s
" || { echo "❌ 应用注册/账密策略失败（GitLab 未就绪或 runner 异常）" >&2; exit 1; }
echo "✅ 注册/账密策略已对齐 conf"
