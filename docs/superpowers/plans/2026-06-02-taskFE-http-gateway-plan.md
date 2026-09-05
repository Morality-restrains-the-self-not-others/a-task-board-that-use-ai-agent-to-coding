# Vue Frontend → Gateway HTTP (Dev Only) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the Vue frontend's `apiBaseUrl` from HTTPS to HTTP for local development, eliminating self-signed certificate friction.

**Architecture:** Single configuration value change in `conf/vue/config.yaml`. The `apiBaseUrl` flows through `scripts/conf-read.py` → `vite.config.js` → frontend `config.js` → browser requests. No code changes needed.

**Tech Stack:** YAML configuration, no code changes.

---

### Task 1: Change apiBaseUrl from HTTPS to HTTP

**Files:**
- Modify: `conf/vue/config.yaml:6`

- [ ] **Step 1: Update the config file**

Change line 6 of `conf/vue/config.yaml`:

```yaml
# Before:
apiBaseUrl: https://172.20.10.3:8443

# After:
apiBaseUrl: http://172.20.10.3:8080
```

Run: `sed -i '' 's|apiBaseUrl: https://172.20.10.3:8443|apiBaseUrl: http://172.20.10.3:8080|' conf/vue/config.yaml`

- [ ] **Step 2: Verify the change**

```bash
grep 'apiBaseUrl' conf/vue/config.yaml
```

Expected output:
```
apiBaseUrl: http://172.20.10.3:8080
```

- [ ] **Step 3: Verify conf-read.py still parses correctly**

```bash
python3 scripts/conf-read.py snapshot-json 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); print('apiBaseUrl:', d.get('vue',{}).get('apiBaseUrl','NOT FOUND'))"
```

Expected: `apiBaseUrl: http://172.20.10.3:8080`

- [ ] **Step 4: Commit**

```bash
git add conf/vue/config.yaml
git commit -m "chore: switch taskFE dev apiBaseUrl from HTTPS to HTTP

Change local dev config to use HTTP :8080 instead of HTTPS :8443
to eliminate self-signed certificate friction in development.
Production config remains unchanged.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Verification (Manual)

After the change, restart the Vite dev server and verify:

1. Open browser at `http://172.20.10.3:4000`
2. Open DevTools → Network tab
3. Perform login and basic operations
4. Confirm API requests go to `http://172.20.10.3:8080/api/...` (not https)
5. Confirm no certificate warnings appear
6. Confirm login, workspace listing, and task operations work correctly
