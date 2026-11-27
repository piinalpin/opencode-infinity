package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixedAdmissionCapacityCategory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edits []string
	}{
		{"relay", []string{"max_inflight: 5", "max_inflight: 1"}},
		{"outbound", []string{"outbound_connections: 32", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c, err := newFixtureCoordinator(modelConfig(t, tc.edits...), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			first, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			if err != nil {
				t.Fatal(err)
			}
			defer first.Body.Close()
			extra, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			if extra != nil {
				extra.Body.Close()
				t.Fatal("kapasitas penuh masih melakukan dispatch")
			}
			var f *Failure
			if !errors.As(err, &f) || f.Code != "gateway_capacity_exceeded" || f.Status != 429 {
				t.Fatalf("saturasi harus 429 gateway_capacity_exceeded, mendapat %v", err)
			}
			if calls != 1 {
				t.Fatal("request ditolak melakukan dispatch")
			}
			first.Body.Close()
			next, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			if err != nil {
				t.Fatal("slot gagal pulih", err)
			}
			next.Body.Close()
		})
	}
}

type fixedInputBody struct {
	io.Reader
	reads, closes atomic.Int32
	started       chan struct{}
	once          sync.Once
}

func (b *fixedInputBody) Read(p []byte) (int, error) {
	b.reads.Add(1)
	if b.started != nil {
		b.once.Do(func() { close(b.started) })
	}
	return b.Reader.Read(p)
}
func (b *fixedInputBody) Close() error { b.closes.Add(1); return nil }

func TestFixedAdmissionBodyBudget(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "known", true: "unknown"}[unknown], func(t *testing.T) {
			var calls atomic.Int32
			c, err := newFixtureCoordinator(modelConfig(t, "body_buffer_bytes: 67108864", "body_buffer_bytes: 300000"), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			first, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			if err != nil {
				t.Fatal(err)
			}
			defer first.Body.Close()
			// EOF belum berarti tail forwarding caller sudah selesai.
			if _, err := io.ReadAll(first.Body); err != nil {
				t.Fatal(err)
			}
			request := chatRequest(`{"model":"coding-default"}`)
			input := &fixedInputBody{Reader: strings.NewReader(`{"model":"coding-default"}`)}
			request.Body = input
			defer request.Body.Close()
			if unknown {
				request.ContentLength = -1
			}
			response, err := c.Chat(request, "")
			if response != nil {
				response.Body.Close()
				t.Fatal("budget agregat dilampaui")
			}
			assertFailure(t, err, "gateway_capacity_exceeded")
			if input.reads.Load() != 0 || calls.Load() != 1 {
				t.Fatal("budget habis masih membaca body atau melakukan dispatch")
			}
			first.Body.Close()
			next, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			if err != nil {
				t.Fatal("anggaran body gagal pulih", err)
			}
			next.Body.Close()
		})
	}
}

func TestFixedAdmissionPendingWorkAndCatalogue(t *testing.T) {
	var calls atomic.Int32
	c, err := newFixtureCoordinator(modelConfig(t, "max_active_requests: 20", "max_active_requests: 1"), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	input := &fixedInputBody{Reader: reader, started: make(chan struct{})}
	request := chatRequest(`{"model":"coding-default"}`)
	request.Body = input
	request.ContentLength = -1
	done := make(chan error, 1)
	go func() {
		response, err := c.Chat(request, "")
		if response != nil {
			response.Body.Close()
		}
		done <- err
	}()
	select {
	case <-input.started:
	case <-time.After(3 * time.Second):
		t.Fatal("persiapan tidak mulai membaca body")
	}
	for range 10 {
		extra := &fixedInputBody{Reader: strings.NewReader(`{"model":"coding-default"}`)}
		r := chatRequest(`{"model":"coding-default"}`)
		r.Body = extra
		response, err := c.Chat(r, "")
		if response != nil {
			response.Body.Close()
			t.Fatal("pending body tidak dihitung pada batas global")
		}
		assertFailure(t, err, "gateway_capacity_exceeded")
		if extra.reads.Load() != 0 {
			t.Fatal("pekerjaan pending tidak dibatasi sebelum read")
		}
	}
	if catalogue, err := c.Models(""); catalogue != nil {
		catalogue.Body.Close()
		t.Fatal("katalog melewati batas global bersama")
	} else {
		assertFailure(t, err, "gateway_capacity_exceeded")
	}
	if calls.Load() != 0 {
		t.Fatal("pending atau request ditolak melakukan dispatch")
	}
	writer.CloseWithError(errors.New("synthetic-input-failure"))
	select {
	case err := <-done:
		assertFailure(t, err, "invalid_request")
	case <-time.After(3 * time.Second):
		t.Fatal("pending tidak selesai setelah reader gagal")
	}
	if input.closes.Load() != 1 {
		t.Fatal("body pending gagal tidak ditutup tepat sekali")
	}
	catalogue, err := c.Models("")
	if err != nil {
		t.Fatal("kapasitas katalog tidak pulih", err)
	}
	defer catalogue.Body.Close()
	if response, err := c.Chat(chatRequest(`{"model":"coding-default"}`), ""); response != nil {
		response.Body.Close()
		t.Fatal("generasi tidak berbagi batas dengan katalog")
	} else {
		assertFailure(t, err, "gateway_capacity_exceeded")
	}
	catalogue.Body.Close()
	response, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if err != nil {
		t.Fatal("kapasitas generasi tidak pulih", err)
	}
	response.Body.Close()
}

func TestFixedAdmissionReleaseLifecycle(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, readErr := range []error{io.EOF, errors.New("synthetic-read-error")} {
			name := map[bool]string{false: "json", true: "sse"}[stream] + "/" + readErr.Error()
			t.Run(name, func(t *testing.T) {
				var calls atomic.Int32
				body := &countedBody{Reader: fixedTailReader{err: readErr}}
				c, err := newFixtureCoordinator(modelConfig(t, "max_active_requests: 20", "max_active_requests: 1"), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
					if calls.Add(1) == 1 {
						header := http.Header{"Content-Type": {"application/json"}}
						if stream {
							header.Set("Content-Type", "text/event-stream")
						}
						return &http.Response{StatusCode: 200, Header: header, Body: body}, nil
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
				}))
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				input := `{"model":"coding-default","stream":` + map[bool]string{false: "false", true: "true"}[stream] + `}`
				response, err := c.Chat(chatRequest(input).WithContext(ctx), "")
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				buffer := make([]byte, 32)
				if n, err := response.Body.Read(buffer); n != 4 || !errors.Is(err, readErr) || string(buffer[:n]) != "tail" {
					t.Fatal("read mengubah tail atau kategori error")
				}
				if extra, err := c.Chat(chatRequest(input), ""); extra != nil {
					extra.Body.Close()
					t.Fatal("lease dilepas sebelum tail selesai ditulis caller")
				} else {
					assertFailure(t, err, "gateway_capacity_exceeded")
				}
				var race sync.WaitGroup
				race.Go(cancel)
				for range 20 {
					race.Go(func() { response.Body.Close() })
				}
				race.Wait()
				if body.closes.Load() != 1 {
					t.Fatal("jaringan ditutup lebih dari sekali")
				}
				next, err := c.Chat(chatRequest(input), "")
				if err != nil || calls.Load() != 2 {
					t.Fatal("kapasitas tidak pulih atau dispatch berulang", err)
				}
				next.Body.Close()
			})
		}
	}
}

type fixedTailReader struct{ err error }

func (r fixedTailReader) Read(p []byte) (int, error) { return copy(p, "tail"), r.err }

func TestLeastInflightCoordinatorSelectedRelay(t *testing.T) {
	cfg := modelConfig(t, "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 5\nupstreams:", "relays: [warp-1]", "relays: [warp-2, warp-1]", "routing: fixed", "routing: least_inflight\n        stateless: true")
	var visits []string
	c, err := newFixtureCoordinator(cfg, dispatchFunc(func(r *http.Request, relay config.Relay) (*http.Response, error) {
		visits = append(visits, relay.ID)
		if r.URL.Host != "api.provider.example" || r.URL.Path != "/v1/chat/completions" {
			t.Error("pilihan relay mengganti target upstream")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), "provider-model-id") {
			t.Error("mapping model berubah")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	one, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	defer one.Body.Close()
	two, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	defer two.Body.Close()
	if len(visits) != 2 || visits[0] != "warp-2" || visits[1] != "warp-1" {
		t.Fatalf("dispatch tidak mengikuti relay reservasi: %v", visits)
	}
}

func TestLeastInflightSingleDispatchOnFailure(t *testing.T) {
	for _, kind := range []string{"transport", "redirect", "upstream429"} {
		t.Run(kind, func(t *testing.T) {
			cfg := modelConfig(t, "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 5\nupstreams:", "relays: [warp-1]", "relays: [warp-2, warp-1]", "routing: fixed", "routing: least_inflight\n        stateless: true")
			calls := 0
			c, err := newFixtureCoordinator(cfg, dispatchFunc(func(r *http.Request, relay config.Relay) (*http.Response, error) {
				calls++
				if relay.ID != "warp-2" {
					t.Error("pilihan awal tidak stabil")
				}
				if kind == "transport" {
					return nil, errors.New("fixture-transport-failure")
				}
				code := 429
				if kind == "redirect" {
					code = 302
				}
				return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "Retry-After": {"60"}, "Location": {"https://other.invalid"}}, Body: io.NopCloser(strings.NewReader(`{"error":"fixture"}`))}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			response, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
			switch kind {
			case "transport":
				assertFailure(t, err, "upstream_transport_error")
			case "redirect":
				assertFailure(t, err, "upstream_redirect_rejected")
			default:
				if err != nil || response == nil || response.StatusCode != 429 || response.Header.Get("Retry-After") != "60" {
					t.Fatal("upstream429 berubah")
				}
				response.Body.Close()
			}
			if calls != 1 {
				t.Fatal("least menjalankan retry/fallback otomatis")
			}
			for _, r := range c.admission.View().Relays {
				if !r.CooldownUntil.IsZero() {
					t.Fatal("upstream429 otomatis diubah menjadi relay cooldown")
				}
			}
		})
	}
}

func TestUpstream429CooldownBlocksSameScopeWithoutRetry(t *testing.T) {
	calls := 0
	c, err := newFixtureCoordinator(modelConfig(t), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": {"120"}}, Body: io.NopCloser(strings.NewReader("limited"))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := func() *http.Request {
		return httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
	}
	response, err := c.Chat(request(), "")
	if err != nil || response == nil || response.StatusCode != 429 {
		t.Fatalf("upstream response tidak diteruskan: %v", err)
	}
	response.Body.Close()
	if _, err := c.Chat(request(), ""); err == nil || err.Error() != "no_eligible_relay" {
		t.Fatalf("429 Retry-After tidak mencegah assignment scope sama: %v", err)
	}
	if calls != 1 {
		t.Fatalf("dispatch otomatis terjadi: %d", calls)
	}
}
