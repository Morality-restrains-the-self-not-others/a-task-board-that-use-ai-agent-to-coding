package main

import (
	"fmt"
	"regexp"
)

// Service segment allows CamelCase (e.g. taskFE) to match runAll service names;
// table/column stay snake_case lowercase.
var fieldNameRe = regexp.MustCompile(`^([a-zA-Z0-9-]+)\.([a-z0-9_]+)\.([a-z0-9_]+)$`)

func ParseFieldName(name string) (provider, table, column string, err error) {
	m := fieldNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", "", "", fmt.Errorf("field name %q must be <service>.<table>.<column>", name)
	}
	return m[1], m[2], m[3], nil
}
