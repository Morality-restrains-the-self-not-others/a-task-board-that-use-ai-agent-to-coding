#!/usr/bin/env bash
# Ensure nested-repo `gitlab` remotes use SSH to local GitLab CE (gitlab-shell),
# and optionally create missing projects under example-user.
#
# Why: public DNS gitlab.daydaymoney.com:2222 may hit HK OpenSSH, not gitlab-shell.
# ~/.ssh/config should map Host gitlab.daydaymoney.com → HostName 127.0.0.1 Port 2222.
#
# Usage:
#   bash db/scripts/ensure_gitlab_mirror_remotes.sh
#   bash db/scripts/ensure_gitlab_mirror_remotes.sh --create-missing
#   GITLAB_PAT=... bash db/scripts/ensure_gitlab_mirror_remotes.sh --create-missing
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NAMESPACE="${GITLAB_NAMESPACE:-example-user}"
CREATE_MISSING=false
API="${GITLAB_API:-https://gitlab.daydaymoney.com/api/v4}"

for arg in "$@"; do
  case "$arg" in
    --create-missing) CREATE_MISSING=true ;;
    -h|--help)
      sed -n '2,16p' "$0"
      exit 0
      ;;
  esac
done

resolve_pat() {
  if [[ -n "${GITLAB_PAT:-}" ]]; then
    printf '%s' "$GITLAB_PAT"
    return 0
  fi
  local cred="$HOME/.config/git/gitlab-daydaymoney.credentials"
  if [[ -f "$cred" ]]; then
    python3 - "$cred" <<'PY'
import sys, urllib.parse
raw = open(sys.argv[1], encoding="utf-8").read().strip().splitlines()[0]
u = urllib.parse.urlparse(raw)
print(urllib.parse.unquote(u.password or ""))
PY
    return 0
  fi
  return 1
}

create_project_if_missing() {
  local path="$1"
  local pat="$2"
  local code
  code=$(curl -sS -o /tmp/gl-ens-proj.json -w '%{http_code}' \
    -H "PRIVATE-TOKEN: $pat" \
    "$API/projects/${NAMESPACE}%2F${path}")
  if [[ "$code" == "200" ]]; then
    echo "exists ${NAMESPACE}/${path}"
    return 0
  fi
  code=$(curl -sS -o /tmp/gl-ens-create.json -w '%{http_code}' \
    -X POST -H "PRIVATE-TOKEN: $pat" -H 'Content-Type: application/json' \
    -d "{\"name\":\"$path\",\"path\":\"$path\",\"visibility\":\"private\",\"initialize_with_readme\":false}" \
    "$API/projects")
  echo "create ${NAMESPACE}/${path} http=$code"
}

if ssh -o BatchMode=yes -o ConnectTimeout=5 -T "git@gitlab.daydaymoney.com" 2>&1 | grep -q 'Welcome to GitLab'; then
  echo "SSH OK: git@gitlab.daydaymoney.com → gitlab-shell"
else
  echo "警告: SSH 未欢迎 GitLab。请检查 ~/.ssh/config Host gitlab.daydaymoney.com 的 HostName 是否指向运行 GitLab 容器的宿主机（常用 127.0.0.1:2222），而不是公网 HK OpenSSH。" >&2
fi

PAT=""
if [[ "$CREATE_MISSING" == true ]]; then
  PAT="$(resolve_pat || true)"
  if [[ -z "$PAT" ]]; then
    echo "错误: --create-missing 需要 GITLAB_PAT 或 ~/.config/git/gitlab-daydaymoney.credentials" >&2
    exit 1
  fi
fi

mapfile -t REPOS < <(find "$ROOT" -maxdepth 2 -type d -name .git 2>/dev/null | sed 's|/.git||' | sort)
for repo in "${REPOS[@]}"; do
  url="$(git -C "$repo" remote get-url gitlab 2>/dev/null || true)"
  [[ -z "$url" ]] && continue
  name="$(basename "$repo")"
  if [[ "$repo" == "$ROOT" ]]; then
    name="ram-work"
  fi
  ssh_url="git@gitlab.daydaymoney.com:${NAMESPACE}/${name}.git"
  if [[ "$url" != "$ssh_url" ]]; then
    git -C "$repo" remote set-url gitlab "$ssh_url"
    echo "remote $name: $url → $ssh_url"
  else
    echo "remote $name: OK"
  fi
  if [[ "$CREATE_MISSING" == true ]]; then
    create_project_if_missing "$name" "$PAT"
  fi
done

echo "Done. Example: git -C gitService push -u gitlab main"
