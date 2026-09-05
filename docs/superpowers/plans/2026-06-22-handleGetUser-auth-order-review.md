# Review: Fix handleGetUser Auth Order

✅ **PASS** — 0 issues.

**Change**: `taskAuth/src/auth_users.go:handleGetUser` — when `resolveTokenUserID` fails, check `requireInternalSecret(r)` before returning 401.

**Correctness**: ✅ Internal service calls with valid secret now proceed. Unauthorized calls still get 401. Same-user token auth unchanged. Matches `handleGetSuperAdmin` pattern.

**Tests**: ✅ 27/27 pass. Login via curl verified.
