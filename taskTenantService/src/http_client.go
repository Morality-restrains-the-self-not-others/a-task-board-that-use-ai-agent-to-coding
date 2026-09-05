package main

import (
	"time"

	"tracelog"
)

// Shared outbound client for Django / peer Go services (no env proxy).
var httpClient = tracelog.DirectClient(15 * time.Second)
