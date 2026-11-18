package main

import (
	"context"
	"fmt"
	"net"
)

type candidatesKey struct{}

// fallbackResolver resolves a hostname to all of its addresses. The first
// address is returned as the destination (and is what the rules check), the
// full list is stashed in the context so the dialer can fall back to the others.
type fallbackResolver struct {
	// network is "ip" (any family), "ip4" or "ip6"
	network string
}

func (r fallbackResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, r.network, name)
	if err != nil {
		return ctx, nil, err
	}
	if len(ips) == 0 {
		return ctx, nil, fmt.Errorf("no addresses found for %s", name)
	}
	return context.WithValue(ctx, candidatesKey{}, ips), ips[0], nil
}

// newDialer returns a dial function that tries the requested address first and
// then the other addresses the hostname resolved to (e.g. IPv4 when IPv6 fails).
// network restricts the address family: "ip" (any), "ip4" or "ip6".
// Fallback addresses are subject to the same internal-IP policy as the first one.
func newDialer(network string, allowInternal bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return func(ctx context.Context, _, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		first := net.ParseIP(host)
		if first == nil {
			return d.DialContext(ctx, "tcp", addr)
		}

		targets := []net.IP{first}
		if all, ok := ctx.Value(candidatesKey{}).([]net.IP); ok {
			for _, ip := range all {
				if ip.Equal(first) {
					continue
				}
				if !allowInternal && isInternalIP(ip) {
					continue
				}
				targets = append(targets, ip)
			}
		}

		var lastErr error
		for _, ip := range targets {
			if !familyMatches(network, ip) {
				continue
			}
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
			if ctx.Err() != nil {
				break
			}
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("no %s address available for %s", network, addr)
		}
		return nil, lastErr
	}
}

func familyMatches(network string, ip net.IP) bool {
	switch network {
	case "ip4":
		return ip.To4() != nil
	case "ip6":
		return ip.To4() == nil
	}
	return true
}
