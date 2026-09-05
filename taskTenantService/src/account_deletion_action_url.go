package main

import (
	"fmt"
	"strings"
)

func tenantMembersActionURL(tenantID string) string {
	return fmt.Sprintf("/tenant/%s/people/manage/", strings.TrimSpace(tenantID))
}
