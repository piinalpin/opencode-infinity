package integration_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/testutil"
)

func gatewayStreamRequest(t *testing.T, server *httptest.Server, path, body string) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), "POST", server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+modelGatewayToken)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "must-not-replay")
	return r
}

func TestGatewayStreamFirstEvent(t *testing.T) {
	for _, path := range []string{"/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
		t.Run(path, func(t *testing.T) {
			next := make(chan struct{})
			finished := make(chan struct{})
			var bad atomic.Bool
			handler := func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				model, key := "provider-model-id", modelPrimaryKey
				if strings.HasPrefix(r.URL.Path, "/v2/") {
					model, key = "secondary-model-id", modelSecondaryKey
				}
				if string(body) != `{"model":"`+model+`","stream":true,"extra":9007199254740993}` || r.Header.Get("Authorization") != "Bearer "+key || r.Header.Get("Idempotency-Key") != "" {
					bad.Store(true)
				}
				w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
				for i, event := range transcript {
					if i == 1 {
						select {
						case <-next:
						case <-r.Context().Done():
							return
						}
					}
					if testutil.WriteFragments(r.Context(), w, []byte("data: "+event+"\n\n"), []int{1, 3, 2, 7}) != nil {
						return
					}
				}
				close(finished)
			}
			h := newModelHarness(t, handler, handler)
			server := httptest.NewServer(h.handler)
			defer server.Close()
			client := server.Client()
			client.Timeout = 1500 * time.Millisecond
			response, err := client.Do(gatewayStreamRequest(t, server, path, `{"model":"coding-default","stream":true,"extra":9007199254740993}`))
			if err != nil {
				t.Fatal("event pertama tertahan sampai event berikutnya/EOF")
			}
			defer response.Body.Close()
			reader := bufio.NewReader(response.Body)
			for i, want := range transcript {
				event, err := readEvent(reader)
				if err != nil || event != want {
					t.Fatal("urutan/byte SSE berubah")
				}
				if i == 0 {
					select {
					case <-finished:
						t.Fatal("stream selesai sebelum gate dibuka")
					default:
					}
					close(next)
				}
			}
			if _, err := reader.ReadByte(); err != io.EOF {
				t.Fatal("stream sukses tidak berakhir normal")
			}
			await(t, finished)
			if bad.Load() || h.primary.Snapshot().Total+h.secondary.Snapshot().Total != 1 {
				t.Fatal("target/key/model/dispatch stream salah")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayStreamOpaqueLargeEvent(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean-eof", true: "upstream-done"}[terminal], func(t *testing.T) {
			wire := ": comment\r\nevent: chunk\r\ndata: {\"choices\":[{\"delta\":{\"content\":\"" + strings.Repeat("x", 96*1024) + "🌏\"}}],\"extra\":{\"n\":9007199254740993,\"big\":1e1000}}\r\n\r\ndata: {\"usage\":null}\r\n\r\n"
			if terminal {
				wire += "data: [DONE]\n\n"
			}
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				for offset := 0; offset < len(wire); offset += 8192 {
					end := min(offset+8192, len(wire))
					if testutil.WriteFragments(r.Context(), w, []byte(wire[offset:end]), []int{127, 4096, 13}) != nil {
						return
					}
				}
			}, func(http.ResponseWriter, *http.Request) { t.Error("large event mengganti upstream") })
			server := httptest.NewServer(h.handler)
			defer server.Close()
			server.Client().Timeout = 3 * time.Second
			// Request false yang mendapat SSE juga wajib melalui accounting dan forwarding stream.
			response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":false}`))
			if err != nil {
				t.Fatal("event besar ditolak")
			}
			defer response.Body.Close()
			data, err := io.ReadAll(io.LimitReader(response.Body, 256*1024))
			if err != nil || !bytes.Equal(data, []byte(wire)) {
				t.Fatal("event besar/usage/extension/line ending berubah")
			}
			if h.primary.Snapshot().Total != 1 {
				t.Fatal("event besar didispatch ulang")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayStreamDrop(t *testing.T) {
	for _, fragment := range []bool{false, true} {
		t.Run(map[bool]string{false: "after-event", true: "partial-event"}[fragment], func(t *testing.T) {
			drop := make(chan struct{})
			dropError := make(chan error, 1)
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				if testutil.WriteFragments(r.Context(), w, []byte("data: ready\n\n"), []int{1}) != nil {
					return
				}
				select {
				case <-drop:
				case <-r.Context().Done():
					return
				}
				if fragment {
					if testutil.WriteFragments(r.Context(), w, []byte(`data: {"choices":[`), []int{2, 1}) != nil {
						return
					}
				}
				dropError <- testutil.DropConnection(w)
			}, func(http.ResponseWriter, *http.Request) { t.Error("stream putus mengganti upstream") })
			server := httptest.NewServer(h.handler)
			defer server.Close()
			server.Client().Timeout = 3 * time.Second
			response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`))
			if err != nil {
				t.Fatal("stream tidak dimulai")
			}
			defer response.Body.Close()
			reader := bufio.NewReader(response.Body)
			first, err := readEvent(reader)
			if err != nil || first != "ready" {
				t.Fatal("event pertama tidak diterima sebelum fault")
			}
			close(drop)
			if err := await(t, dropError); err != nil {
				t.Fatal("fixture fault gagal")
			}
			remaining, err := io.ReadAll(reader)
			if err == nil || bytes.Contains(remaining, []byte("[DONE]")) || bytes.Contains(remaining, []byte(`"error"`)) {
				t.Fatal("stream putus diberi EOF/terminal/envelope palsu")
			}
			if h.primary.Snapshot().Total != 1 || h.secondary.Snapshot().Total != 0 || len(h.a.Snapshot().Connects) != 1 {
				t.Fatal("stream putus diulang")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayStreamCancellation(t *testing.T) {
	for _, mode := range []string{"context", "body-close"} {
		t.Run(mode, func(t *testing.T) {
			var waiting atomic.Bool
			waiting.Store(true)
			cancelled := make(chan time.Time, 1)
			release := make(chan struct{})
			var releaseOnce atomic.Bool
			defer func() {
				if releaseOnce.CompareAndSwap(false, true) {
					close(release)
				}
			}()
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if !waiting.Load() {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				if testutil.WriteFragments(r.Context(), w, []byte("data: ready\n\n"), []int{1}) != nil {
					return
				}
				<-r.Context().Done()
				cancelled <- time.Now()
				select {
				case <-release:
				case <-time.After(time.Second):
				}
			}, func(http.ResponseWriter, *http.Request) { t.Error("cancel mengganti upstream") }, "max_active_requests: 20", "max_active_requests: 1")
			returned := make(chan struct{}, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() { returned <- struct{}{} }()
				h.handler.ServeHTTP(w, r)
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`).WithContext(ctx))
			if err != nil {
				t.Fatal("stream cancel tidak dimulai")
			}
			defer response.Body.Close()
			if first, err := readEvent(bufio.NewReader(response.Body)); err != nil || first != "ready" {
				t.Fatal("event sebelum cancel hilang")
			}
			at := time.Now()
			if mode == "context" {
				cancel()
			} else {
				response.Body.Close()
			}
			observed := await(t, cancelled)
			if observed.Before(at) || observed.Sub(at) > 2*time.Second {
				t.Fatal("cancel upstream melewati 2 detik")
			}
			if h.primary.Snapshot().Dispatches[0].Returned {
				t.Fatal("cancel hanya terlihat saat cleanup")
			}
			await(t, returned)
			waiting.Store(false)
			if releaseOnce.CompareAndSwap(false, true) {
				close(release)
			}
			if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 200 {
				t.Fatal("slot cancel tidak dilepas")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayStreamTimeouts(t *testing.T) {
	for _, phase := range []string{"header", "idle", "total"} {
		t.Run(phase, func(t *testing.T) {
			var waiting atomic.Bool
			waiting.Store(true)
			cancelled := make(chan time.Time, 1)
			edits := []string{"max_active_requests: 20", "max_active_requests: 1", "response_headers_seconds: 120", "response_headers_seconds: 2", "stream_idle_seconds: 120", "stream_idle_seconds: 2", "total_request_seconds: 1800", "total_request_seconds: 4"}
			switch phase {
			case "header":
				edits[3] = "response_headers_seconds: 1"
			case "idle":
				edits[5] = "stream_idle_seconds: 1"
			case "total":
				edits[7] = "total_request_seconds: 1"
			}
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if !waiting.Load() {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				if phase != "header" {
					w.Header().Set("Content-Type", "text/event-stream")
					if testutil.WriteFragments(r.Context(), w, []byte("data: ready\n\n"), []int{1, 3}) != nil {
						return
					}
				}
				if phase == "total" {
					ticker := time.NewTicker(50 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-r.Context().Done():
							cancelled <- time.Now()
							return
						case <-ticker.C:
							if testutil.WriteFragments(r.Context(), w, []byte(": progress\n\n"), []int{12}) != nil {
								cancelled <- time.Now()
								return
							}
						}
					}
				}
				<-r.Context().Done()
				cancelled <- time.Now()
			}, func(http.ResponseWriter, *http.Request) { t.Error("timeout mengganti upstream") }, edits...)
			returned := make(chan struct{}, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() { returned <- struct{}{} }()
				h.handler.ServeHTTP(w, r)
			}))
			defer server.Close()
			server.Client().Timeout = 3 * time.Second
			at := time.Now()
			response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`))
			if err != nil {
				t.Fatal("timeout fixture gagal sebelum respons gateway")
			}
			defer response.Body.Close()
			if phase == "header" {
				data, err := io.ReadAll(response.Body)
				if err != nil || response.StatusCode != 504 || !bytes.Contains(data, []byte(`"code":"upstream_timeout"`)) {
					t.Fatal("header timeout tidak dibedakan sebelum stream")
				}
			} else {
				reader := bufio.NewReader(response.Body)
				first, err := readEvent(reader)
				if err != nil || first != "ready" {
					t.Fatal("event sebelum timeout hilang")
				}
				data, err := io.ReadAll(reader)
				if err == nil || bytes.Contains(data, []byte("[DONE]")) || bytes.Contains(data, []byte(`"error"`)) {
					t.Fatal("timeout body menjadi sukses/envelope palsu")
				}
			}
			observed := await(t, cancelled)
			if observed.Sub(at) < 800*time.Millisecond || observed.Sub(at) > 2500*time.Millisecond {
				t.Fatal("fase timeout tidak memakai konfigurasi yang diuji")
			}
			await(t, returned)
			waiting.Store(false)
			if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 200 {
				t.Fatal("lease timeout tidak kembali")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayStreamByteProgressResetsIdle(t *testing.T) {
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i, part := range []string{"data: ", "o", "k\n\n"} {
			if i > 0 {
				timer := time.NewTimer(600 * time.Millisecond)
				select {
				case <-timer.C:
				case <-r.Context().Done():
					timer.Stop()
					return
				}
			}
			if testutil.WriteFragments(r.Context(), w, []byte(part), []int{1}) != nil {
				return
			}
		}
	}, func(http.ResponseWriter, *http.Request) { t.Error("idle reset mengganti upstream") }, "stream_idle_seconds: 120", "stream_idle_seconds: 1", "total_request_seconds: 1800", "total_request_seconds: 4")
	server := httptest.NewServer(h.handler)
	defer server.Close()
	server.Client().Timeout = 3 * time.Second
	response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`))
	if err != nil {
		t.Fatal("stream progress tidak dimulai")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil || string(data) != "data: ok\n\n" {
		t.Fatal("kemajuan fragmen byte dianggap idle/event diubah")
	}
	h.verifyPath(t)
}

func TestGatewayStreamEncodedFirstEvent(t *testing.T) {
	next := make(chan struct{})
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Content-Encoding", "gzip")
		compressor := gzip.NewWriter(w)
		defer compressor.Close()
		if _, err := io.WriteString(compressor, "data: ready\n\n"); err != nil {
			return
		}
		if compressor.Flush() != nil {
			return
		}
		if http.NewResponseController(w).Flush() != nil {
			return
		}
		select {
		case <-next:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(compressor, "data: [DONE]\n\n")
	}, func(http.ResponseWriter, *http.Request) { t.Error("gzip mengganti upstream") })
	h.clients["warp-1"].Transport.(*http.Transport).DisableCompression = true
	server := httptest.NewServer(h.handler)
	defer server.Close()
	server.Client().Timeout = 3 * time.Second
	request := gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`)
	// Cegah auto-decode client gateway agar header/body opaque dapat diperiksa.
	server.Client().Transport.(*http.Transport).DisableCompression = true
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal("gzip event pertama tertahan")
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Encoding") != "gzip" {
		t.Fatal("metadata gzip opaque tidak diteruskan")
	}
	decoded, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatal("gzip stream invalid")
	}
	defer decoded.Close()
	consumer := bufio.NewReader(decoded)
	if first, err := readEvent(consumer); err != nil || first != "ready" {
		t.Fatal("gzip event pertama tidak melewati barrier")
	}
	close(next)
	if last, err := readEvent(consumer); err != nil || last != "[DONE]" {
		t.Fatal("terminal gzip berubah")
	}
	if _, err := consumer.ReadByte(); err != io.EOF {
		t.Fatal("gzip selesai tidak memiliki EOF valid")
	}
	h.verifyPath(t)
}

func TestGatewayStreamRejectedBeforeNetwork(t *testing.T) {
	h := newModelHarness(t, func(http.ResponseWriter, *http.Request) { t.Error("request stream ditolak mencapai upstream") }, func(http.ResponseWriter, *http.Request) { t.Error("request stream ditolak mencapai upstream lain") })
	for _, path := range []string{"/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
		for _, tc := range []struct {
			body, token, header string
			status              int
		}{
			{`{"model":"coding-default","stream":true}`, "wrong", "", 401},
			{`{"model":"coding-default","stream":true}`, modelAdminToken, "", 401},
			{`{"model":"missing","stream":true}`, modelGatewayToken, "", 404},
			{`{"model":"coding-default","stream":true,"str\u0065am":false}`, modelGatewayToken, "", 400},
			{`{"model":"coding-default","stream":null}`, modelGatewayToken, "", 400},
			{`{"model":"coding-default","stream":true}`, modelGatewayToken, "X-Relay-Target", 400},
			{`{"model":"coding-default","stream":true,"x":"` + strings.Repeat("x", 4096) + `"}`, modelGatewayToken, "", 413},
		} {
			r := httptest.NewRequest("POST", path, strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer "+tc.token)
			r.Header.Set("Content-Type", "application/json")
			if tc.header != "" {
				r.Header.Set(tc.header, "https://ignored.invalid")
			}
			w := httptest.NewRecorder()
			h.handler.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatal("request stream invalid tidak ditolak")
			}
			if strings.Contains(w.Body.String(), modelGatewayToken) || strings.Contains(w.Body.String(), modelPrimaryKey) {
				t.Fatal("secret bocor pada error stream")
			}
		}
	}
	if h.primary.Snapshot().Total != 0 || h.secondary.Snapshot().Total != 0 || h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 0 {
		t.Fatal("penolakan stream membuka CONNECT/dispatch")
	}
}

func TestGatewayStreamAmbiguousPOST(t *testing.T) {
	for _, warm := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "reused"}[warm], func(t *testing.T) {
			var drop atomic.Bool
			failed := make(chan error, 1)
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if drop.Load() {
					failed <- testutil.DropConnection(w)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_ = testutil.WriteFragments(r.Context(), w, []byte("data: [DONE]\n\n"), []int{1})
			}, func(http.ResponseWriter, *http.Request) { t.Error("POST stream ambigu mengganti upstream") })
			server := httptest.NewServer(h.handler)
			defer server.Close()
			server.Client().Timeout = 3 * time.Second
			call := func() *http.Response {
				response, err := server.Client().Do(gatewayStreamRequest(t, server, "/v1/chat/completions", `{"model":"coding-default","stream":true}`))
				if err != nil {
					t.Fatal("request gateway tidak selesai")
				}
				return response
			}
			if warm {
				response := call()
				_, err := io.Copy(io.Discard, response.Body)
				response.Body.Close()
				if err != nil || response.StatusCode != 200 {
					t.Fatal("warmup stream gagal")
				}
			}
			drop.Store(true)
			response := call()
			data, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != 502 || !bytes.Contains(data, []byte(`"code":"upstream_transport_error"`)) {
				t.Fatal("POST stream ambigu tidak gagal eksplisit")
			}
			if err := await(t, failed); err != nil {
				t.Fatal("fixture ambiguous drop gagal")
			}
			want := 1
			if warm {
				want = 2
			}
			if h.primary.Snapshot().Total != want || h.secondary.Snapshot().Total != 0 || len(h.a.Snapshot().Connects) != 1 {
				t.Fatal("POST stream ambigu diulang")
			}
			h.verifyPath(t)
		})
	}
}
