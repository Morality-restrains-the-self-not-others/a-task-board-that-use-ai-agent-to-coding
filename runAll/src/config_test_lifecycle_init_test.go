package main

import "os"

func init() {
	if os.Getenv("RUNALL_LIFECYCLE_STRICT") == "" {
		_ = os.Setenv("RUNALL_LIFECYCLE_STRICT", "0")
	}
}
