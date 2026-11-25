package gateway

import (
	"errors"
	"io"
	"net/http"
	"opencode-infinity/internal/config"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type waitingStream struct {
	closed chan struct{}
	once   sync.Once
	closes atomic.Int32
}

func (b *waitingStream) Read([]byte) (int, error) { <-b.closed; return 0, io.ErrClosedPipe }
func (b *waitingStream) Close() error {
	b.once.Do(func() { b.closes.Add(1); close(b.closed) })
	return nil
}

func TestGatewayStreamIdleDeadline(t *testing.T) {
	body := &waitingStream{closed: make(chan struct{})}
	c, err := New(modelConfig(t, "stream_idle_seconds: 120", "stream_idle_seconds: 1"), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	response, err := c.Chat(chatRequest(`{"model":"coding-default","stream":true}`), "")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	result := make(chan error, 1)
	go func() { _, err := response.Body.Read(make([]byte, 1)); result <- err }()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("idle dianggap EOF sukses")
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("idle upstream tidak membatalkan Read")
	}
	response.Body.Close()
	response.Body.Close()
	if body.closes.Load() != 1 {
		t.Fatal("idle menutup jaringan lebih dari sekali")
	}
	// Waktu caller tidak membaca bukan upstream idle.
	next, err := c.Chat(chatRequest(`{"model":"coding-default","stream":true}`), "")
	if err != nil {
		t.Fatal("slot idle tidak dilepas")
	}
	next.Body.Close()
}

func TestGatewayStreamMIME(t *testing.T) {
	for _, value := range []string{"text/event-stream", "TEXT/EVENT-STREAM; charset=utf-8"} {
		if !IsEventStream(&http.Response{Header: http.Header{"Content-Type": {value}}}) {
			t.Fatal("media SSE valid tidak dikenali")
		}
	}
	for _, value := range []string{"application/json", "text/event-stream-bad", "text/event-stream; broken"} {
		if IsEventStream(&http.Response{Header: http.Header{"Content-Type": {value}}}) {
			t.Fatal("media non-SSE dikenali sebagai stream")
		}
	}
}

func TestGatewayStreamIdleOnlyWhileReading(t *testing.T) {
	body := &countedBody{Reader: strings.NewReader(strings.Repeat("x", 256))}
	var releases atomic.Int32
	managed := &responseBody{body: body, idle: 100 * time.Millisecond, cancel: func() {}, release: func() { releases.Add(1) }}
	for range 20 {
		if n, err := managed.Read(make([]byte, 1)); n != 1 || err != nil {
			t.Fatal("kemajuan byte dianggap idle")
		}
	}
	time.Sleep(150 * time.Millisecond)
	if body.closes.Load() != 0 || releases.Load() != 0 {
		t.Fatal("waktu backpressure/caller dianggap idle upstream")
	}
	for range 20 {
		if n, err := managed.Read(make([]byte, 1)); n != 1 || err != nil {
			t.Fatal("callback Read lama membatalkan Read berikutnya")
		}
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { managed.Close() })
	}
	wg.Wait()
	if body.closes.Load() != 1 || releases.Load() != 1 {
		t.Fatal("close/callback menggandakan cleanup")
	}
}

func TestGatewayStreamEmptyReadDoesNotCancel(t *testing.T) {
	body := &countedBody{Reader: strings.NewReader("still-active")}
	managed := &responseBody{body: body, idle: time.Second, cancel: func() {}, release: func() {}}
	defer managed.Close()
	if n, err := managed.Read(nil); n != 0 || err != nil || body.closes.Load() != 0 {
		t.Fatal("Read kosong membatalkan stream aktif")
	}
}

type noProgressStream struct {
	closed        atomic.Bool
	reads, closes atomic.Int32
}

func (b *noProgressStream) Read([]byte) (int, error) {
	b.reads.Add(1)
	if b.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}
func (b *noProgressStream) Close() error { b.closes.Add(1); b.closed.Store(true); return nil }
func TestGatewayStreamNoProgressIsBounded(t *testing.T) {
	body := &noProgressStream{}
	var released atomic.Int32
	managed := &responseBody{body: body, idle: time.Second, cancel: func() {}, release: func() { released.Add(1) }}
	if _, err := managed.Read(make([]byte, 1)); !errors.Is(err, io.ErrNoProgress) {
		t.Fatal("reader tanpa kemajuan tidak dibatasi eksplisit")
	}
	if body.reads.Load() > 100 || body.closes.Load() != 1 || released.Load() != 0 {
		t.Fatal("work/abort tanpa kemajuan tidak bounded atau lease dilepas dini")
	}
	managed.Close()
	managed.Close()
	if released.Load() != 1 || body.closes.Load() != 1 {
		t.Fatal("cleanup reader tanpa kemajuan tidak tepat sekali")
	}
}
