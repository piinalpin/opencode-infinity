package integration_test

import (
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"opencode-infinity/internal/testutil"
)

func ledgerUpstream(t *testing.T, h http.HandlerFunc) *testutil.Upstream {
	t.Helper()
	u, err := testutil.NewUpstream(h)
	if err != nil {
		t.Fatalf("upstream fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := u.Close(); err != nil {
			t.Error(err)
		}
	})
	return u
}

func TestAmbiguousPOSTNotReplayed(t *testing.T) {
	for _, warm := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "reused"}[warm], func(t *testing.T) {
			var handlers atomic.Int32
			dropErrors := make(chan error, 1)
			u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				handlers.Add(1)
				if r.URL.Path == "/warm" {
					_, _ = io.WriteString(w, "ok")
					return
				}
				dropErrors <- testutil.DropConnection(w)
			})
			p := newProxy(t, "relay", u.Authority, u.Addr(), false)
			client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
			if warm {
				resp, err := post(t, client, u.Authority, "/warm")
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			resp, err := post(t, client, u.Authority, "/ambiguous")
			if err == nil {
				resp.Body.Close()
				t.Fatal("POST sudah diterima tetapi fault sebelum header tidak teramati")
			}
			if err := <-dropErrors; err != nil {
				t.Fatal(err)
			}
			want := 1
			if warm {
				want = 2
			}
			state := u.Snapshot()
			if state.Total != want || handlers.Load() != int32(want) || len(state.Dispatches) != want || !state.Dispatches[want-1].BodyRead || len(p.Snapshot().Connects) != 1 {
				t.Fatalf("ambiguity/replay: upstream=%+v proxy=%+v", state, p.Snapshot())
			}
		})
	}
}

func TestReusableConnectionCounters(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") })
	p := newProxy(t, "relay", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	for range 2 {
		resp, err := post(t, client, u.Authority, "/control")
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	dispatches, proxy := u.Snapshot().Dispatches, p.Snapshot()
	if len(dispatches) != 2 || len(proxy.Connects) != 1 {
		t.Fatalf("dispatch=%+v proxy=%+v", dispatches, proxy)
	}
	for _, d := range dispatches {
		if !d.BodyRead || d.BodyBytes != len(`{"fixture":true}`) || !d.Returned || d.Canceled {
			t.Fatalf("lifecycle=%+v", d)
		}
	}
	if err := testutil.VerifyProxyPath(dispatches, proxy); err != nil {
		t.Fatal(err)
	}
}

func TestRefusalBeforeDispatch(t *testing.T) {
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	p := newProxy(t, "refused", u.Authority, u.Addr(), true)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	resp, err := post(t, client, u.Authority, "/control")
	if err == nil {
		resp.Body.Close()
		t.Fatal("refusal sebelum tunnel malah mengirim request")
	}
	trace := p.Snapshot()
	if len(trace.Connects) != 1 || !trace.Connects[0].Refused || trace.Connects[0].TargetPeer != "" || u.Snapshot().Total != 0 {
		t.Fatalf("bukti no-send tidak valid: proxy=%+v upstream=%+v", trace, u.Snapshot())
	}
}
