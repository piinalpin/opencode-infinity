package integration_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"opencode-infinity/internal/testutil"
	"strings"
	"sync"
	"testing"
	"time"
)

// Consumer kontrol SSE berbuffer terbatas; bukan parser produksi.
func readEvent(r *bufio.Reader) (string, error) {
	var line, data []byte
	total := 0
	for {
		part, err := r.ReadSlice('\n')
		total += len(part)
		if total > testutil.MaxFrameBytes {
			return "", fmt.Errorf("frame SSE melewati batas")
		}
		line = append(line, part...)
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			if total > 0 {
				return "", io.ErrUnexpectedEOF
			}
			return "", err
		}
		s := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
		line = line[:0]
		if s == "" {
			return strings.TrimSuffix(string(data), "\n"), nil
		}
		if strings.HasPrefix(s, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(s, "data:"), " ")...)
			data = append(data, '\n')
		}
	}
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("observasi fixture melewati deadline")
		var zero T
		return zero
	}
}

const hostname = "upstream.invalid"

func testUpstream(t *testing.T, h http.HandlerFunc) (*httptest.Server, string, *x509.CertPool) {
	t.Helper()
	u := ledgerUpstream(t, h)
	return u.Server, u.Authority, u.Roots
}

type dialLog struct {
	mu            sync.Mutex
	proxy, direct int
}

func controlClient(t *testing.T, proxyAddr, authority, target string, roots *x509.CertPool, direct bool) (*http.Client, *dialLog) {
	t.Helper()
	log := &dialLog{}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		MaxConnsPerHost: 8, MaxIdleConns: 8, MaxIdleConnsPerHost: 8,
		TLSHandshakeTimeout: time.Second, ResponseHeaderTimeout: 2 * time.Second,
		IdleConnTimeout: time.Second, MaxResponseHeaderBytes: 8192,
	}
	if !direct {
		tr.Proxy = http.ProxyURL(&url.URL{Scheme: "socks5h", Host: proxyAddr})
	}
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		log.mu.Lock()
		switch address {
		case proxyAddr:
			log.proxy++
		case authority:
			log.direct++
			address = target
		default:
			log.mu.Unlock()
			return nil, &net.AddrError{Err: "di luar fixture", Addr: address}
		}
		log.mu.Unlock()
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, address)
	}
	t.Cleanup(tr.CloseIdleConnections)
	return &http.Client{Transport: tr, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, log
}

func post(t *testing.T, client *http.Client, authority, path string) (*http.Response, error) {
	t.Helper()
	// Body opaque: pasang GetBody=nil dan tanpa idempotency header.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://"+authority+path, io.NopCloser(strings.NewReader(`{"fixture":true}`)))
	if err != nil {
		t.Fatal(err)
	}
	return client.Do(req)
}

func newProxy(t *testing.T, relay, authority, target string, refuse bool) *testutil.SOCKS5 {
	t.Helper()
	s, err := testutil.NewSOCKS5(relay, authority, target, refuse)
	if err != nil {
		t.Fatalf("fixture SOCKS5: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}
