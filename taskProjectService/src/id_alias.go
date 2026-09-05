package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Overflow IDs from the 2026-08-21 inventory (OPT-20260821-013).
// Old genID used UnixNano()*1000+seq which overflowed int64. New IDs are
// positive snowflake; these remain stored primary keys until expand/contract.
// Do not lock-table UPDATE the PK.
const (
	idAliasKindWorkspace = "workspace"
	idAliasKindProject   = "project"
)

// KnownOverflowWorkspaceIDs is the live workspace PK inventory (2 rows).
var KnownOverflowWorkspaceIDs = []string{
	"ws_-2309487803472456748",
	"ws_-2740859684112864748",
}

// KnownOverflowProjectIDs is the live project PK inventory (1 row).
var KnownOverflowProjectIDs = []string{
	"proj_-2304947540687519745",
}

var overflowNegativeIDRe = regexp.MustCompile(`^(ws|proj)_-\d+$`)

// idAliases maps kind+"\x00"+canonical snowflake → stored overflow PK.
// Empty in production = identity (overflow PKs keep working as opaque strings).
var (
	idAliasMu sync.RWMutex
	idAliases = map[string]string{}
)

func aliasKey(kind, canonical string) string {
	return kind + "\x00" + canonical
}

// IsOverflowNegativeID reports whether id is a ws_- / proj_- overflow PK.
// Matching is string-only; the suffix is never parsed as int64.
func IsOverflowNegativeID(id string) bool {
	return overflowNegativeIDRe.MatchString(strings.TrimSpace(id))
}

func clearIDAliases() {
	idAliasMu.Lock()
	defer idAliasMu.Unlock()
	idAliases = map[string]string{}
}

// RegisterIDAlias records canonical snowflake → stored overflow PK for dual-write.
// Production stays empty until expand-phase loads from project_id_alias (plan only).
func RegisterIDAlias(kind, canonical, stored string) error {
	kind = strings.TrimSpace(kind)
	canonical = strings.TrimSpace(canonical)
	stored = strings.TrimSpace(stored)
	if kind != idAliasKindWorkspace && kind != idAliasKindProject {
		return fmt.Errorf("unknown id alias kind %q", kind)
	}
	if canonical == "" {
		return fmt.Errorf("canonical id is empty")
	}
	if IsOverflowNegativeID(canonical) {
		return fmt.Errorf("canonical id %q must be a positive snowflake, not overflow", canonical)
	}
	if !IsOverflowNegativeID(stored) {
		return fmt.Errorf("stored id %q is not an overflow negative id", stored)
	}
	wantPrefix := "ws_"
	if kind == idAliasKindProject {
		wantPrefix = "proj_"
	}
	if !strings.HasPrefix(stored, wantPrefix) {
		return fmt.Errorf("stored id %q prefix does not match kind %s", stored, kind)
	}
	idAliasMu.Lock()
	idAliases[aliasKey(kind, canonical)] = stored
	idAliasMu.Unlock()
	logInfo(fmt.Sprintf("id_alias registered: kind=%s canonical=%s stored=%s", kind, canonical, stored), "")
	return nil
}

func resolveStoredID(kind, incoming string) string {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		return incoming
	}
	idAliasMu.RLock()
	stored, ok := idAliases[aliasKey(kind, incoming)]
	idAliasMu.RUnlock()
	if !ok || stored == "" {
		return incoming
	}
	return stored
}

func resolveRequestID(kind, incoming, traceID string) string {
	stored := resolveStoredID(kind, incoming)
	if stored != incoming {
		logInfo(fmt.Sprintf("id_alias resolved: kind=%s incoming=%s stored=%s", kind, incoming, stored), traceID)
	}
	return stored
}
