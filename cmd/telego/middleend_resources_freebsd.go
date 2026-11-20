//go:build freebsd

package main

import "golang.org/x/sys/unix"

func currentFileDescriptorLimit() (fileDescriptorLimit, error) {
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return fileDescriptorLimit{}, err
	}
	return fileDescriptorLimit{soft: uint64(limit.Cur), hard: uint64(limit.Max)}, nil
}
