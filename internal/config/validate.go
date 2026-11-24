package config

import (
	"errors"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func identifier(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for i, b := range []byte(s) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' {
			continue
		}
		if i > 0 && (b == '.' || b == '_' || b == '-') {
			continue
		}
		return false
	}
	return true
}
func printable(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for _, b := range []byte(s) {
		if b < 33 || b > 126 {
			return false
		}
	}
	return true
}
func validateDocument(d *document) error {
	if len(d.Relays) < 1 || len(d.Relays) > 128 || len(d.Upstreams) < 1 || len(d.Upstreams) > 64 {
		return errors.New("config_invalid_registry_size")
	}
	relays := make(map[string]bool, len(d.Relays))
	for _, r := range d.Relays {
		if !identifier(r.ID, 64) || relays[r.ID] {
			return errors.New("config_invalid_relay_id")
		}
		relays[r.ID] = true
	}
	upstreams := make(map[string]bool, len(d.Upstreams))
	models := 0
	for i := range d.Upstreams {
		u := &d.Upstreams[i]
		if !identifier(u.ID, 64) || upstreams[u.ID] {
			return errors.New("config_invalid_upstream_id")
		}
		upstreams[u.ID] = true
		if len(u.Relays) < 1 || len(u.Relays) > 128 || len(u.Models) < 1 || len(u.Models) > 256 {
			return errors.New("config_invalid_registry_size")
		}
		models += len(u.Models)
		if models > 1024 {
			return errors.New("config_invalid_registry_size")
		}
		allowed := make(map[string]bool, len(u.Relays))
		for _, id := range u.Relays {
			if !relays[id] || allowed[id] {
				return errors.New("config_invalid_relay_reference")
			}
			allowed[id] = true
		}
		aliases := make(map[string]bool, len(u.Models))
		for j := range u.Models {
			m := &u.Models[j]
			if !identifier(m.Alias, 128) || aliases[m.Alias] || !printable(m.UpstreamModel, 256) {
				return errors.New("config_invalid_model")
			}
			aliases[m.Alias] = true
			if !allowed[m.PrimaryRelay] || len(m.FallbackRelays) > 128 {
				return errors.New("config_invalid_primary_relay")
			}
			seen := map[string]bool{m.PrimaryRelay: true}
			for _, id := range m.FallbackRelays {
				if !allowed[id] || seen[id] {
					return errors.New("config_invalid_fallback_relay")
				}
				seen[id] = true
			}
			if m.Routing == "" {
				m.Routing = "fixed"
			}
			if m.Affinity == "" {
				m.Affinity = "none"
			}
			// shortcut: fixed/none saja; mode lain menunggu kontrak dan implementasi task pool/affinity.
			if m.Routing != "fixed" || m.Affinity != "none" {
				return errors.New("config_unsupported_routing")
			}
		}
	}
	if d.Server.DefaultUpstream == "" && len(d.Upstreams) == 1 {
		d.Server.DefaultUpstream = d.Upstreams[0].ID
	}
	if !upstreams[d.Server.DefaultUpstream] {
		return errors.New("config_invalid_default_upstream")
	}
	return nil
}

func port(s string) bool {
	if s == "" {
		return false
	}
	for _, b := range []byte(s) {
		if b < '0' || b > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && n <= 65535
}
func hostname(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	s = strings.TrimSuffix(s, ".")
	if s == "" || len(s) > 253 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, b := range []byte(label) {
			if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-') {
				return false
			}
		}
	}
	return true
}
func targetURL(raw string, relay bool) bool {
	if !printable(raw, 4096) || strings.ContainsAny(raw, "?#\\") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.User != nil || !hostname(u.Hostname()) {
		return false
	}
	if strings.Contains(u.Hostname(), ":") && !strings.HasPrefix(u.Host, "[") {
		return false
	}
	p := u.Port()
	if p != "" && !port(p) || strings.HasSuffix(u.Host, ":") {
		return false
	}
	if relay {
		return u.Scheme == "socks5h" && p != "" && u.Path == "" && u.RawPath == ""
	}
	if u.Scheme != "https" || u.Path != "" && !strings.HasPrefix(u.Path, "/") {
		return false
	}
	for _, part := range strings.Split(u.Path, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	escaped := u.EscapedPath()
	for i := 0; i < len(escaped); i++ {
		if escaped[i] != '%' {
			continue
		}
		if i+2 >= len(escaped) {
			return false
		}
		n, err := strconv.ParseUint(escaped[i+1:i+3], 16, 8)
		if err != nil || n == '.' || n == '/' || n == '\\' || n == '%' || n < 32 || n == 127 {
			return false
		}
		i += 2
	}
	return true
}
func listener(s string) (net.IP, string, bool) {
	h, p, err := net.SplitHostPort(s)
	ip := net.ParseIP(h)
	if err != nil || ip == nil || !port(p) {
		return nil, "", false
	}
	n, _ := strconv.Atoi(p)
	return ip, strconv.Itoa(n), true
}
func validateTargets(d *document) error {
	a, ap, aok := listener(d.Server.Listen)
	b, bp, bok := listener(d.Server.AdminListen)
	if !aok || !bok {
		return errors.New("config_invalid_listener")
	}
	overlap := a.Equal(b) || a.IsUnspecified() && (a.To4() == nil || b.To4() != nil) || b.IsUnspecified() && (b.To4() == nil || a.To4() != nil)
	if ap == bp && overlap {
		return errors.New("config_listener_conflict")
	}
	for _, r := range d.Relays {
		if !targetURL(r.ProxyURL, true) {
			return errors.New("config_invalid_proxy_url")
		}
	}
	for _, u := range d.Upstreams {
		if !targetURL(u.BaseURL, false) {
			return errors.New("config_invalid_upstream_url")
		}
	}
	return nil
}

// shortcut: hanya validasi anggaran; accounting/admission diterapkan task pemilik runtime.
func validateLimits(d *document) error {
	l := d.Limits
	s := d.Server
	t := d.Timeouts
	for _, n := range []int64{s.MaxActiveRequests, l.InboundConnections, l.OutboundConnections, l.Bindings, l.AdminSessions, l.ParallelProbes, l.ParallelStatus} {
		if n < 1 || n > math.MaxInt32 {
			return errors.New("config_invalid_count")
		}
	}
	for _, r := range d.Relays {
		if r.MaxInflight < 1 || r.MaxInflight > math.MaxInt32 {
			return errors.New("config_invalid_count")
		}
	}
	for _, n := range []int64{t.ConnectSeconds, t.ResponseHeadersSeconds, t.StreamIdleSeconds, t.TotalRequestSeconds} {
		if n < 1 || n > math.MaxInt64/int64(time.Second) {
			return errors.New("config_invalid_timeout")
		}
	}
	var total int64
	for _, n := range []int64{l.BodyBufferBytes, l.StreamBufferBytes, l.StateBytes, l.MonitoringBytes} {
		if n < 1 || n > math.MaxInt64-total {
			return errors.New("config_invalid_byte_budget")
		}
		total += n
	}
	if s.MaxBodyBytes < 1 || s.MaxBodyBytes > 10<<20 || l.BodyBufferBytes < s.MaxBodyBytes {
		return errors.New("config_invalid_body_limit")
	}
	if l.SnapshotBytes < 1 || l.SnapshotBytes > l.MonitoringBytes/l.ParallelStatus || l.ParallelProbes > l.OutboundConnections {
		return errors.New("config_inconsistent_limits")
	}
	return nil
}
