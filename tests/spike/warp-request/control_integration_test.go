package warp_request_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/testutil"
)

func TestWarpControlBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generation.used")
	var admitted atomic.Int32
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			if (postBudget{path: path}).reserve() == nil {
				admitted.Add(1)
			}
		})
	}
	workers.Wait()
	if admitted.Load() != 1 {
		t.Fatalf("generation budget admitted=%d, want 1", admitted.Load())
	}
	if err := (postBudget{path: path}).reserve(); err == nil {
		t.Fatal("budget persisten direset oleh instance baru")
	}
}

const deltaFrame = "data: {\"choices\":[{\"delta\":{\"content\":\"Halo 🌏\"}}]}\n\n"
const terminalFrames = "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

func TestWarpControlIncrementalStream(t *testing.T) {
	next := make(chan struct{})
	_, _, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if err := testutil.WriteFragments(r.Context(), w, []byte(deltaFrame), []int{1, 3}); err != nil {
			return
		}
		select {
		case <-next:
		case <-r.Context().Done():
			return
		}
		_ = testutil.WriteFragments(r.Context(), w, []byte(terminalFrames), []int{2, 1})
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", s.URL+generationPath, strings.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-T04-Run", "fixture-nonce")
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatalf("first stream event buffered until completion: %v", err)
	}
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)
	first, err := r.ReadString('\n')
	if err != nil || !strings.HasPrefix(first, "data:") {
		t.Fatal("event pertama tidak sampai")
	}
	close(next)
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	if c.trace.snapshot().Failure != "" {
		t.Fatalf("stream fixture gagal: %+v", c.trace.snapshot())
	}
}

func TestWarpControlRejectsInvalidStreams(t *testing.T) {
	for _, tc := range []struct{ name, payload string }{
		{"empty", ""}, {"no-terminal", deltaFrame}, {"terminal-only", "data: [DONE]\n\n"},
		{"invalid-json", "data: not-json\n\n"}, {"error", `data: {"error":{"message":"PRIVATE-CREDENTIAL-PROMPT"}}` + "\n\n"},
		{"large-frame", "data: " + strings.Repeat("x", 65537) + "\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, tc.payload)
			})
			req, _ := http.NewRequestWithContext(t.Context(), "POST", s.URL+generationPath, strings.NewReader(requestBody))
			req.Header.Set("X-T04-Run", "fixture-nonce")
			resp, err := s.Client().Do(req)
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			if c.trace.snapshot().Failure == "" {
				t.Fatal("invalid stream classified as success")
			}
		})
	}
}

func setupControl(t *testing.T, h http.HandlerFunc) (*testutil.Upstream, *testutil.SOCKS5, *control, *httptest.Server) {
	t.Helper()
	u, err := testutil.NewUpstream(h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := u.Close(); err != nil {
			t.Error(err)
		}
	})
	p, err := testutil.NewSOCKS5("t04-relay", u.Authority, u.Addr(), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	c, err := newControl(p.Addr(), "https://"+u.Authority+generationPath, "fixture-nonce", filepath.Join(t.TempDir(), "generation.used"), u.Roots)
	if err != nil {
		t.Fatalf("control fixture: %v", err)
	}
	s := httptest.NewServer(c)
	t.Cleanup(s.Close)
	return u, p, c, s
}
func generation(t *testing.T, s *httptest.Server, nonce, body string) *http.Response {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), "POST", s.URL+generationPath, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("X-T04-Run", nonce)
	resp, err := s.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

const requestBody = `{"model":"deepseek-v4.1-flash-free","stream":true,"max_tokens":512,"messages":[{"role":"user","content":"fixture"}]}`

func TestWarpControlRoute(t *testing.T) {
	u, p, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != generationPath || r.Header.Get("X-T04-Run") != "" || r.Header.Get("x-opencode-client") != "desktop" {
			t.Error("path/header target salah")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, deltaFrame+terminalFrames)
	})
	for _, tc := range []struct {
		nonce, body string
		status      int
	}{
		{"wrong", requestBody, 401},
		{"fixture-nonce", strings.Replace(requestBody, selectedModel, "other-model", 1), 400},
		{"fixture-nonce", strings.Replace(requestBody, `"stream":true`, `"stream":false`, 1), 400},
		{"fixture-nonce", strings.Replace(requestBody, `512`, `513`, 1), 400},
		{"fixture-nonce", requestBody, 200},
		{"fixture-nonce", requestBody, 409},
	} {
		resp := generation(t, s, tc.nonce, tc.body)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("status=%d want%d", resp.StatusCode, tc.status)
		}
	}
	state := c.trace.snapshot()
	if state.Admitted != 1 || state.Dials != 1 || state.Connects != 1 || state.LocalWrites != 1 || !state.TLSVerified || state.SOCKSHost != testutil.Hostname || u.Snapshot().Total != 1 || len(p.Snapshot().Connects) != 1 {
		t.Fatalf("metadata jalur=%+v upstream=%+v", state, u.Snapshot())
	}
}

func TestWarpControlFailuresDoNotReplay(t *testing.T) {
	for _, mode := range []string{"dead-relay", "ambiguous", "header-timeout", "stream-started", "status403", "status429", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			dropped := make(chan struct{}, 1)
			u, p, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "ambiguous":
					_ = testutil.DropConnection(w)
				case "header-timeout":
					<-r.Context().Done()
				case "stream-started":
					w.Header().Set("Content-Type", "text/event-stream")
					if err := testutil.WriteFragments(r.Context(), w, []byte(deltaFrame), []int{1}); err == nil {
						_ = testutil.DropConnection(w)
					}
				case "status403":
					http.Error(w, "PRIVATE-SECRET-RESPONSE", 403)
				case "status429":
					http.Error(w, "PRIVATE-SECRET-RESPONSE", 429)
				case "redirect":
					w.Header().Set("Location", "https://other.invalid/private")
					w.WriteHeader(307)
				default:
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = io.WriteString(w, deltaFrame+terminalFrames)
				}
				dropped <- struct{}{}
			})
			if mode == "dead-relay" {
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "header-timeout" {
				c.client.Transport.(*http.Transport).ResponseHeaderTimeout = 50 * time.Millisecond
			}
			resp := generation(t, s, "fixture-nonce", requestBody)
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			again := generation(t, s, "fixture-nonce", requestBody)
			_, _ = io.Copy(io.Discard, again.Body)
			again.Body.Close()
			if again.StatusCode != 409 {
				t.Fatal("retry diteruskan")
			}
			want := 1
			if mode == "dead-relay" {
				want = 0
			} else {
				select {
				case <-dropped:
				case <-time.After(2 * time.Second):
					t.Fatal("handler tidak selesai")
				}
			}
			if u.Snapshot().Total != want || c.trace.snapshot().Admitted != 1 || c.trace.snapshot().Dials != 1 || c.trace.snapshot().Failure == "" {
				t.Fatalf("fault metadata=%+v dispatch=%d", c.trace.snapshot(), u.Snapshot().Total)
			}
			raw, _ := json.Marshal(c.trace.snapshot())
			if strings.Contains(string(raw), "PRIVATE-") {
				t.Fatal("payload bocor dalam metadata")
			}
		})
	}
}

func TestWarpControlTLSAndConfigGuards(t *testing.T) {
	for _, target := range []string{"http://opencode.ai" + generationPath, "https://user:PRIVATE@opencode.ai" + generationPath, "https://opencode.ai" + generationPath + "?target=evil", "https://opencode.ai/other", "https://opencode.ai" + generationPath + "#secret"} {
		if _, err := newControl("127.0.0.1:1", target, "nonce", filepath.Join(t.TempDir(), "used"), nil); err == nil {
			t.Fatal("target invalid diterima")
		}
	}
	u, err := testutil.NewUpstream(func(w http.ResponseWriter, r *http.Request) { t.Error("TLS salah mencapai handler") })
	if err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	_, port, _ := net.SplitHostPort(u.Authority)
	wrong := net.JoinHostPort("wrong.invalid", port)
	p, err := testutil.NewSOCKS5("wrong", wrong, u.Addr(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	c, err := newControl(p.Addr(), "https://"+wrong+generationPath, "fixture-nonce", filepath.Join(t.TempDir(), "used"), u.Roots)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	s := httptest.NewServer(c)
	defer s.Close()
	resp := generation(t, s, "fixture-nonce", requestBody)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 502 || c.trace.snapshot().TLSVerified || u.Snapshot().Total != 0 {
		t.Fatal("TLS hostname tidak diverifikasi")
	}
}

func TestSOCKSHandshakeBoundaries(t *testing.T) {
	for _, reply := range [][]byte{{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}, {5, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, {5, 0, 0, 3, 1, 'x', 0, 0}, {5, 2, 0, 1, 0, 0, 0, 0, 0, 0}, {5, 0, 0, 9}, {5, 0, 0, 1, 0}} {
		l, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			conn, err := l.Accept()
			if err != nil {
				done <- err
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(time.Second))
			var greet [3]byte
			if _, err := io.ReadFull(conn, greet[:]); err != nil {
				done <- err
				return
			}
			if err := writeAll(conn, []byte{5, 0}); err != nil {
				done <- err
				return
			}
			var head [5]byte
			if _, err := io.ReadFull(conn, head[:]); err != nil {
				done <- err
				return
			}
			tail := make([]byte, int(head[4])+2)
			if _, err := io.ReadFull(conn, tail); err != nil {
				done <- err
				return
			}
			for _, b := range reply {
				if err := writeAll(conn, []byte{b}); err != nil {
					done <- err
					return
				}
			}
			// Handshake consumer tidak boleh menelan data tunnel berikutnya.
			if reply[1] == 0 && (reply[3] == 1 && len(reply) == 10 || reply[3] == 4 || reply[3] == 3) {
				_ = writeAll(conn, []byte("TLS_SENTINEL"))
			}
			done <- nil
		}()
		trace := &recorder{}
		conn, err := dialSOCKS(t.Context(), l.Addr().String(), "upstream.invalid", "443", trace)
		valid := reply[1] == 0 && (reply[3] == 1 && len(reply) == 10 || reply[3] == 4 || reply[3] == 3)
		if valid {
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(conn)
			conn.Close()
			if err != nil || string(b) != "TLS_SENTINEL" {
				t.Fatal("boundary tunnel rusak")
			}
			if trace.snapshot().Connects != 1 || trace.snapshot().SOCKSHost != "upstream.invalid" {
				t.Fatal("CONNECT metadata salah")
			}
		} else if err == nil {
			conn.Close()
			t.Fatal("reply rusak diterima")
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handshake worker tertinggal")
		}
		l.Close()
	}
}

func TestWarpControlCancellationAndConcurrentBudget(t *testing.T) {
	canceled := make(chan struct{}, 1)
	u, _, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if err := testutil.WriteFragments(r.Context(), w, []byte(deltaFrame), []int{1}); err != nil {
			return
		}
		<-r.Context().Done()
		canceled <- struct{}{}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", s.URL+generationPath, strings.NewReader(requestBody))
	req.Header.Set("X-T04-Run", "fixture-nonce")
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			r := generation(t, s, "fixture-nonce", requestBody)
			body, err := io.ReadAll(io.LimitReader(r.Body, 128))
			r.Body.Close()
			// Satu stream aktif + delapan caller dapat memenuhi delapan slot.
			// Kedua penolakan sah harus mempunyai kategori tepat dan tidak menambah dispatch.
			budgetDenied := r.StatusCode == 409 && string(body) == "generation_budget_used\n"
			capacityDenied := r.StatusCode == 503 && string(body) == "control_capacity\n"
			if err != nil || (!budgetDenied && !capacityDenied) {
				t.Errorf("parallel duplicate tidak ditolak sesuai kontrak: status=%d", r.StatusCode)
			}
		})
	}
	workers.Wait()
	cancel()
	resp.Body.Close()
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream tidak dibatalkan")
	}
	if u.Snapshot().Total != 1 || c.trace.snapshot().Admitted != 1 || c.trace.snapshot().Denied != 8 {
		t.Fatal("counter paralel salah")
	}
}
