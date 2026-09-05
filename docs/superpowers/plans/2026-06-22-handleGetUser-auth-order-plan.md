# Plan: Fix handleGetUser Auth Order

## Tasks

### 1. Fix auth order in handleGetUser
- **File**: `taskAuth/src/auth_users.go`
- **Status**: ✅ Done
- **Details**: When `resolveTokenUserID` returns error, check `requireInternalSecret(r)` before returning 401

### 2. Run tests
- **Command**: `cd taskAuth && go test ./src/ -count=1`
- **Status**: ✅ Done (27/27 pass)

### 3. Verify login flow
- **Command**: curl login API with valid password hash
- **Status**: ✅ Done (token returned, user authenticated)
