package main

import "testing"

func TestUnauthenticatedServerIsLoopbackOnly(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8080", "[::1]:8080", "localhost:8080"} {
		if err := validateListenAddress(addr, false); err != nil {
			t.Fatalf("loopback %s rejected: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:8080", ":8080", "192.0.2.10:8080"} {
		if err := validateListenAddress(addr, false); err == nil {
			t.Fatalf("unauthenticated non-loopback %s accepted", addr)
		}
		if err := validateListenAddress(addr, true); err != nil {
			t.Fatalf("authenticated address %s rejected: %v", addr, err)
		}
	}
}
