package netacl

import (
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseErrors(t *testing.T) {
	for _, bad := range []string{"", "  ", "nope", "300.1.1.1", "10.0.0.0/33", "10.0.0.1/", "http://10.0.0.1"} {
		if _, err := Parse([]string{"10.0.0.1", bad}); err == nil {
			t.Errorf("Parse(%q) expected error", bad)
		}
	}
}

func TestAllowed(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		ip      string
		want    bool
	}{
		{"empty allows all", nil, "8.8.8.8", true},
		{"single ip match", []string{"192.168.1.20"}, "192.168.1.20", true},
		{"single ip miss", []string{"192.168.1.20"}, "192.168.1.21", false},
		{"cidr match", []string{"192.168.1.0/24"}, "192.168.1.200", true},
		{"cidr miss", []string{"192.168.1.0/24"}, "192.168.2.1", false},
		{"cidr unmasked host bits", []string{"192.168.1.77/24"}, "192.168.1.5", true},
		{"ipv6 match", []string{"fd00::1"}, "fd00::1", true},
		{"ipv6 cidr", []string{"fd00::/64"}, "fd00::abcd", true},
		{"ipv6 miss", []string{"fd00::1"}, "fd00::2", false},
		{"mapped client vs v4 entry", []string{"192.168.1.20"}, "::ffff:192.168.1.20", true},
		{"mapped entry vs v4 client", []string{"::ffff:192.168.1.20"}, "192.168.1.20", true},
		{"mapped cidr entry", []string{"::ffff:192.168.1.0/120"}, "192.168.1.9", true},
		{"loopback v4 always", []string{"10.0.0.1"}, "127.0.0.1", true},
		{"loopback v6 always", []string{"10.0.0.1"}, "::1", true},
		{"mapped loopback always", []string{"10.0.0.1"}, "::ffff:127.0.0.1", true},
		{"whitespace entry", []string{" 10.0.0.1 "}, "10.0.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl, err := Parse(tt.entries)
			if err != nil {
				t.Fatal(err)
			}
			if got := acl.AllowedIP(netip.MustParseAddr(tt.ip)); got != tt.want {
				t.Errorf("AllowedIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestNilACLAllows(t *testing.T) {
	var acl *ACL
	if !acl.Allowed(&net.TCPAddr{IP: net.ParseIP("8.8.8.8")}) || acl.Active() || acl.Len() != 0 {
		t.Error("nil ACL should allow everything and be inactive")
	}
}

func TestUnknownAddrDeniedWhenActive(t *testing.T) {
	acl, _ := Parse([]string{"10.0.0.1"})
	if acl.Allowed(&net.UnixAddr{Name: "/tmp/x", Net: "unix"}) {
		t.Error("non-IP address should be denied when ACL is active")
	}
}

func TestWrapInactiveReturnsSameListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if Wrap(ln, nil, "x", nil) != ln {
		t.Error("inactive ACL must not wrap the listener")
	}
	empty, _ := Parse(nil)
	if Wrap(ln, empty, "x", nil) != ln {
		t.Error("empty ACL must not wrap the listener")
	}
}

// Loopback is always allowed, so a connection from 127.0.0.1 passes even when
// the list names only other addresses; a denied path is exercised via deny().
func TestWrapAllowsLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	acl, _ := Parse([]string{"203.0.113.9"})
	wl := Wrap(ln, acl, "test", nil)

	go func() {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err == nil {
			c.Close()
		}
	}()
	conn, err := wl.Accept()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestDenyClosesAndThrottlesLog(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	acl, _ := Parse([]string{"203.0.113.9"})
	var logs atomic.Int32
	l := Wrap(ln, acl, "test", func(f string, a ...any) {
		logs.Add(1)
		if !strings.Contains(f, "denied") {
			t.Errorf("unexpected log format %q", f)
		}
	}).(*listener)

	for i := 0; i < 3; i++ {
		client, server := net.Pipe()
		l.deny(server)
		buf := make([]byte, 1)
		if _, err := client.Read(buf); err == nil {
			t.Error("denied connection should be closed")
		}
		client.Close()
	}
	if logs.Load() != 1 {
		t.Errorf("expected throttled single log, got %d", logs.Load())
	}
}
