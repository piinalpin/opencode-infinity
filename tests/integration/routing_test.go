package integration_test

import (
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"opencode-infinity/internal/testutil"
)

func TestRoutingThroughSOCKS5(t *testing.T) {
	var dispatch atomic.Int32
	peer := make(chan string, 1)
	up, authority, roots := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		dispatch.Add(1)
		peer <- r.RemoteAddr
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	a := newProxy(t, "relay-a", authority, up.Listener.Addr().String(), false)
	b := newProxy(t, "relay-b", authority, up.Listener.Addr().String(), false)
	client, log := controlClient(t, b.Addr(), authority, up.Listener.Addr().String(), roots, false)
	resp, err := post(t, client, authority, "/control")
	if err != nil {
		t.Fatalf("request melalui SOCKS5 gagal: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != `{"ok":true}` {
		t.Fatalf("body=%q err=%v", body, err)
	}
	trace := b.Snapshot()
	if a.Snapshot().Accepted != 0 || len(trace.Connects) != 1 || dispatch.Load() != 1 {
		t.Fatalf("jalur/dispatch salah: a=%+v b=%+v dispatch=%d", a.Snapshot(), trace, dispatch.Load())
	}
	c := trace.Connects[0]
	_, port, _ := net.SplitHostPort(authority)
	if c.Relay != "relay-b" || c.Hostname != hostname || c.Port != port || c.TargetPeer != <-peer {
		t.Fatalf("metadata CONNECT salah: %+v", c)
	}
	log.mu.Lock()
	direct, attempts := log.direct, log.proxy
	log.mu.Unlock()
	if direct != 0 || attempts != 1 {
		t.Fatalf("dial direct=%d proxy=%d", direct, attempts)
	}
}

func TestUnavailableProxyAndDirectOracle(t *testing.T) {
	peers := make(chan string, 1)
	up, authority, roots := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		peers <- r.RemoteAddr
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, "ok")
	})
	proxy := newProxy(t, "unavailable", authority, up.Listener.Addr().String(), false)
	if err := proxy.Close(); err != nil {
		t.Fatal(err)
	}
	client, log := controlClient(t, proxy.Addr(), authority, up.Listener.Addr().String(), roots, false)
	if resp, err := post(t, client, authority, "/control"); err == nil {
		resp.Body.Close()
		t.Fatal("proxy mati harus menggagalkan request")
	}
	select {
	case <-peers:
		t.Fatal("dispatch langsung saat proxy mati")
	default:
	}
	log.mu.Lock()
	direct, attempts := log.direct, log.proxy
	log.mu.Unlock()
	if direct != 0 || attempts != 1 {
		t.Fatalf("direct=%d proxy attempts=%d", direct, attempts)
	}
	// Kontrol positif: upstream yang sama memang dapat dijangkau langsung.
	broken, directLog := controlClient(t, proxy.Addr(), authority, up.Listener.Addr().String(), roots, true)
	resp, err := post(t, broken, authority, "/control")
	if err != nil {
		t.Fatalf("direct control tidak mencapai upstream: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	peer := <-peers
	if err := testutil.VerifyProxyPath([]testutil.Dispatch{{Peer: peer}}, proxy.Snapshot()); err == nil {
		t.Fatal("oracle meloloskan bypass direct yang mencapai upstream")
	}
	directLog.mu.Lock()
	d := directLog.direct
	directLog.mu.Unlock()
	if d != 1 {
		t.Fatalf("direct attempts=%d", d)
	}
}

func TestTLSHostnameMismatch(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) { t.Error("TLS salah hostname mencapai HTTP handler") })
	_, port, _ := net.SplitHostPort(u.Authority)
	wrong := net.JoinHostPort("wrong.invalid", port)
	p := newProxy(t, "relay", wrong, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), wrong, u.Addr(), u.Roots, false)
	resp, err := post(t, client, wrong, "/control")
	if err == nil {
		resp.Body.Close()
		t.Fatal("hostname TLS salah diterima")
	}
	var hostnameError x509.HostnameError
	if !errors.As(err, &hostnameError) {
		t.Fatalf("bukan kegagalan verifikasi hostname: %v", err)
	}
	if len(p.Snapshot().Connects) != 1 || u.Snapshot().Total != 0 || client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verifikasi dilemahkan/dispatch salah")
	}
}

func TestProxyRejectsOtherTarget(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) { t.Error("target di luar allowlist mencapai handler") })
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	_, port, _ := net.SplitHostPort(u.Authority)
	other := net.JoinHostPort("other.invalid", port)
	client, _ := controlClient(t, p.Addr(), other, u.Addr(), u.Roots, false)
	resp, err := post(t, client, other, "/control")
	if err == nil {
		resp.Body.Close()
		t.Fatal("target di luar allowlist diterima")
	}
	if u.Snapshot().Total != 0 || len(p.Snapshot().Connects) != 0 {
		t.Fatal("target lain didial")
	}
}

func TestUpstreamBodyAndRecordBounds(t *testing.T) {
	var calls atomic.Int32
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = io.WriteString(w, "ok") })
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	req, err := http.NewRequestWithContext(t.Context(), "POST", "https://"+u.Authority+"/large", io.NopCloser(strings.NewReader(strings.Repeat("x", testutil.MaxBodyBytes+1))))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge || calls.Load() != 0 {
		t.Fatal("body oversized lolos")
	}
	for range testutil.MaxRecords {
		resp, err = post(t, client, u.Authority, "/control")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	state := u.Snapshot()
	if state.Total != testutil.MaxRecords+1 || len(state.Dispatches) != testutil.MaxRecords || state.Rejected != 1 || state.Dispatches[0].BodyRead {
		t.Fatalf("record/body bound=%+v", state)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status saat ledger penuh=%d", resp.StatusCode)
	}
}
