// Package netacl implements a simple source-IP allow list for TCP listeners.
//
// It is a coarse network filter, not authentication: it trusts the TCP peer
// address only (never proxy headers) and cannot stop address spoofing on a
// shared network segment.
package netacl

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// ACL is a parsed allow list. A nil or empty ACL allows everything.
type ACL struct {
	prefixes []netip.Prefix
}

// Parse builds an ACL from entries that are single IPs ("192.168.1.20",
// "fe80::1") or CIDR ranges ("192.168.1.0/24"). An invalid entry is an error:
// silently ignoring it would make a typo widen access.
func Parse(entries []string) (*ACL, error) {
	acl := &ACL{}
	for _, raw := range entries {
		e := strings.TrimSpace(raw)
		if e == "" {
			return nil, fmt.Errorf("allowed_ips: empty entry")
		}
		if strings.Contains(e, "/") {
			p, err := netip.ParsePrefix(e)
			if err != nil {
				return nil, fmt.Errorf("allowed_ips: invalid CIDR %q: %w", raw, err)
			}
			acl.prefixes = append(acl.prefixes, normalizePrefix(p))
			continue
		}
		a, err := netip.ParseAddr(e)
		if err != nil {
			return nil, fmt.Errorf("allowed_ips: invalid IP %q", raw)
		}
		a = a.Unmap()
		acl.prefixes = append(acl.prefixes, netip.PrefixFrom(a, a.BitLen()))
	}
	return acl, nil
}

// normalizePrefix converts an IPv4-mapped IPv6 prefix (::ffff:a.b.c.d/n) to
// its IPv4 form so it matches unmapped client addresses.
func normalizePrefix(p netip.Prefix) netip.Prefix {
	a := p.Addr()
	if a.Is4In6() && p.Bits() >= 96 {
		return netip.PrefixFrom(a.Unmap(), p.Bits()-96).Masked()
	}
	return p.Masked()
}

// Active reports whether the ACL restricts access.
func (a *ACL) Active() bool { return a != nil && len(a.prefixes) > 0 }

// Len returns the number of configured entries.
func (a *ACL) Len() int {
	if a == nil {
		return 0
	}
	return len(a.prefixes)
}

// AllowedIP reports whether ip may connect. Loopback is always allowed.
func (a *ACL) AllowedIP(ip netip.Addr) bool {
	if !a.Active() {
		return true
	}
	ip = ip.Unmap().WithZone("")
	if ip.IsLoopback() {
		return true
	}
	for _, p := range a.prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// Allowed reports whether a connection from addr may proceed. Addresses that
// cannot be parsed as an IP are denied when the ACL is active.
func (a *ACL) Allowed(addr net.Addr) bool {
	if !a.Active() {
		return true
	}
	ip, ok := addrIP(addr)
	if !ok {
		return false
	}
	return a.AllowedIP(ip)
}

func addrIP(addr net.Addr) (netip.Addr, bool) {
	if t, ok := addr.(*net.TCPAddr); ok {
		ip, ok := netip.AddrFromSlice(t.IP)
		return ip, ok
	}
	if addr == nil {
		return netip.Addr{}, false
	}
	ap, err := netip.ParseAddrPort(addr.String())
	if err != nil {
		return netip.Addr{}, false
	}
	return ap.Addr(), true
}

// Wrap returns ln with connections from disallowed addresses closed
// immediately and never returned from Accept. When the ACL is not active, ln
// is returned unchanged. logf, if non-nil, is called (throttled per IP) when a
// connection is denied.
func Wrap(ln net.Listener, acl *ACL, name string, logf func(format string, args ...any)) net.Listener {
	if !acl.Active() {
		return ln
	}
	return &listener{Listener: ln, acl: acl, name: name, logf: logf, seen: map[string]time.Time{}}
}

type listener struct {
	net.Listener
	acl  *ACL
	name string
	logf func(string, ...any)

	mu   sync.Mutex
	seen map[string]time.Time
}

const (
	logInterval = time.Minute
	maxSeen     = 1024
)

func (l *listener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if l.acl.Allowed(conn.RemoteAddr()) {
			return conn, nil
		}
		l.deny(conn)
	}
}

func (l *listener) deny(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	_ = conn.Close()
	if l.logf == nil {
		return
	}
	host := remote
	if h, _, err := net.SplitHostPort(remote); err == nil {
		host = h
	}
	l.mu.Lock()
	now := time.Now()
	if len(l.seen) >= maxSeen {
		l.seen = map[string]time.Time{}
	}
	last, ok := l.seen[host]
	quiet := ok && now.Sub(last) < logInterval
	if !quiet {
		l.seen[host] = now
	}
	l.mu.Unlock()
	if !quiet {
		l.logf("%s: denied connection from %s (not in allowed_ips)", l.name, host)
	}
}
