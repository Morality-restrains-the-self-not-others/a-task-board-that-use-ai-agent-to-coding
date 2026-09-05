#!/usr/bin/env bash
# Syntax-check the GitLab traffic quota initializer (no GitLab runtime).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FILE="$ROOT/initializers/zzz_trae_gitlab_traffic_quota.rb"

token_check() {
  grep -q 'Gitlab::GitAccess.prepend' "$FILE"
  grep -q 'gitlab-traffic-gate' "$FILE"
  grep -q 'TRAE_TASKBILL_BASE' "$FILE"
  grep -q 'GATE_UNREACHABLE' "$FILE"
}

if command -v ruby >/dev/null 2>&1; then
  ruby -c "$FILE"
  echo "PASS ruby -c zzz_trae_gitlab_traffic_quota.rb"
  ruby "$ROOT/scripts/test_zzz_traffic_quota_skip.rb"
  exit 0
fi

if command -v docker >/dev/null 2>&1 && docker inspect gitlab >/dev/null 2>&1; then
  RUBY=/opt/gitlab/embedded/bin/ruby
  docker cp "$FILE" gitlab:/tmp/zzz_trae_gitlab_traffic_quota.rb
  docker cp "$ROOT/scripts/test_zzz_traffic_quota_skip.rb" gitlab:/tmp/test_zzz_traffic_quota_skip.rb
  docker exec gitlab "$RUBY" -c /tmp/zzz_trae_gitlab_traffic_quota.rb
  echo "PASS gitlab container ruby -c zzz_trae_gitlab_traffic_quota.rb"
  docker exec -e TRAE_QUOTA_INIT=/tmp/zzz_trae_gitlab_traffic_quota.rb gitlab "$RUBY" /tmp/test_zzz_traffic_quota_skip.rb
  exit 0
fi

token_check
echo "PASS token check zzz_trae_gitlab_traffic_quota.rb (ruby unavailable)"
