package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixedAdmissionCapacityBeforeDispatch(t *testing.T) {
	const primary = "/v1/chat/completions"
	const secondary = "/providers/secondary/v1/chat/completions"
	for _, tc := range []struct {
		name      string
		edits     []string
		stream    bool
		relayOnly bool
	}{
		{name: "global", edits: []string{"max_active_requests: 20", "max_active_requests: 1"}},
		{name: "relay", relayOnly: true, edits: []string{"max_inflight: 5", "max_inflight: 1"}},
		{name: "outbound", edits: []string{"outbound_connections: 32", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1"}},
		{name: "body", edits: []string{"body_buffer_bytes: 67108864", "body_buffer_bytes: 300000"}},
		{name: "stream", stream: true, edits: []string{"stream_buffer_bytes: 8388608", "stream_buffer_bytes: 32768"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := make(chan struct{}, 1)
			release := make(chan struct{})
			var hold atomic.Bool
			hold.Store(true)
			var once sync.Once
			unblock := func() { hold.Store(false); once.Do(func() { close(release) }) }
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.stream {
					w.Header().Set("Content-Type", "text/event-stream")
				} else {
					w.Header().Set("Content-Type", "application/json")
				}
				if hold.Load() {
					if tc.stream {
						_, _ = io.WriteString(w, "data: {\"held\":true}\n\n")
					} else {
						_, _ = io.WriteString(w, `{"held":`)
					}
					http.NewResponseController(w).Flush()
					started <- struct{}{}
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
					if !tc.stream {
						_, _ = io.WriteString(w, "true}")
					}
				} else if !tc.stream {
					_, _ = io.WriteString(w, `{}`)
				}
				if tc.stream {
					_, _ = io.WriteString(w, "data: [DONE]\n\n")
				}
			}, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`)
			}, tc.edits...)
			firstReturned := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Fixture-Admission-First") == "true" {
					defer close(firstReturned)
				}
				h.handler.ServeHTTP(w, r)
			}))
			defer server.Close()
			defer unblock()
			server.Client().Timeout = 3 * time.Second
			input := `{"model":"coding-default","stream":` + map[bool]string{false: "false", true: "true"}[tc.stream] + `}`
			first := gatewayStreamRequest(t, server, primary, input)
			first.Header.Set("X-Fixture-Admission-First", "true")
			firstDone := make(chan error, 1)
			go func() {
				response, err := server.Client().Do(first)
				if err == nil {
					_, err = io.Copy(io.Discard, response.Body)
					response.Body.Close()
					if response.StatusCode != 200 {
						err = fmt.Errorf("status awal fixture %d", response.StatusCode)
					}
				}
				firstDone <- err
			}()
			await(t, started)
			paths := []string{primary, secondary}
			if tc.relayOnly {
				paths = []string{primary}
			}
			for _, path := range paths {
				for _, unknown := range []bool{false, true} {
					beforeA, beforeB := h.a.Snapshot(), h.b.Snapshot()
					beforeP, beforeS := h.primary.Snapshot().Total, h.secondary.Snapshot().Total
					request := gatewayStreamRequest(t, server, path, input)
					if unknown {
						request.ContentLength = -1
					}
					response, err := server.Client().Do(request)
					if err != nil {
						t.Fatal("request kapasitas tidak selesai", err)
					}
					body, err := io.ReadAll(response.Body)
					response.Body.Close()
					if err != nil || response.StatusCode != 429 || !strings.Contains(string(body), `"code":"gateway_capacity_exceeded"`) || response.Header.Get("X-Infinity-Error-Source") != "gateway" || response.Header.Get("X-Request-ID") == "" {
						t.Fatal("penolakan kapasitas tidak memakai status/envelope gateway")
					}
					afterA, afterB := h.a.Snapshot(), h.b.Snapshot()
					if h.primary.Snapshot().Total != beforeP || h.secondary.Snapshot().Total != beforeS || afterA.Accepted != beforeA.Accepted || afterB.Accepted != beforeB.Accepted || len(afterA.Connects) != len(beforeA.Connects) || len(afterB.Connects) != len(beforeB.Connects) {
						t.Fatal("penolakan kapasitas menambah koneksi/CONNECT/dispatch")
					}
				}
			}
			if tc.relayOnly {
				response, err := server.Client().Do(gatewayStreamRequest(t, server, secondary, input))
				if err != nil {
					t.Fatal(err)
				}
				io.Copy(io.Discard, response.Body)
				response.Body.Close()
				if response.StatusCode != 200 || h.secondary.Snapshot().Total != 1 || h.primary.Snapshot().Total != 1 {
					t.Fatal("relay penuh menahan relay lain atau memicu fallback tersembunyi")
				}
			}
			unblock()
			if err := await(t, firstDone); err != nil {
				t.Fatal("request aktif tidak selesai", err)
			}
			await(t, firstReturned)
			response, err := server.Client().Do(gatewayStreamRequest(t, server, primary, input))
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode != 200 || h.primary.Snapshot().Total != 2 {
				t.Fatal("kapasitas tidak pulih setelah forwarding selesai")
			}
			h.verifyPath(t)
		})
	}
}
