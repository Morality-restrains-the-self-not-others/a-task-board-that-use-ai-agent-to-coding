# config.go Companion

`loadGitOauthSSOSecret` 必须走 `confload.ReadAppFragment`，以便合并 `conf-local/auth/git-oauth/django.yaml`。禁止 `os.ReadFile` 直读已跟踪 YAML 取密钥。
