package main

import (
	"context"
	"net"
	"testing"

	"github.com/armon/go-socks5"
)

func TestIsInternalIP(t *testing.T) {
	cases := []struct {
		ip       string
		internal bool
	}{
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.5.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.63.0.1", false},
		{"100.128.0.1", false},
		{"0.0.0.0", true},
		{"::1", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"2001:4860:4860::8888", false},
	}

	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("failed to parse %s", tc.ip)
		}
		got := isInternalIP(ip)
		if got != tc.internal {
			t.Errorf("isInternalIP(%s) = %v, want %v", tc.ip, got, tc.internal)
		}
	}
}

func TestInternalFilterRule(t *testing.T) {
	ctx := context.Background()

	blocked := newInternalFilterRule(false)
	allowed := newInternalFilterRule(true)

	reqPrivate := &socks5.Request{
		Command:  socks5.ConnectCommand,
		DestAddr: &socks5.AddrSpec{IP: net.ParseIP("192.168.1.10"), Port: 80},
	}
	reqPublic := &socks5.Request{
		Command:  socks5.ConnectCommand,
		DestAddr: &socks5.AddrSpec{IP: net.ParseIP("1.1.1.1"), Port: 443},
	}

	if _, ok := blocked.Allow(ctx, reqPrivate); ok {
		t.Error("expected private IP blocked when allow-internal=false")
	}
	if _, ok := blocked.Allow(ctx, reqPublic); !ok {
		t.Error("expected public IP allowed when allow-internal=false")
	}
	if _, ok := allowed.Allow(ctx, reqPrivate); !ok {
		t.Error("expected private IP allowed when allow-internal=true")
	}
}
