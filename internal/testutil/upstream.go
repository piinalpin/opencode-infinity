package testutil

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

const Hostname = "upstream.invalid"
const MaxBodyBytes = 64 * 1024

type Dispatch struct {
	Peer                         string
	BodyBytes                    int
	BodyRead, Returned, Canceled bool
}
type UpstreamSnapshot struct {
	Dispatches                         []Dispatch
	Total, Rejected, ActiveConnections int
}

// Upstream merekam metadata saja, bukan body/header. Handler harus menghormati Context.
// shortcut: fixture HTTP/1.1; validasi HTTP/2 dilakukan saat task integrasinya memerlukan.
type Upstream struct {
	Server          *httptest.Server
	Authority       string
	Roots           *x509.CertPool
	listener        *boundedListener
	mu              sync.Mutex
	dispatches      []Dispatch
	total, rejected int
	closeOnce       sync.Once
	done            chan struct{}
}

func NewUpstream(h http.HandlerFunc) (*Upstream, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: Hostname}, DNSNames: []string{Hostname}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	bl := &boundedListener{Listener: l, live: make(map[*trackedConn]struct{})}
	u := &Upstream{listener: bl, Roots: x509.NewCertPool(), done: make(chan struct{})}
	u.Roots.AddCert(cert)
	_, port, _ := net.SplitHostPort(l.Addr().String())
	u.Authority = net.JoinHostPort(Hostname, port)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.total++
		if len(u.dispatches) >= MaxRecords {
			u.rejected++
			u.mu.Unlock()
			w.Header().Set("Connection", "close")
			http.Error(w, "fixture record limit", http.StatusServiceUnavailable)
			return
		}
		index := len(u.dispatches)
		u.dispatches = append(u.dispatches, Dispatch{Peer: r.RemoteAddr})
		u.mu.Unlock()
		defer func() {
			u.mu.Lock()
			u.dispatches[index].Returned = true
			// Diperiksa sebelum wrapper ServeHTTP kembali; server juga membatalkan
			// Context setelah return normal, yang bukan pembatalan saat aktif.
			u.dispatches[index].Canceled = r.Context().Err() != nil
			u.mu.Unlock()
		}()
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
		u.mu.Lock()
		u.dispatches[index].BodyBytes = len(body)
		u.dispatches[index].BodyRead = err == nil
		u.mu.Unlock()
		if err != nil {
			http.Error(w, "fixture body limit/read error", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		h(w, r)
	})
	u.Server = &httptest.Server{Listener: bl, Config: &http.Server{Handler: handler, ReadHeaderTimeout: time.Second, ReadTimeout: FixtureTimeout, WriteTimeout: FixtureTimeout, IdleTimeout: time.Second, MaxHeaderBytes: 8192}, TLS: &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}}}
	u.Server.StartTLS()
	return u, nil
}
func (u *Upstream) Addr() string { return u.Server.Listener.Addr().String() }
func (u *Upstream) Snapshot() UpstreamSnapshot {
	u.mu.Lock()
	defer u.mu.Unlock()
	return UpstreamSnapshot{Dispatches: append([]Dispatch(nil), u.dispatches...), Total: u.total, Rejected: u.rejected, ActiveConnections: u.listener.active()}
}
func (u *Upstream) Close() error {
	u.closeOnce.Do(func() {
		u.listener.closeAll()
		go func() { u.Server.Close(); close(u.done) }()
	})
	select {
	case <-u.done:
		return nil
	case <-time.After(FixtureTimeout):
		return fmt.Errorf("cleanup upstream melewati deadline")
	}
}

// VerifyProxyPath menghubungkan peer TCP HTTP dengan socket keluar relay.
// Nol dispatch bukan bukti belum terkirim; fungsi ini bukan izin retry.
func VerifyProxyPath(dispatches []Dispatch, proxy ProxySnapshot) error {
	for _, d := range dispatches {
		matched := false
		for _, c := range proxy.Connects {
			if !c.Refused && c.TargetPeer != "" && c.TargetPeer == d.Peer {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("dispatch tidak memiliki koneksi relay yang sesuai")
		}
	}
	return nil
}

// DropConnection memutus HTTP/1.1 tanpa menulis response header/terminal chunk.
func DropConnection(w http.ResponseWriter) error {
	c, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return err
	}
	return c.Close()
}

type boundedListener struct {
	net.Listener
	mu       sync.Mutex
	closed   bool
	accepted int
	live     map[*trackedConn]struct{}
}
type trackedConn struct {
	net.Conn
	owner *boundedListener
	once  sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.owner.mu.Lock(); delete(c.owner.live, c); c.owner.mu.Unlock() })
	return err
}
func (l *boundedListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		l.mu.Lock()
		if l.closed || len(l.live) >= MaxConnections || l.accepted >= MaxRecords {
			l.mu.Unlock()
			c.Close()
			continue
		}
		l.accepted++
		tracked := &trackedConn{Conn: c, owner: l}
		l.live[tracked] = struct{}{}
		l.mu.Unlock()
		return tracked, nil
	}
}
func (l *boundedListener) active() int { l.mu.Lock(); defer l.mu.Unlock(); return len(l.live) }
func (l *boundedListener) closeAll() {
	l.mu.Lock()
	l.closed = true
	_ = l.Listener.Close()
	live := make([]*trackedConn, 0, len(l.live))
	for c := range l.live {
		live = append(live, c)
	}
	l.mu.Unlock()
	for _, c := range live {
		_ = c.Close()
	}
}
