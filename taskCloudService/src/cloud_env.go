package main

import (
	"os"
	"strings"
)

func useInMemoryCloud() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("USE_IN_MEMORY_CLOUD")), "false") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("USE_IN_MEMORY_SERVICES")), "true") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("USE_IN_MEMORY_CLOUD")), "true")
}
