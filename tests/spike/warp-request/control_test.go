// Helper spike hanya dikompilasi sebagai test; bukan gateway produksi.
package warp_request_test

import (
	"bytes"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const selectedModel = "deepseek-v4.1-flash-free"
const generationPath = "/inference/openai/v1/chat/completions"
const liveTarget = "https://opencode.ai" + generationPath
const maxRequest = 64 * 1024
const maxResponse = 1024 * 1024

type postBudget struct{ path string }

func (b postBudget) reserve() error {
	f, err := os.OpenFile(b.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("generation_budget_unavailable")
	}
	if err := f.Close(); err != nil {
		return errors.New("generation_budget_close")
	}
	return nil
}

type observation struct {
	Admitted, Denied, Dials, Connects, LocalWrites, Status int
	SOCKSHost, SOCKSPort                                   string
	Reads, Bytes, WireReads                                int
	FirstReadNS, LastReadNS                                int64
	TLSVerified                                            bool
	Events, Deltas, FirstDeltaWire, TerminalWire           int
	Done, Finish, EOF, Incremental                         bool
	LastRejected                                           string
	Failure                                                string
}
type recorder struct {
	mu    sync.Mutex
	value observation
}

func (r *recorder) update(f func(*observation)) { r.mu.Lock(); defer r.mu.Unlock(); f(&r.value) }
func (r *recorder) snapshot() observation       { r.mu.Lock(); defer r.mu.Unlock(); return r.value }

type control struct {
	client *http.Client
	trace  *recorder
	target *url.URL
	nonce  string
	budget postBudget
	slots  chan struct{}
}

// shortcut: HTTP/1.1, fixed endpoint dan SOCKS5 no-auth; perluas hanya pada task integrasi terkait.
func newControl(relay, target, nonce, budgetPath string, roots *x509.CertPool) (*control, error) {
	u, err := url.Parse(target)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != generationPath || u.RawPath != "" {
		return nil, errors.New("invalid_target")
	}
	h, p, err := net.SplitHostPort(relay)
	if err != nil || net.ParseIP(h) == nil {
		return nil, errors.New("relay_must_be_literal_ip")
	}
	port, err := strconv.Atoi(p)
	if err != nil || port <= 0 || port > 65535 || nonce == "" || budgetPath == "" {
		return nil, errors.New("invalid_control_config")
	}
	trace := &recorder{}
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}},
		DisableKeepAlives: true, MaxConnsPerHost: 1, MaxResponseHeaderBytes: 8192, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second,
	}
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		expected := u.Host
		if u.Port() == "" {
			expected = net.JoinHostPort(u.Hostname(), "443")
		}
		if address != expected {
			return nil, errors.New("target_dial_rejected")
		}
		return dialSOCKS(ctx, relay, u.Hostname(), u.Port(), trace)
	}
	return &control{client: &http.Client{Transport: tr, Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, trace: trace, target: u, nonce: nonce, budget: postBudget{budgetPath}, slots: make(chan struct{}, 8)}, nil
}
func (c *control) Close() { c.client.Transport.(*http.Transport).CloseIdleConnections() }
func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

type idleConn struct {
	net.Conn
	trace *recorder
}

func (c idleConn) Read(p []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.trace.update(func(o *observation) { o.WireReads++ })
	}
	return n, err
}

// Metadata diambil dari handshake yang benar-benar ditulis; tidak mengamati byte TLS/body.
func dialSOCKS(ctx context.Context, relay, host, port string, trace *recorder) (net.Conn, error) {
	if port == "" {
		port = "443"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || len(host) == 0 || len(host) > 255 || net.ParseIP(host) != nil {
		return nil, errors.New("invalid_socks_domain")
	}
	trace.update(func(o *observation) { o.Dials++ })
	raw, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", relay)
	if err != nil {
		return nil, errors.New("relay_dial_failed")
	}
	success := false
	defer func() {
		if !success {
			raw.Close()
		}
	}()
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()
	_ = raw.SetDeadline(time.Now().Add(10 * time.Second))
	if err := writeAll(raw, []byte{5, 1, 0}); err != nil {
		return nil, errors.New("socks_greeting_write")
	}
	var auth [2]byte
	if _, err := io.ReadFull(raw, auth[:]); err != nil || auth != [2]byte{5, 0} {
		return nil, errors.New("socks_greeting_rejected")
	}
	request := append([]byte{5, 1, 0, 3, byte(len(host))}, []byte(host)...)
	request = binary.BigEndian.AppendUint16(request, uint16(n))
	if err := writeAll(raw, request); err != nil {
		return nil, errors.New("socks_connect_write")
	}
	trace.update(func(o *observation) { o.SOCKSHost = host; o.SOCKSPort = port })
	var reply [4]byte
	if _, err := io.ReadFull(raw, reply[:]); err != nil || reply[0] != 5 || reply[1] != 0 || reply[2] != 0 {
		return nil, errors.New("socks_connect_rejected")
	}
	size := 0
	switch reply[3] {
	case 1:
		size = 4
	case 4:
		size = 16
	case 3:
		var b [1]byte
		if _, err := io.ReadFull(raw, b[:]); err != nil {
			return nil, errors.New("socks_reply_domain")
		}
		size = int(b[0])
	default:
		return nil, errors.New("socks_reply_type")
	}
	var discard [257]byte
	if _, err := io.ReadFull(raw, discard[:size+2]); err != nil {
		return nil, errors.New("socks_reply_truncated")
	}
	trace.update(func(o *observation) { o.Connects++ })
	_ = raw.SetDeadline(time.Time{})
	success = true
	return idleConn{raw, trace}, nil
}
func (c *control) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reject := func(status int, reason string) {
		c.trace.update(func(o *observation) { o.Denied++; o.LastRejected = reason })
		http.Error(w, reason, status)
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		reject(503, "control_capacity")
		return
	}
	if r.Method != "POST" || r.URL.Path != generationPath || r.URL.RawQuery != "" {
		reject(404, "invalid_route")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-T04-Run")), []byte(c.nonce)) != 1 {
		reject(401, "unauthorized_fixture")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequest))
	if err != nil {
		reject(413, "request_limit")
		return
	}
	var data map[string]any
	if json.Unmarshal(body, &data) != nil || data["model"] != selectedModel || data["stream"] != true {
		reject(400, "invalid_generation")
		return
	}
	tokens, ok := data["max_tokens"].(float64)
	if !ok || tokens < 1 || tokens > 512 || tokens != float64(int(tokens)) || data["max_completion_tokens"] != nil {
		reject(400, "invalid_token_cap")
		return
	}
	if messages, ok := data["messages"].([]any); !ok || len(messages) == 0 {
		reject(400, "missing_messages")
		return
	}
	if tools, exists := data["tools"]; exists {
		if list, ok := tools.([]any); !ok || len(list) > 0 {
			reject(400, "tools_disabled")
			return
		}
	}
	if err := c.budget.reserve(); err != nil {
		reject(409, "generation_budget_used")
		return
	}
	c.trace.update(func(o *observation) { o.Admitted++ })
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		TLSHandshakeDone: func(s tls.ConnectionState, err error) {
			c.trace.update(func(o *observation) {
				o.TLSVerified = err == nil && len(s.VerifiedChains) > 0 && s.ServerName == c.target.Hostname()
			})
		},
		WroteRequest: func(httptrace.WroteRequestInfo) { c.trace.update(func(o *observation) { o.LocalWrites++ }) },
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.target.String(), io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		c.trace.update(func(o *observation) { o.Failure = "request_build" })
		http.Error(w, "request_build", 500)
		return
	}
	req.Header = r.Header.Clone()
	for _, h := range []string{"X-T04-Run", "Connection", "Proxy-Authorization", "Proxy-Connection", "Transfer-Encoding", "Upgrade", "TE", "Trailer", "Content-Length"} {
		req.Header.Del(h)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-opencode-client", "desktop")
	resp, err := c.client.Do(req)
	if err != nil {
		c.trace.update(func(o *observation) { o.Failure = "transport" })
		http.Error(w, "upstream_transport", 502)
		return
	}
	defer resp.Body.Close()
	c.trace.update(func(o *observation) { o.Status = resp.StatusCode })
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		c.trace.update(func(o *observation) { o.Failure = "upstream_status_or_type" })
		http.Error(w, "upstream_status_or_type", 502)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	pumpStream(w, resp.Body, c.trace)
}
func pumpStream(w http.ResponseWriter, body io.Reader, trace *recorder) {
	fail := func(reason string) {
		trace.update(func(o *observation) { o.Failure = reason })
		panic(http.ErrAbortHandler)
	}
	buf := make([]byte, 4096)
	total := 0
	decoder := eventDecoder{trace: trace}
	for {
		n, err := body.Read(buf)
		if n > 0 {
			total += n
			if total > maxResponse {
				fail("stream_limit")
			}
			trace.update(func(o *observation) {
				o.Reads++
				o.Bytes += n
				if o.FirstReadNS == 0 {
					o.FirstReadNS = time.Now().UnixNano()
				}
				o.LastReadNS = time.Now().UnixNano()
			})
			if e := decoder.feed(buf[:n]); e != nil {
				fail(e.Error())
			}
			if written, e := w.Write(buf[:n]); e != nil || written != n {
				fail("client_write")
			}
			if e := http.NewResponseController(w).Flush(); e != nil {
				fail("client_flush")
			}
		}
		if err != nil {
			if err != io.EOF {
				fail("stream_read")
			}
			if strings.TrimSpace(string(decoder.buf)) != "" {
				fail("sse_truncated")
			}
			state := trace.snapshot()
			if !state.Done || !state.Finish || state.Deltas == 0 {
				fail("generation_incomplete")
			}
			trace.update(func(o *observation) {
				o.EOF = true
				o.Incremental = o.FirstDeltaWire > 0 && o.TerminalWire > o.FirstDeltaWire
			})
			return
		}
	}
}

// Framing minimum SSE endpoint uji; data hanya berada sementara di buffer terbatas.
type eventDecoder struct {
	buf   []byte
	trace *recorder
}

func (d *eventDecoder) feed(data []byte) error {
	for _, b := range data {
		d.buf = append(d.buf, b)
		if len(d.buf) > 64*1024 {
			return errors.New("sse_frame_limit")
		}
		if !bytes.HasSuffix(d.buf, []byte("\n\n")) && !bytes.HasSuffix(d.buf, []byte("\r\n\r\n")) {
			continue
		}
		frame := strings.ReplaceAll(string(d.buf), "\r\n", "\n")
		d.buf = d.buf[:0]
		var values []string
		for _, line := range strings.Split(frame, "\n") {
			if strings.HasPrefix(line, "event:") && strings.TrimSpace(strings.TrimPrefix(line, "event:")) == "error" {
				return errors.New("sse_error")
			}
			if strings.HasPrefix(line, "data:") {
				values = append(values, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		if len(values) == 0 {
			continue
		}
		payload := strings.Join(values, "\n")
		prior := d.trace.snapshot()
		if prior.Events >= 256 || prior.Done {
			return errors.New("sse_event_limit_or_after_terminal")
		}
		wire := prior.WireReads
		d.trace.update(func(o *observation) { o.Events++ })
		if payload == "[DONE]" {
			d.trace.update(func(o *observation) { o.Done = true; o.TerminalWire = wire })
			continue
		}
		var event struct {
			Error   json.RawMessage
			Choices []struct {
				Delta        map[string]json.RawMessage
				FinishReason *string `json:"finish_reason"`
			}
		}
		if json.Unmarshal([]byte(payload), &event) != nil {
			return errors.New("sse_json")
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return errors.New("sse_error")
		}
		delta, finish := false, false
		for _, choice := range event.Choices {
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				finish = true
			}
			for _, name := range []string{"content", "reasoning_content", "reasoning"} {
				if raw, ok := choice.Delta[name]; ok {
					var value string
					if json.Unmarshal(raw, &value) != nil {
						return errors.New("sse_delta_type")
					}
					delta = delta || value != ""
				}
			}
			if raw, ok := choice.Delta["tool_calls"]; ok {
				var tools []json.RawMessage
				if json.Unmarshal(raw, &tools) != nil {
					return errors.New("sse_tool_type")
				}
				delta = delta || len(tools) > 0
			}
		}
		d.trace.update(func(o *observation) {
			if delta {
				o.Deltas++
				if o.FirstDeltaWire == 0 {
					o.FirstDeltaWire = wire
				}
			}
			o.Finish = o.Finish || finish
		})
	}
	return nil
}
