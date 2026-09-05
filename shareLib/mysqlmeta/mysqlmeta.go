// Package mysqlmeta converts mysql-CLI scripts into server-executable SQL.
//
// mysql CLI supports DELIMITER (client-only). Go database/sql sends COM_QUERY
// directly, so DELIMITER lines must be removed and alternate terminators
// (typically "//") rewritten to ";". apply_datamigrate.sh keeps the original
// file and relies on DELIMITER for CREATE PROCEDURE bodies containing ';'.
package mysqlmeta

import "strings"

// StripMySQLClientMeta removes mysql CLI-specific DELIMITER directives and
// rewrites custom statement terminators to ";".
//
// This allows SQL files written for the mysql CLI (which may use DELIMITER //
// around CREATE PROCEDURE bodies) to be executed through Go's database/sql
// driver, which sends statements directly to the server via COM_QUERY.
func StripMySQLClientMeta(sql string) string {
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
			// Preserve indentation; replace only the trailing custom delimiter.
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
