package main

import (
	"os"
	"path/filepath"
	"strings"
)

// appendOnlineServiceSrcOverlayMounts bind-mounts the monorepo
// trae-agent/onlineServiceJS/src directory into the container so new modules
// (e.g. layerFileContent.mjs) are available without rebuilding the registry image.
// Disabled when RELAY_OVERLAY_ONLINE_SERVICE_SRC is 0/false/off, or when src is missing.
func appendOnlineServiceSrcOverlayMounts(runArgs []string, repoRoot string) []string {
	overlayFlag := strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_OVERLAY_ONLINE_SERVICE_SRC")))
	if overlayFlag == "0" || overlayFlag == "false" || overlayFlag == "off" {
		return runArgs
	}
	if strings.TrimSpace(repoRoot) == "" {
		return runArgs
	}
	srcDir := filepath.Join(repoRoot, "trae-agent", "onlineServiceJS", "src")
	st, err := os.Stat(srcDir)
	if err != nil || !st.IsDir() {
		return runArgs
	}
	// Directory mount covers every .mjs under src (server + helpers); file-list
	// whitelist previously missed new modules and broke imports after overlaying server.mjs.
	runArgs = append(runArgs, "-v", srcDir+":/app/onlineServiceJS/src:ro")
	appendLog("[relayToTrae] overlay mount onlineServiceJS/src from monorepo")
	return runArgs
}
