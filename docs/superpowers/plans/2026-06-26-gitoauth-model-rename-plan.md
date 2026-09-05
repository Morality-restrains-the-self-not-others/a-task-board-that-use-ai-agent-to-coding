# Plan: Rename GithubAppUserCredential → GitOAuthAppUserCredential

### Task 1: Rename models + fields in models.py
- `GithubAppUserCredential` → `GitOAuthAppUserCredential`
- `github_user_id` → `remote_user_id`
- `github_login` → `remote_login`
- `GithubAppAccessTokenUseAudit` → `GitOAuthAppAccessTokenUseAudit`
- `GithubTaskCredentialAudit` → `GitOAuthTaskCredentialAudit`

### Task 2: Create migration (RenameModel + RenameField)
### Task 3: Update all references (~15 files)
### Task 4: Run tests
