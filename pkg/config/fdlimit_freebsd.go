//go:build freebsd

package config

import "golang.org/x/sys/unix"

// FreeBSD exposes RLIMIT_NOFILE with the same soft/hard semantics as Linux.
// Enforce the automatic cap here too, rather than allowing unlimited clients.
func currentFileDescriptorSoftLimit() (uint64, bool) {
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return 0, false
	}
	return limit.Cur, true
}
