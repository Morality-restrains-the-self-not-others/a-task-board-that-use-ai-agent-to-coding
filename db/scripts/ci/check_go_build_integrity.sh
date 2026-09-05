#!/usr/bin/env bash
# Pre-commit / CI gate: check Go services for build integrity.
#
# Scans for Go services with pending changes and verifies:
#   1. go mod tidy -diff (no missing/extra deps)
#   2. go build ./... (compiles cleanly)
#
# Usage:
#   bash db/scripts/ci/check_go_build_integrity.sh          # check changed services
#   bash db/scripts/ci/check_go_build_integrity.sh --all    # check all services
#
# Exit code: 0 = all clean, 1 = issues found.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MODE="${1:-}"
FAILED=()

# List of all Go service directories (relative to repo root).
# Each entry must contain go.mod.
ALL_GO_SERVICES=(
    taskAIComment taskAIEndPoint taskAgentSupport taskAiProvider
    taskAuth taskBill taskCloudService taskContainerGateway
    taskCredentialService taskEvents taskGitOauth taskProjectService
    taskReferral taskSSE taskTaskService taskTenantService
    valueStream go_relayToTrae
    runAll shareLib
)

find_changed_go_services() {
    local changed=()
    for svc in "${ALL_GO_SERVICES[@]}"; do
        if [ -f "$ROOT/$svc/go.mod" ]; then
            # Check if any go files in this service were changed (staged or unstaged)
            if git diff --name-only HEAD 2>/dev/null | grep -q "^$svc/" || \
               git diff --name-only --cached 2>/dev/null | grep -q "^$svc/" || \
               git diff --name-only 2>/dev/null | grep -q "^$svc/"; then
                changed+=("$svc")
            fi
        fi
    done
    echo "${changed[@]}"
}

main() {
    local services=()
    if [ "$MODE" = "--all" ]; then
        services=("${ALL_GO_SERVICES[@]}")
        echo "[go-build-check] Checking ALL Go services (${#services[@]})"
    else
        services=($(find_changed_go_services))
        if [ ${#services[@]} -eq 0 ]; then
            echo "[go-build-check] No Go service changes detected — skipping"
            exit 0
        fi
        echo "[go-build-check] Checking changed Go services: ${services[*]}"
    fi

    for svc in "${services[@]}"; do
        local dir="$ROOT/$svc"
        if [ ! -f "$dir/go.mod" ]; then
            continue
        fi
        echo "  → $svc"

        # 1) go mod tidy -diff
        if (cd "$dir" && go mod tidy -diff 2>&1); then
            echo "    tidy: OK"
        else
            echo "    ❌ tidy: go.sum out of sync — run 'cd $dir && go mod tidy'"
            FAILED+=("$svc:tidy")
        fi

        # 2) go build ./...
        if (cd "$dir" && go build ./... 2>&1); then
            echo "    build: OK"
        else
            echo "    ❌ build: compilation error"
            FAILED+=("$svc:build")
        fi
    done

    if [ ${#FAILED[@]} -gt 0 ]; then
        echo ""
        echo "❌ FAILED: ${FAILED[*]}"
        exit 1
    fi
    echo "[go-build-check] ✅ All clean"
}

main
