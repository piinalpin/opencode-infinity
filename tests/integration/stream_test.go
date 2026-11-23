package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"opencode-infinity/internal/testutil"
)

var transcript = []string{
	`{"choices":[{"index":0,"delta":{"content":"Halo 🌏"}}],"extra":{"fixture":true}}`,
	`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_fixture","type":"function","function":{"name":"cek","arguments":"{\"x\":"}}]}}]}`,
	`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"1}"}}]}}]}`,
	`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"total_tokens":3}}`,
	`[DONE]`,
}

func TestFragmentedSSESemanticsAndFlush(t *testing.T) {
	for i, sizes := range [][]int{{1}, {2, 3, 1, 7}, {11, 5, 2}} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			next := make(chan struct{})
			finished := make(chan struct{})
			errors := make(chan error, 1)
			u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				for i, event := range transcript {
					if i == 1 {
						select {
						case <-next:
						case <-r.Context().Done():
							return
						}
					}
					if err := testutil.WriteFragments(r.Context(), w, []byte("data: "+event+"\n\n"), sizes); err != nil {
						errors <- err
						return
					}
				}
				close(finished)
			})
			p := newProxy(t, "relay", u.Authority, u.Addr(), false)
			client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
			resp, err := post(t, client, u.Authority, "/stream")
			if err != nil {
				t.Fatalf("event pertama belum di-flush sebelum gate: %v", err)
			}
			defer resp.Body.Close()
			r := bufio.NewReader(resp.Body)
			var content, args string
			for i, want := range transcript {
				got, err := readEvent(r)
				if err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Fatalf("event%d=%q want%q", i, got, want)
				}
				if i == 0 {
					select {
					case <-finished:
						t.Fatal("stream selesai sebelum gate dibuka")
					default:
					}
					close(next)
				}
				if got == "[DONE]" {
					continue
				}
				var actual, expected map[string]any
				if err := json.Unmarshal([]byte(got), &actual); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(want), &expected); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(actual, expected) {
					t.Fatal("semantik JSON berubah")
				}
				var chunk struct {
					Choices []struct {
						Delta struct {
							Content   string
							ToolCalls []struct {
								Index    int
								ID       string
								Function struct{ Name, Arguments string }
							} `json:"tool_calls"`
						}
					}
				}
				if err := json.Unmarshal([]byte(got), &chunk); err != nil {
					t.Fatal(err)
				}
				for _, c := range chunk.Choices {
					content += c.Delta.Content
					for _, tool := range c.Delta.ToolCalls {
						if tool.Index != 0 {
							t.Fatal("tool index berubah")
						}
						args += tool.Function.Arguments
					}
				}
			}
			if content != "Halo 🌏" || args != `{"x":1}` {
				t.Fatalf("content=%q args=%q", content, args)
			}
			if _, err := readEvent(r); err != io.EOF {
				t.Fatalf("akhir stream: %v", err)
			}
			await(t, finished)
			select {
			case err := <-errors:
				t.Fatal(err)
			default:
			}
			if u.Snapshot().Total != 1 {
				t.Fatal("stream didispatch ulang")
			}
		})
	}
}

func TestStreamStartedNotReplayed(t *testing.T) {
	drop := make(chan struct{})
	dropped := make(chan error, 1)
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if err := testutil.WriteFragments(r.Context(), w, []byte("data: "+transcript[0]+"\n\n"), []int{1, 3}); err != nil {
			dropped <- err
			return
		}
		select {
		case <-drop:
		case <-r.Context().Done():
			dropped <- r.Context().Err()
			return
		}
		dropped <- testutil.DropConnection(w)
	})
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	resp, err := post(t, client, u.Authority, "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)
	first, err := readEvent(r)
	if err != nil || first != transcript[0] {
		t.Fatalf("first=%q err=%v", first, err)
	}
	close(drop)
	if err := await(t, dropped); err != nil {
		t.Fatal(err)
	}
	if event, err := readEvent(r); err == nil || event == "[DONE]" || err == io.EOF {
		t.Fatalf("stream putus dianggap sukses: event=%q err=%v", event, err)
	}
	if u.Snapshot().Total != 1 || len(p.Snapshot().Connects) != 1 {
		t.Fatal("stream didispatch ulang")
	}
}

func waitReturned(t *testing.T, u *testutil.Upstream) testutil.UpstreamSnapshot {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		s := u.Snapshot()
		returned := 0
		for _, d := range s.Dispatches {
			if d.Returned {
				returned++
			}
		}
		if returned == s.Total && s.Rejected == 0 {
			return s
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatalf("handler belum kembali: %+v", s)
		}
	}
}

func TestClientDisconnectCancelsUpstream(t *testing.T) {
	for _, mode := range []string{"context", "body-close"} {
		t.Run(mode, func(t *testing.T) {
			observed := make(chan time.Time, 1)
			release := make(chan struct{})
			// Cancellation harus terlihat saat handler masih aktif, sebelum teardown.
			u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				if err := testutil.WriteFragments(r.Context(), w, []byte("data: ready\n\n"), []int{1}); err != nil {
					return
				}
				<-r.Context().Done()
				observed <- time.Now()
				select {
				case <-release:
				case <-time.After(time.Second):
				}
			})
			p := newProxy(t, "relay", u.Authority, u.Addr(), false)
			client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, "POST", "https://"+u.Authority+"/cancel", io.NopCloser(strings.NewReader(`{"fixture":true}`)))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if first, err := readEvent(bufio.NewReader(resp.Body)); err != nil || first != "ready" {
				t.Fatalf("first=%q err=%v", first, err)
			}
			start := time.Now()
			if mode == "context" {
				cancel()
			} else {
				if err := resp.Body.Close(); err != nil {
					t.Fatal(err)
				}
			}
			seen := await(t, observed)
			if seen.Before(start) || seen.Sub(start) > 2*time.Second {
				t.Fatalf("cancellation latency=%s", seen.Sub(start))
			}
			active := u.Snapshot()
			if active.Total != 1 || len(active.Dispatches) != 1 || active.Dispatches[0].Returned {
				t.Fatalf("cancellation bukan saat handler aktif: %+v", active)
			}
			close(release)
			state := waitReturned(t, u)
			if !state.Dispatches[0].Canceled || !state.Dispatches[0].BodyRead || len(p.Snapshot().Connects) != 1 {
				t.Fatalf("cancel lifecycle=%+v", state)
			}
		})
	}
}

func TestSuccessfulStreamIsNotCancellation(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if err := testutil.WriteFragments(r.Context(), w, []byte("data: [DONE]\n\n"), []int{1}); err != nil {
			return
		}
	})
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	resp, err := post(t, client, u.Authority, "/success")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)
	if event, err := readEvent(r); err != nil || event != "[DONE]" {
		t.Fatalf("event=%q err=%v", event, err)
	}
	if _, err := readEvent(r); err != io.EOF {
		t.Fatalf("EOF=%v", err)
	}
	state := waitReturned(t, u)
	if state.Total != 1 || state.Dispatches[0].Canceled {
		t.Fatalf("sukses tercatat cancellation: %+v", state)
	}
}

func TestConcurrentFixtureCleanup(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/error":
			_ = testutil.DropConnection(w)
		case "/cancel":
			if err := testutil.WriteFragments(r.Context(), w, []byte("data: ready\n\n"), []int{1}); err != nil {
				return
			}
			<-r.Context().Done()
		default:
			_, _ = io.WriteString(w, "ok")
		}
	})
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	var workers sync.WaitGroup
	for i := range 6 {
		workers.Go(func() {
			path := []string{"/success", "/cancel", "/error"}[i%3]
			resp, err := post(t, client, u.Authority, path)
			if path == "/error" {
				if err == nil {
					resp.Body.Close()
					t.Error("fault tidak muncul")
				}
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			if path == "/cancel" {
				if _, err := readEvent(bufio.NewReader(resp.Body)); err != nil {
					t.Error(err)
				}
			} else {
				if _, err := io.Copy(io.Discard, resp.Body); err != nil {
					t.Error(err)
				}
			}
			resp.Body.Close()
		})
	}
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	await(t, done)
	state := waitReturned(t, u)
	canceled := 0
	for _, d := range state.Dispatches {
		if d.Canceled {
			canceled++
		}
	}
	// Error yang memutus socket juga boleh membatalkan context; kedua cancel-stream wajib terlihat.
	if state.Total != 6 || canceled < 2 {
		t.Fatalf("mixed lifecycle=%+v", state)
	}
	if err := testutil.VerifyProxyPath(state.Dispatches, p.Snapshot()); err != nil {
		t.Fatal(err)
	}
	client.Transport.(*http.Transport).CloseIdleConnections()
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := u.Close(); err != nil {
		t.Fatal(err)
	}
	if p.Snapshot().Active != 0 || u.Snapshot().ActiveConnections != 0 {
		t.Fatal("socket fixture tertinggal")
	}
	if err := u.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestControlSSERejectsTruncatedAndLargeFrames(t *testing.T) {
	for _, payload := range []string{"data: incomplete", "data: " + strings.Repeat("x", testutil.MaxFrameBytes) + "\n\n"} {
		if _, err := readEvent(bufio.NewReader(strings.NewReader(payload))); err == nil {
			t.Fatal("frame invalid diterima")
		}
	}
}
