//go:build linux

package main

import (
	"syscall"
)

// soReusePort is the Linux SO_REUSEPORT socket option number on the
// architectures this monorepo builds for (amd64/arm64 etc., value 15). Go's
// syscall package exports SO_REUSEPORT only for a subset of Linux GOARCH
// values (arm64, not amd64), so define it here instead of importing x/sys/unix.
// The mips*/ppc64*/s390x variants use a different value (0x200) and are not
// build targets here.
const soReusePort = 0xf

func reusePortControl(network, address string, c syscall.RawConn) error {
	var opErr error
	err := c.Control(func(fd uintptr) {
		opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		if opErr != nil {
			return
		}
		opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, soReusePort, 1)
	})
	if err != nil {
		return err
	}
	return opErr
}
