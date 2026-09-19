package main

import (
	"context"
	"net"

	"github.com/armon/go-socks5"
)

// internalFilterRule blocks connections to internal/private destinations
// unless allowInternal is true. Connect is always required; Bind/Associate
// follow the same IP policy when enabled by the base rule set.
type internalFilterRule struct {
	allowInternal bool
	base          socks5.RuleSet
}

func newInternalFilterRule(allowInternal bool) socks5.RuleSet {
	return &internalFilterRule{
		allowInternal: allowInternal,
		base:          socks5.PermitAll(),
	}
}

func (r *internalFilterRule) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	ctx, ok := r.base.Allow(ctx, req)
	if !ok {
		return ctx, false
	}
	if r.allowInternal {
		return ctx, true
	}

	ip := req.DestAddr.IP
	if ip == nil || isInternalIP(ip) {
		return ctx, false
	}
	return ctx, true
}

// isInternalIP reports whether ip is loopback, private, link-local,
// unspecified, or CGNAT (RFC 6598) shared address space.
func isInternalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// RFC 6598 Carrier-grade NAT: 100.64.0.0/10
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}
