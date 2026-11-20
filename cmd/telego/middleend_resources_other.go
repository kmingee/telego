//go:build !linux && !freebsd

package main

func currentFileDescriptorLimit() (fileDescriptorLimit, error) {
	return fileDescriptorLimit{}, errFileDescriptorLimitUnavailable
}
