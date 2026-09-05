#!/usr/bin/env bash
# 回归：conf/auth/git-oauth/providers 下全部 GitLab provider 的 redirect_uri
# 须等于 conf/base.yaml 展开的
# ${scheme}://${subdomains.base}/redirect/gitsite/<website-host>/oauth/callback/
# （v2 回调契约；gitsite = target.website 主机名，含 camelCase 子域如 gitlabTencentSh1）
# 禁止硬编码 FQDN；期望主机只来自 base.yaml
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PROVIDERS_DIR="$ROOT/conf/auth/git-oauth/providers"

if [[ ! -d "$PROVIDERS_DIR" ]]; then
  echo "FAIL: missing $PROVIDERS_DIR" >&2
  exit 1
fi

python3 - "$PROVIDERS_DIR" "$ROOT/conf/base.yaml" <<'PY'
import os, re, sys
from pathlib import Path
from urllib.parse import urlparse

try:
    import yaml
except ImportError:
    print("FAIL: PyYAML required", file=sys.stderr)
    raise SystemExit(1)

prov_dir = Path(sys.argv[1])
base_path = Path(sys.argv[2])
base = yaml.safe_load(base_path.read_text(encoding="utf-8")) or {}
_env_default_re = re.compile(r"\$\{(\w+):-([^}]*)\}")
pat = re.compile(r"\$\{(scheme|baseDomain|subdomains\.[A-Za-z][A-Za-z0-9]*)\}")

def resolve_env_default(v: str) -> str:
    def repl(m):
        return os.environ.get(m.group(1), m.group(2) or "")
    return _env_default_re.sub(repl, str(v))

scheme = os.environ.get("PUBLIC_SCHEME") or resolve_env_default(str(base.get("scheme", "https")))
scheme = scheme.strip().lower().rstrip(":/") or "https"
base_domain = os.environ.get("BASE_DOMAIN") or resolve_env_default(str(base.get("baseDomain", "")))
if not base_domain:
    print("FAIL: empty baseDomain", file=sys.stderr)
    raise SystemExit(1)
dm = {"scheme": scheme, "baseDomain": base_domain}
subs = base.get("subdomains") or {}
if isinstance(subs, dict):
    for key, template in subs.items():
        val = str(template).replace("${scheme}", scheme).replace("${baseDomain}", base_domain)
        dm[f"subdomains.{key}"] = val

def resolve(s: str) -> str:
    return pat.sub(lambda m: dm.get(m.group(1), m.group(0)), s)

errors = 0
checked = 0
for path in sorted(prov_dir.glob("*.yaml")):
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if str(data.get("provider") or "").strip().lower() != "gitlab":
        continue
    checked += 1
    target = data.get("target") or {}
    ru = str(target.get("redirect_uri") or "")
    website = str(target.get("website") or "")
    allowed_raw = str((data.get("service") or {}).get("allowedHost") or "")
    resolved = resolve(ru)
    website_resolved = resolve(website)
    allowed = resolve(allowed_raw)
    if "${" in resolved or "${" in website_resolved:
        print(f"FAIL: {path.name} unresolved template redirect_uri={resolved!r} website={website_resolved!r}", file=sys.stderr)
        errors += 1
        continue
    host = urlparse(website_resolved if "://" in website_resolved else "https://" + website_resolved).hostname or ""
    if not host:
        print(f"FAIL: {path.name} empty website host from {website_resolved!r}", file=sys.stderr)
        errors += 1
        continue
    expected = f"{scheme}://{dm.get('subdomains.base','')}/redirect/gitsite/{host}/oauth/callback/"
    expected_allowed = f"{scheme}://{dm.get('subdomains.base','')}"
    if resolved != expected:
        print(f"FAIL: {path.name} redirect_uri={resolved} want {expected}", file=sys.stderr)
        errors += 1
    if allowed and allowed != expected_allowed:
        print(f"FAIL: {path.name} allowedHost={allowed} want {expected_allowed}", file=sys.stderr)
        errors += 1
    text = path.read_text(encoding="utf-8")
    if not re.search(r"^\s*(redirect_uri|allowedHost):.*subdomains\.base", text, re.M):
        print(f"FAIL: {path.name} redirect_uri/allowedHost should use ${{subdomains.base}}", file=sys.stderr)
        errors += 1
    for line in text.splitlines():
        if re.match(r"^\s*(redirect_uri|allowedHost):", line) and re.search(
            r"[a-z0-9.-]+\.(com|net|org|io)\b", line
        ):
            print(f"FAIL: {path.name} hardcodes a domain; use ${{subdomains.*}} templates only: {line}", file=sys.stderr)
            errors += 1
            break
    else:
        print(f"OK {path.name} redirect_uri={resolved}")

if checked == 0:
    print("FAIL: no GitLab provider YAML found", file=sys.stderr)
    raise SystemExit(1)
if errors:
    raise SystemExit(1)
print(f"OK checked {checked} GitLab provider YAML(s)")
PY
