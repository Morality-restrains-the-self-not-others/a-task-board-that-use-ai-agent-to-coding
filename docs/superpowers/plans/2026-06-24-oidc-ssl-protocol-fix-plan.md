# Implementation Plan: OIDC SSL Protocol Fix

> 输入:
> - 设计文档: `docs/specs/oidc-ssl-debug/design.md`
> - 价值流: `docs/superpowers/plans/2026-06-24-oidc-ssl-protocol-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-24-oidc-ssl-protocol-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-24-oidc-ssl-protocol-fix-ddd.md` (skipped)

## Overview

纯基础设施修复。在 GitLab 容器内注入 Rails initializer，修正 `openid_connect` gem 的 `SWD.url_builder` 默认为 `URI::HTTPS` 的 bug，恢复 HTTP issuer 的正确 scheme 传递。

## Phase 1: Immediate Fix (✅ Applied)

### Task 1.1: Create GitLab Rails Initializer

- **Status**: ✅ Done
- **File**: `/opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb`
- **Command**: `docker exec gitlab bash -c 'cat > .../zzz_fix_oidc_http.rb << "RUBY" ...'`
- **Verification**: `docker exec gitlab cat .../zzz_fix_oidc_http.rb`

### Task 1.2: Restart GitLab

- **Status**: ✅ Done
- **Command**: `docker exec gitlab gitlab-ctl restart`
- **Verification**: `docker exec gitlab gitlab-rails runner "puts 'ready'"` → ready

### Task 1.3: Verify OIDC Discovery Protocol

- **Status**: ✅ Done
- **Command**: `docker exec gitlab gitlab-rails runner "require 'openid_connect'; ... Resource.new(uri).endpoint"`
- **Expected**: `http://183.250.1.132:8003/.well-known/openid-configuration`
- **Result**: ✅ Output matches expected

### Task 1.4: Verify Full OIDC Discovery

- **Status**: ✅ Done
- **Command**: `docker exec gitlab gitlab-rails runner "OpenIDConnect::Discovery::Provider::Config.discover!('http://183.250.1.132:8003')"`
- **Expected**: SUCCESS — returns config with authorization_endpoint, token_endpoint, issuer
- **Result**: ✅ Discovery succeeded

### Task 1.5: End-to-End SSO Verification

- **Status**: ⬜ Manual verification
- **Command**: Browser → `http://183.250.1.132:8012/users/sign_in` → click "taskAuth SSO"
- **Expected**: Redirect to taskAuth authorize page, user authenticates, redirected back to GitLab
- **Result**: Pending manual test

## Phase 2: Persistence (Future)

### Task 2.1: Integrate Initializer into gitService Run Script

- **Status**: ⬜ Not started
- **File**: `gitService/run.sh` or `gitService/scripts/sync_omniauth_oidc.sh`
- **Scope**: After GitLab is ready, check and create initializer if missing; then restart Rails
- **Test**: Rebuild GitLab container → run.sh → initializer exists → discovery uses HTTP

### Task 2.2: Document in Runbook

- **Status**: ⬜ Not started
- **File**: `docs/specs/oidc-ssl-debug/design.md` (already covers this)
- **Scope**: Ensure design doc covers the exact initializer content and verification steps

## Files Changed

| File | Change | Status |
|------|--------|--------|
| GitLab container: `.../initializers/zzz_fix_oidc_http.rb` | New file (3 lines Ruby) | ✅ |
| `docs/superpowers/plans/*-value-stream.md` | New value stream doc | ✅ |
| `conf/value-stream.yaml` | Appended `oidc-ssl-protocol-fix` stream | ✅ |
| `docs/specs/oidc-ssl-debug/design.md` | Design doc with root cause + fix | ✅ |
| `docs/specs/oidc-ssl-debug/reproduce_ssl_error.py` | Diagnostic script | ✅ |
