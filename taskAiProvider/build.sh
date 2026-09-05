#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
mkdir -p bin
echo "==> Building taskAiProvider (./src -> bin/taskAiProvider)..."
go build -buildvcs=false -o bin/taskAiProvider ./src
echo "==> Done: $ROOT/bin/taskAiProvider"

# ── Frontend dist — SSoT: conf/base.yaml (scheme + subdomains.www) ──────────
# Missing dist caused the 2026-08-06 SSO 404 (handleSPA silently ServeFile 404).
# See .learnings/OPTIMIZATION_TODOS.md OPT-20260806-024.
base_yaml="${ROOT}/../conf/base.yaml"
if [ ! -f "$base_yaml" ]; then
  echo "ERROR: conf/base.yaml not found ($base_yaml); cannot resolve main SaaS origin" >&2
  exit 1
fi

# Expand base.yaml ${VAR:-default} env-override tokens (PUBLIC_SCHEME / BASE_DOMAIN).
expand_token() { # $1 = raw value (e.g. ${PUBLIC_SCHEME:-https})
  local raw="$1" content="" var="" dflt=""
  if [[ "$raw" == \$\{*\:-*\} ]]; then
    content="${raw#\$\{}"; content="${content%\}}"
    var="${content%%:-*}"; dflt="${content#*:-}"
    printf '%s' "${!var:-$dflt}"
  else
    printf '%s' "$raw"
  fi
}

scheme_raw="$(sed -nE 's/^scheme:[[:space:]]*([^[:space:]#]+).*/\1/p' "$base_yaml" | head -1)"
base_domain_raw="$(sed -nE 's/^baseDomain:[[:space:]]*([^[:space:]#]+).*/\1/p' "$base_yaml" | head -1)"
www_raw="$(sed -nE 's/^[[:space:]]*www:[[:space:]]*([^[:space:]#]+).*/\1/p' "$base_yaml" | head -1)"
[ -n "$scheme_raw" ] && [ -n "$www_raw" ] || {
  echo "ERROR: cannot resolve scheme/subdomains.www from $base_yaml" >&2
  exit 1
}
scheme="$(expand_token "$scheme_raw")"
base_domain="$(expand_token "$base_domain_raw")"
www_domain="${www_raw//\$\{baseDomain\}/$base_domain}"
main_saas_origin="${scheme}://${www_domain}"
echo "==> Main SaaS origin: $main_saas_origin"

if [ -d frontend/node_modules ]; then
  echo "==> frontend node_modules present, skipping npm ci"
else
  echo "==> Installing frontend deps (npm ci)..."
  (cd frontend && npm ci --no-audit --no-fund)
fi

echo "==> Building frontend (vite) -> frontend/dist..."
(
  cd frontend
  VITE_MAIN_SAAS_ORIGIN="$main_saas_origin" \
  VITE_MAIN_SAAS_ADMIN_SSO_ORIGIN="$main_saas_origin" \
  VITE_MAIN_SAAS_VENDOR_SSO_ORIGIN="$main_saas_origin" \
  VITE_MAIN_SAAS_SESSION_ORIGIN="$main_saas_origin" \
  npm run build
)
if [ ! -f frontend/dist/index.html ]; then
  echo "ERROR: frontend build did not produce frontend/dist/index.html" >&2
  exit 1
fi
echo "==> Done: $ROOT/frontend/dist/index.html"
