package testutil_test

import (
	"context"
	"io"
	"net"
	"net/http/httptest"
	"opencode-infinity/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestFixtureRejectsNonLoopback(t *testing.T) {
	for _, target := range []string{"203.0.113.1:443", "example.invalid:443", "127.0.0.1:0", "127.0.0.1:65536"} {
		if proxy, err := testutil.NewSOCKS5("relay", "upstream.invalid:443", target, false); err == nil {
			proxy.Close()
			t.Fatalf("target tidak aman diterima: %s", target)
		}
	}
}

func TestFragmentGuards(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		sizes      []int
		canceled   bool
	}{
		{"large", strings.Repeat("x", testutil.MaxFrameBytes+1), []int{1}, false},
		{"zero", "x", []int{0}, false},
		{"negative", "x", []int{-1}, false},
		{"canceled", "x", []int{1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			w := httptest.NewRecorder()
			if err := testutil.WriteFragments(ctx, w, []byte(tc.data), tc.sizes); err == nil {
				t.Fatal("input tidak valid lolos")
			}
			if w.Body.Len() != 0 {
				t.Fatal("input tidak valid sempat ditulis")
			}
		})
	}
}

func TestMalformedSOCKS5(t *testing.T) {
	p, err := testutil.NewSOCKS5("relay", "upstream.invalid:1", "127.0.0.1:1", false)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for _, tc := range []struct {
		name              string
		greeting, request []byte
		accepted          bool
	}{
		{"auth", []byte{5, 1, 2}, nil, false},
		{"bad-version", []byte{4, 1, 0}, nil, false},
		{"truncated-greeting", []byte{5}, nil, false},
		{"bad-command", []byte{5, 1, 0}, []byte{5, 2, 0, 3, 1, 'x', 0, 1}, true},
		{"bad-atyp", []byte{5, 1, 0}, []byte{5, 1, 0, 1, 127, 0, 0, 1, 0, 1}, true},
		{"truncated-domain", []byte{5, 1, 0}, []byte{5, 1, 0, 3, 255, 'x'}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := net.DialTimeout("tcp4", p.Addr(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			c.SetDeadline(time.Now().Add(time.Second))
			if _, err := c.Write(tc.greeting); err != nil {
				t.Fatal(err)
			}
			if tc.accepted {
				var reply [2]byte
				if _, err := io.ReadFull(c, reply[:]); err != nil || reply != [2]byte{5, 0} {
					t.Fatalf("greeting=%v err=%v", reply, err)
				}
				if _, err := c.Write(tc.request); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.(*net.TCPConn).CloseWrite(); err != nil {
				t.Fatal(err)
			}
			payload, err := io.ReadAll(c)
			if err != nil {
				if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
					t.Fatalf("malformed input menggantung: %v", err)
				}
			}
			if tc.name == "auth" && string(payload) != string([]byte{5, 255}) {
				t.Fatalf("auth reply=%v", payload)
			}
		})
	}
	if len(p.Snapshot().Connects) != 0 {
		t.Fatal("malformed handshake membuat tunnel")
	}
}

func TestProxyConnectionBoundAndCleanup(t *testing.T) {
	p, err := testutil.NewSOCKS5("relay", "upstream.invalid:1", "127.0.0.1:1", false)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for range testutil.MaxConnections {
		c, err := net.DialTimeout("tcp4", p.Addr(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
	}
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for p.Snapshot().Active != testutil.MaxConnections {
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("active=%d", p.Snapshot().Active)
		}
	}
	extra, err := net.DialTimeout("tcp4", p.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer extra.Close()
	extra.SetReadDeadline(time.Now().Add(time.Second))
	var one [1]byte
	if n, err := extra.Read(one[:]); n != 0 || err == nil {
		t.Fatalf("koneksi berlebih tidak ditolak: n=%d err=%v", n, err)
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("koneksi berlebih dibiarkan menunggu")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if p.Snapshot().Active != 0 {
		t.Fatalf("koneksi tertinggal: %+v", p.Snapshot())
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
