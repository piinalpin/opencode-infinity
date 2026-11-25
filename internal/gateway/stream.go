package gateway

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"sync"
	"time"
)

// StreamCopyBytes adalah ukuran copy, bukan batas ukuran event SSE.
const StreamCopyBytes = 32 * 1024

// IsEventStream memakai metadata upstream, bukan header pemilih target dari client.
func IsEventStream(response *http.Response) bool {
	if response == nil {
		return false
	}
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	return err == nil && media == "text/event-stream"
}

var errStreamIdle = errors.New("upstream_idle_timeout")

// Timer hanya hidup selama Read, bukan selama Write/Flush akibat backpressure downstream.
func (b *responseBody) readIdle(p []byte) (int, error) {
	var mu sync.Mutex
	ended, expired := false, false
	callbackDone := make(chan struct{})
	timer := time.AfterFunc(b.idle, func() {
		defer close(callbackDone)
		mu.Lock()
		if ended {
			mu.Unlock()
			return
		}
		expired = true
		mu.Unlock()
		b.abort()
	})
	var n int
	var err error
	// shortcut: paling banyak 100 Read tanpa kemajuan; tinjau saat adapter runtime membuktikan kebutuhan lain.
	for empty := 0; ; empty++ {
		n, err = b.body.Read(p)
		if n > 0 || err != nil {
			break
		}
		if empty >= 99 {
			err = io.ErrNoProgress
			break
		}
	}
	mu.Lock()
	ended = true
	timedOut := expired
	mu.Unlock()
	// Stop(false) tidak menunggu AfterFunc; join agar callback lama tidak mengenai Read berikutnya.
	if !timer.Stop() {
		<-callbackDone
	}
	if timedOut {
		return n, errStreamIdle
	}
	return n, err
}
