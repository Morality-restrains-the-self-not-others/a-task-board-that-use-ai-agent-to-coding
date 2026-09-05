package main

import (
	"strings"
)

// stripMySQLClientMeta converts mysql-CLI scripts into server-executable SQL.
//
// mysql CLI supports DELIMITER (client-only). Go database/sql sends COM_QUERY
// directly, so DELIMITER lines must be removed and alternate terminators
// (typically "//") rewritten to ";". apply_datamigrate.sh keeps the original
// file and relies on DELIMITER for CREATE PROCEDURE bodies containing ';'.
func stripMySQLClientMeta(sql string) string {
	lines := strings.Split(sql, "\n")
	out := make([]string, 0, len(lines))
	delim := ";"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "DELIMITER") {
			rest := strings.TrimSpace(trimmed[len("DELIMITER"):])
			if rest == "" {
				delim = ";"
			} else {
				delim = rest
			}
			continue
		}
		if delim != ";" && strings.HasSuffix(trimmed, delim) {
			withoutTrailSpace := strings.TrimRight(line, " \t\r")
			if strings.HasSuffix(withoutTrailSpace, delim) {
				base := strings.TrimRight(strings.TrimSuffix(withoutTrailSpace, delim), " \t")
				line = base + ";"
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
