package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"opencode-infinity/internal/catalog"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/transport"
	"reflect"
	"sync"
	"time"
)

type Coordinator struct {
	registry        *catalog.Registry
	boundary        *transport.RequestBoundary
	admission       *pool.Manager
	dispatch        Dispatcher
	maxBody, budget int64
	catalogBudget   int64
	timeout         time.Duration
	idle            time.Duration
}

func (c *Coordinator) Models(id string) (*http.Response, error) {
	if c == nil || c.admission == nil {
		return nil, failure("invalid_request")
	}
	lease, ok := c.admission.Begin(c.catalogBudget)
	if !ok {
		return nil, failure("gateway_capacity_exceeded")
	}
	if id == "" {
		id = c.registry.DefaultUpstream()
	}
	upstream, ok := c.registry.Upstream(id)
	if !ok {
		lease.Release()
		return nil, failure("upstream_not_found")
	}
	if err := lease.AdmitLocal(); err != nil {
		lease.Release()
		return nil, failure("gateway_capacity_exceeded")
	}
	type model struct {
		ID     string `json:"id"`
		Object string `json:"object"`
		Owner  string `json:"owned_by"`
	}
	result := struct {
		Object string  `json:"object"`
		Data   []model `json:"data"`
	}{Object: "list", Data: make([]model, 0, len(upstream.Models))}
	for _, m := range upstream.Models {
		result.Data = append(result.Data, model{m.Alias, "model", upstream.ID})
	}
	data, _ := json.Marshal(result)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: &responseBody{body: io.NopCloser(bytes.NewReader(data)), release: lease.Release, cancel: func() {}}}, nil
}

func New(c *config.Config, d Dispatcher, clock func() time.Time, policy pool.Policy) (*Coordinator, *pool.Manager, error) {
	if !c.Validated() || nilDispatcher(d) {
		return nil, nil, errors.New("invalid_gateway_component")
	}
	owned := *c
	registry, _ := catalog.New(&owned)
	boundary, _ := transport.NewRequestBoundary(&owned)
	admission, err := pool.New(&owned, clock, policy)
	if err != nil {
		return nil, nil, errors.New("invalid_gateway_component")
	}
	// Dua salinan metadata maksimum: resolve coordinator dan boundary credential.
	// 128 byte membatasi header struct Model; setiap elemen slice string 16 byte pada amd64.
	var metadata int64
	for _, u := range owned.Upstreams() {
		size := int64(len(u.Models))*128 + int64(len(u.Relays))*16
		for _, m := range u.Models {
			size += int64(len(m.FallbackRelays)) * 16
		}
		if size > metadata {
			metadata = size
		}
	}
	maxBody := owned.Server().MaxBodyBytes
	// Dua buffer wire/remap; 256 KiB untuk scanner JSON, header, key dan copy respons 32 KiB.
	budget := 2*(maxBody+1) + 2*metadata + 256*1024
	return &Coordinator{registry: registry, boundary: boundary, admission: admission, dispatch: d, maxBody: maxBody, budget: budget, catalogBudget: 2*metadata + 256*1024, timeout: time.Duration(owned.Timeouts().TotalRequestSeconds) * time.Second, idle: time.Duration(owned.Timeouts().StreamIdleSeconds) * time.Second}, admission, nil
}

func nilDispatcher(d Dispatcher) bool {
	if d == nil {
		return true
	}
	v := reflect.ValueOf(d)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func failure(code string) *Failure {
	status := http.StatusBadRequest
	switch code {
	case "upstream_not_found", "model_not_found":
		status = 404
	case "request_too_large":
		status = 413
	case "gateway_capacity_exceeded":
		status = 429
	case "no_eligible_relay":
		status = 503
	case "unsupported_api":
		status = 501
	case "unsupported_content_encoding":
		status = 415
	case "upstream_transport_error", "upstream_redirect_rejected":
		status = 502
	case "upstream_timeout":
		status = 504
	}
	return &Failure{Code: code, Status: status}
}

func (c *Coordinator) Chat(in *http.Request, id string) (*http.Response, error) {
	if c == nil || c.admission == nil || in == nil || in.Body == nil || in.Method != http.MethodPost {
		return nil, failure("invalid_request")
	}
	reservation, ok := c.admission.Begin(c.budget)
	if !ok {
		return nil, failure("gateway_capacity_exceeded")
	}
	transferred := false
	defer func() {
		if !transferred {
			reservation.Release()
		}
	}()
	defer in.Body.Close()
	ctx, cancel := context.WithTimeout(in.Context(), c.timeout)
	defer func() {
		if !transferred {
			cancel()
		}
	}()
	// Persiapan juga memvalidasi header sebelum alokasi/membaca body; body ini belum dikirim.
	prepared, err := c.boundary.ChatCompletions(in, id)
	if err != nil {
		return nil, failure(err.Error())
	}
	if in.ContentLength > c.maxBody {
		return nil, failure("request_too_large")
	}
	body := make([]byte, c.maxBody+1)
	n, err := io.ReadFull(in.Body, body)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, failure("invalid_request")
	}
	if int64(n) > c.maxBody {
		return nil, failure("request_too_large")
	}
	body = body[:n]
	parsed, err := parseChat(body)
	if err != nil {
		return nil, failure(err.Error())
	}
	_, model, ok := c.registry.Resolve(id, parsed.alias)
	if !ok {
		return nil, failure("model_not_found")
	}
	replacement, _ := json.Marshal(model.UpstreamModel)
	if int64(len(body)-(parsed.end-parsed.start)+len(replacement)) > c.maxBody {
		return nil, failure("request_too_large")
	}
	streamReserved := parsed.stream
	var streamBytes int64
	if streamReserved {
		streamBytes = StreamCopyBytes
	}
	upstreamID := id
	if upstreamID == "" {
		upstreamID = c.registry.DefaultUpstream()
	}
	if err := reservation.AdmitFixed(upstreamID, model.Alias, streamBytes); err != nil {
		if errors.Is(err, pool.ErrCapacityExceeded) {
			return nil, failure("gateway_capacity_exceeded")
		}
		return nil, failure("no_eligible_relay")
	}
	relay, _ := c.registry.Relay(model.PrimaryRelay)
	mapped := parsed.remap(body, replacement)
	prepared = prepared.WithContext(ctx)
	prepared.Body = io.NopCloser(bytes.NewReader(mapped))
	prepared.ContentLength = int64(len(mapped))
	prepared.GetBody = nil
	if prepared.Header.Get("Content-Type") == "" {
		prepared.Header.Set("Content-Type", "application/json")
	}
	defer prepared.Body.Close()
	response, err := c.dispatch.Dispatch(prepared, relay)
	if err != nil || response == nil || response.Body == nil || response.StatusCode < 200 || response.StatusCode > 599 {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		var timeout net.Error
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
			return nil, failure("upstream_timeout")
		}
		return nil, failure("upstream_transport_error")
	}
	// Endpoint generasi tidak menerima respons 3xx, termasuk 304 tanpa request conditional.
	// Tolak sebelum forwarding dan tanpa membaca body atau mengulang POST.
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		response.Body.Close()
		return nil, failure("upstream_redirect_rejected")
	}
	if IsEventStream(response) && !streamReserved && !reservation.ReserveStream(StreamCopyBytes) {
		response.Body.Close()
		return nil, failure("gateway_capacity_exceeded")
	}
	managed := &responseBody{body: response.Body, release: reservation.Release, cancel: cancel}
	if IsEventStream(response) {
		managed.idle = c.idle
	}
	response.Body = managed
	// Timeout/disconnect juga menutup respons yang belum dibaca caller, tanpa menunggu cleanup fixture.
	context.AfterFunc(ctx, managed.abort)
	transferred = true
	return response, nil
}

type responseBody struct {
	body        io.ReadCloser
	closeOnce   sync.Once
	releaseOnce sync.Once
	err         error
	release     func()
	cancel      context.CancelFunc
	idle        time.Duration
}

func (b *responseBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	var n int
	var err error
	if b.idle > 0 {
		n, err = b.readIdle(p)
	} else {
		n, err = b.body.Read(p)
	}
	if err != nil {
		b.abort()
	}
	return n, err
}
func (b *responseBody) Close() error {
	b.abort()
	b.releaseOnce.Do(b.release)
	return b.err
}

// EOF/cancellation menutup jaringan; lease tetap dipegang sampai caller selesai menulis respons.
// Caller wajib Close juga setelah EOF, sama dengan kontrak http.Response.Body.
func (b *responseBody) abort() { b.closeOnce.Do(func() { b.cancel(); b.err = b.body.Close() }) }
