# Implementation Plan: Fix Password Reset Datetime Parsing

> Inputs:
> - Design: `docs/superpowers/specs/2026-06-22-fix-password-reset-datetime-parsing.md`
> - Value Stream: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-ddd.md`

## Tasks

### 1. Add parseDateTime() helper
- **File**: `taskAuth/src/db.go`
- **Status**: ✅ Done
- **Verification**: `go build -o taskAuth ./src`
- **Details**: Function tries 4 datetime formats in order: Go microsecond, Go second, RFC 3339 Nano, RFC 3339

### 2. Refactor isPasswordResetTokenValid()
- **File**: `taskAuth/src/db.go` (line 281-295)
- **Status**: ✅ Done
- **Verification**: `curl http://127.0.0.1:8003/api/accounts/users/get-reset-user-info/{token}/` returns 200
- **Details**: Replaced inline two-format parsing with `parseDateTime()` call

### 3. Refactor isActivationTokenValid()
- **File**: `taskAuth/src/db.go` (line 238-252)
- **Status**: ✅ Done
- **Verification**: Existing `TestBootstrapAdminCreatesFreshDB` covers activation flow
- **Details**: Same fix as password reset — use `parseDateTime()` helper

### 4. Run existing tests
- **Command**: `cd taskAuth && go test ./src/ -count=1`
- **Status**: ✅ Done (27/27 pass)
- **Verification**: All tests green

### 5. Rebuild and restart taskAuth
- **Command**: `cd taskAuth && go build -o taskAuth ./src && pkill -f '[/]taskAuth' && ./taskAuth &`
- **Status**: ✅ Done
- **Verification**: Service healthy on port 8003

### 6. End-to-end verification
- **API test**: `curl http://183.250.1.132:18081/api/accounts/users/get-reset-user-info/{token}/` → 200 + identifier
- **Playwright E2E**: Password reset page loads without error
- **Status**: ✅ Done

### 7. Code review
- **Status**: ⬜ Pending (step 8)
- **Details**: Review for correctness, style, and edge cases

### 8. Ship
- **Status**: ⬜ Pending (step 9)
- **Details**: Commit + PR
