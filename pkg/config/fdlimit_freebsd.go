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
	// x/sys/unix uses int64 for FreeBSD's Rlimit fields, unlike Linux.
	return uint64(limit.Cur), true
}
