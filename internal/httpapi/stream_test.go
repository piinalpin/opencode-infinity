package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type controlledStreamWriter struct {
	*httptest.ResponseRecorder
	mu        sync.Mutex
	when      time.Time
	changed   chan struct{}
	allow     chan struct{}
	started   chan struct{}
	startOnce sync.Once
	deadlines atomic.Int32
	flushes   atomic.Int32
	failFlush int32
	short     bool
}

func newControlledWriter(block bool) *controlledStreamWriter {
	w := &controlledStreamWriter{ResponseRecorder: httptest.NewRecorder(), changed: make(chan struct{}), allow: make(chan struct{}), started: make(chan struct{})}
	if !block {
		close(w.allow)
	}
	return w
}
func (w *controlledStreamWriter) SetWriteDeadline(when time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.when = when
	w.deadlines.Add(1)
	close(w.changed)
	w.changed = make(chan struct{})
	return nil
}
func (w *controlledStreamWriter) FlushError() error {
	if w.flushes.Add(1) == w.failFlush {
		return errors.New("synthetic_flush_failure")
	}
	return nil
}
func (w *controlledStreamWriter) Write(p []byte) (int, error) {
	w.startOnce.Do(func() { close(w.started) })
	for {
		w.mu.Lock()
		when, changed := w.when, w.changed
		w.mu.Unlock()
		var timer *time.Timer
		var timeout <-chan time.Time
		if !when.IsZero() {
			timer = time.NewTimer(time.Until(when))
			timeout = timer.C
		}
		select {
		case <-w.allow:
			if timer != nil {
				timer.Stop()
			}
			if w.short {
				return 0, nil
			}
			return w.ResponseRecorder.Write(p)
		case <-changed:
			if timer != nil {
				timer.Stop()
			}
		case <-timeout:
			return 0, os.ErrDeadlineExceeded
		}
	}
}

type countedStreamReader struct {
	remaining     int
	reads, closes atomic.Int32
	eofWithData   bool
}

func (b *countedStreamReader) Read(p []byte) (int, error) {
	b.reads.Add(1)
	if b.remaining == 0 {
		return 0, io.EOF
	}
	n := min(len(p), b.remaining)
	clear(p[:n])
	b.remaining -= n
	if b.remaining == 0 && b.eofWithData {
		return n, io.EOF
	}
	return n, nil
}
func (b *countedStreamReader) Close() error { b.closes.Add(1); return nil }
func streamAPIRequest(ctx context.Context) *http.Request {
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default","stream":true}`)).WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	return r
}
func waitStream[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("sink/callback streaming tidak selesai")
		var zero T
		return zero
	}
}

func TestGatewayStreamBackpressureAndLease(t *testing.T) {
	for _, eof := range []bool{false, true} {
		size := gateway.StreamCopyBytes * 3
		if eof {
			size = 100
		}
		body := &countedStreamReader{remaining: size, eofWithData: eof}
		var calls atomic.Int32
		h, err := fixtureModelAPI(apiConfig(t, "max_active_requests: 20", "max_active_requests: 1"), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body}, nil
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		writer := newControlledWriter(true)
		done := make(chan any, 1)
		go func() { defer func() { done <- recover() }(); h.ServeHTTP(writer, streamAPIRequest(ctx)) }()
		waitStream(t, writer.started)
		if body.reads.Load() != 1 {
			t.Fatal("read-ahead melewati buffer tetap ketika sink terblokir")
		}
		blocked := httptest.NewRecorder()
		h.ServeHTTP(blocked, streamAPIRequest(t.Context()))
		if blocked.Code != 429 || calls.Load() != 1 {
			t.Fatal("lease dilepas saat forwarding masih berjalan")
		}
		cancel()
		if value := waitStream(t, done); value != http.ErrAbortHandler {
			t.Fatal("disconnect tidak mengakhiri framing stream")
		}
		if body.closes.Load() != 1 {
			t.Fatal("jaringan tidak ditutup tepat sekali")
		}
		after := httptest.NewRecorder()
		h.ServeHTTP(after, streamAPIRequest(t.Context()))
		if after.Code != 200 || calls.Load() != 2 {
			t.Fatal("resource sink gagal tidak dilepas")
		}
	}
}

func TestGatewayStreamWriterFailures(t *testing.T) {
	for _, mode := range []string{"unsupported", "flush", "short"} {
		body := &countedStreamReader{remaining: 10}
		h, err := fixtureModelAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}, "Content-Encoding": {"gzip"}}, Body: body}, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if mode == "unsupported" {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, streamAPIRequest(t.Context()))
			if w.Code != 501 || w.Header().Get("Content-Encoding") != "" || body.reads.Load() != 0 {
				t.Fatal("unsupported writer memulai stream/metadata encoding salah")
			}
		} else {
			w := newControlledWriter(false)
			if mode == "flush" {
				w.failFlush = 2
			} else {
				w.short = true
			}
			func() {
				defer func() {
					if recover() != http.ErrAbortHandler {
						t.Error("writer/flush gagal diberi EOF sukses")
					}
				}()
				h.ServeHTTP(w, streamAPIRequest(t.Context()))
			}()
		}
		if body.closes.Load() != 1 {
			t.Fatal("writer failure tidak menutup jaringan sekali")
		}
	}
}

func TestGatewayStreamWriteDeadlineAndCallback(t *testing.T) {
	h := modelHandler{idle: 30 * time.Millisecond, total: time.Second}
	writer := newControlledWriter(true)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stream, err := h.prepareStream(writer, streamAPIRequest(ctx), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.copy(strings.NewReader("data: ready\n\n")); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("Write lambat tidak dibatasi deadline")
	}
	stream.close()
	// Callback yang dihentikan tidak boleh mengubah deadline koneksi sesudah return.
	secondWriter := newControlledWriter(false)
	other, stop := context.WithCancel(t.Context())
	stream, err = h.prepareStream(secondWriter, streamAPIRequest(other), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.copy(strings.NewReader("data: ready\n\n")); err != nil {
		t.Fatal(err)
	}
	stream.close()
	count := secondWriter.deadlines.Load()
	stop()
	if count != secondWriter.deadlines.Load() {
		t.Fatal("callback lama mengubah deadline request berikutnya")
	}
}

type gatedDeadlineWriter struct {
	*controlledStreamWriter
	entered, release chan struct{}
}

func (w *gatedDeadlineWriter) SetWriteDeadline(when time.Time) error {
	if !when.IsZero() && !when.After(time.Now()) {
		close(w.entered)
		<-w.release
	}
	return w.controlledStreamWriter.SetWriteDeadline(when)
}
func TestGatewayStreamCallbackJoin(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	writer := &gatedDeadlineWriter{controlledStreamWriter: newControlledWriter(false), entered: make(chan struct{}), release: make(chan struct{})}
	h := modelHandler{idle: time.Second, total: 2 * time.Second}
	stream, err := h.prepareStream(writer, streamAPIRequest(ctx), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	waitStream(t, writer.entered)
	done := make(chan struct{})
	go func() { stream.close(); close(done) }()
	select {
	case <-done:
		t.Fatal("handler kembali sebelum callback deadline selesai")
	default:
	}
	close(writer.release)
	waitStream(t, done)
	writer.mu.Lock()
	when := writer.when
	writer.mu.Unlock()
	if !when.IsZero() {
		t.Fatal("callback lama menimpa cleanup deadline koneksi")
	}
}
