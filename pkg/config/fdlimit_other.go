//go:build !linux && !freebsd

package config

func currentFileDescriptorSoftLimit() (uint64, bool) { return 0, false }
