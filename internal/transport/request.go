// Package transport menyiapkan pesan HTTP untuk transport upstream tepercaya.
package transport

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/catalog"
	"opencode-infinity/internal/config"
	"strings"
)

// RequestBoundary mengikat metadata dan key pada satu snapshot tervalidasi.
type RequestBoundary struct {
	snapshot *config.Config
	registry *catalog.Registry
}

// shortcut: instance/snapshot baru bukan restart proses; pembuktiannya menunggu perakitan/lifecycle gateway.
func NewRequestBoundary(c *config.Config) (*RequestBoundary, error) {
	if !c.Validated() {
		return nil, errors.New("request_unvalidated_config")
	}
	// Miliki value snapshot sendiri agar assignment object caller tidak mencampur metadata dan key.
	owned := *c
	r, err := catalog.New(&owned)
	if err != nil {
		return nil, errors.New("request_unvalidated_config")
	}
	return &RequestBoundary{snapshot: &owned, registry: r}, nil
}

// shortcut: hanya preparation; perakitan transport harus menegakkan SOCKS5/TLS/no-redirect/no-retry dan admission.
// Pesan hasil memuat credential; hanya komponen server tepercaya boleh menggunakannya, tanpa dump/log.
func (b *RequestBoundary) ChatCompletions(in *http.Request, id string) (*http.Request, error) {
	if b == nil || b.registry == nil || b.snapshot == nil {
		return nil, errors.New("request_unvalidated_config")
	}
	if in == nil || in.Method != http.MethodPost || in.ContentLength < -1 || ((in.Body == nil || in.Body == http.NoBody) && in.ContentLength != 0) {
		return nil, errors.New("invalid_upstream_request")
	}

	if id == "" {
		id = b.registry.DefaultUpstream()
	}
	u, ok := b.registry.Upstream(id)
	if !ok {
		return nil, errors.New("upstream_not_found")
	}
	key, ok := b.snapshot.APIKey(id)
	if !ok {
		return nil, errors.New("upstream_key_unavailable")
	}
	headers, err := forwardHeaders(in.Header)
	if err != nil {
		return nil, err
	}
	target := strings.TrimSuffix(u.BaseURL, "/") + "/chat/completions"
	out, err := http.NewRequestWithContext(in.Context(), http.MethodPost, target, in.Body)
	if err != nil {
		return nil, errors.New("invalid_upstream_request")
	}
	out.ContentLength = in.ContentLength
	out.GetBody = nil
	out.Header = headers
	out.Header.Set("Authorization", "Bearer "+key.Reveal())
	return out, nil
}

func validHeaderValue(s string) bool {
	for _, b := range []byte(s) {
		if b != '\t' && (b < 32 || b > 126) {
			return false
		}
	}
	return true
}
func forwardHeaders(source http.Header) (http.Header, error) {
	fail := func() (http.Header, error) { return nil, errors.New("invalid_forward_header") }
	hop := make(map[string]bool)
	connectionBytes := 0
	for name, values := range source {
		if strings.EqualFold(name, "X-Relay-Target") || strings.EqualFold(name, "X-Relay-Path") {
			return nil, errors.New("forbidden_target_override")
		}
		if !strings.EqualFold(name, "Connection") {
			continue
		}
		for _, value := range values {
			if len(value) > 8192 || !validHeaderValue(value) {
				return fail()
			}
			connectionBytes += len(name) + len(value) + 4
			if connectionBytes > 8192 {
				return fail()
			}
			for _, token := range strings.Split(value, ",") {
				hop[strings.ToLower(strings.TrimSpace(token))] = true
			}
		}
	}
	out := make(http.Header)
	seen := make(map[string]bool)
	size := 0
	for name, values := range source {
		key := strings.ToLower(name)
		if key != "content-type" && key != "accept" && key != "content-encoding" {
			continue
		}
		if seen[key] || len(values) == 0 {
			return fail()
		}
		seen[key] = true
		if key == "content-encoding" {
			if len(values) != 1 || !strings.EqualFold(values[0], "identity") {
				return nil, errors.New("unsupported_content_encoding")
			}
			continue
		}
		if key == "content-type" && len(values) != 1 {
			return fail()
		}
		for _, value := range values {
			if len(value) > 8192 || !validHeaderValue(value) {
				return fail()
			}
			size += len(name) + len(value) + 4
			if size > 8192 {
				return fail()
			}
			if !hop[key] {
				out.Add(name, value)
			}
		}
	}
	return out, nil
}
