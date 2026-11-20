// Copyright (c) 2026 Telego contributors.
// Licensed under the Apache License, Version 2.0. See LICENSE.
//go:build linux || freebsd || dragonfly

package gnet

import (
	"net"
	"testing"
	"time"
)

func unusedTCP4Address(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func assertTCP4AddressReleased(t *testing.T, address string) {
	t.Helper()
	ln, err := net.Listen("tcp4", address)
	if err != nil {
		t.Fatalf("listener descriptor or bound port leaked at %s: %v", address, err)
	}
	_ = ln.Close()
}

func TestTelegoPartialListenerCreationReleasesEarlierSockets(t *testing.T) {
	address := unusedTCP4Address(t)
	blocker, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()

	listeners, _, err := createListeners(
		[]string{"tcp4://" + address, "tcp4://" + blocker.Addr().String()},
		WithReusePort(false), WithReuseAddr(false),
	)
	if err == nil {
		for _, ln := range listeners {
			ln.close()
		}
		t.Fatal("expected second listener bind to fail")
	}
	assertTCP4AddressReleased(t, address)
}

func TestTelegoKeepAliveSetupFailureReleasesSocket(t *testing.T) {
	address := unusedTCP4Address(t)
	// Positive but subsecond duration is deliberately invalid for the TCP
	// keepalive sockopts. Failure happens after the listener is bound.
	listeners, _, err := createListeners(
		[]string{"tcp4://" + address},
		WithReusePort(false), WithTCPKeepAlive(time.Nanosecond),
	)
	if err == nil {
		for _, ln := range listeners {
			ln.close()
		}
		t.Fatal("expected keepalive setup to fail")
	}
	assertTCP4AddressReleased(t, address)
}
