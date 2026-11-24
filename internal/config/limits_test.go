package config

import (
	"strings"
	"testing"
)

func TestExplicitLimits(t *testing.T) {
	fields := []string{"max_body_bytes: 10485760", "max_active_requests: 20", "body_buffer_bytes: 67108864", "stream_buffer_bytes: 8388608", "state_bytes: 16777216", "monitoring_bytes: 8388608", "inbound_connections: 64", "outbound_connections: 32", "bindings: 1024", "admin_sessions: 32", "parallel_probes: 2", "parallel_status: 4", "snapshot_bytes: 262144", "connect_seconds: 10", "response_headers_seconds: 120", "stream_idle_seconds: 120", "total_request_seconds: 1800", "max_inflight: 5"}
	for _, field := range fields {
		key := strings.Split(field, ":")[0]
		for _, value := range []string{"0", "-1", "null", "1.5", "'20'"} {
			t.Run(key+"/"+value, func(t *testing.T) { rejectConfig(t, strings.Replace(validYAML, field, key+": "+value, 1)) })
		}
		t.Run(key+"/missing", func(t *testing.T) { rejectConfig(t, strings.Replace(validYAML, field, "", 1)) })
	}
	for name, pair := range map[string][2]string{
		"body_max":                   {"max_body_bytes: 10485760", "max_body_bytes: 10485761"},
		"body_aggregate":             {"body_buffer_bytes: 67108864", "body_buffer_bytes: 10485759"},
		"count_overflow":             {"inbound_connections: 64", "inbound_connections: 2147483648"},
		"int_overflow":               {"state_bytes: 16777216", "state_bytes: 9223372036854775808"},
		"sum_overflow":               {"state_bytes: 16777216", "state_bytes: 9223372036854775807"},
		"duration_overflow":          {"connect_seconds: 10", "connect_seconds: 9223372037"},
		"snapshot_aggregate":         {"snapshot_bytes: 262144", "snapshot_bytes: 2097153"},
		"snapshot_multiply_overflow": {"snapshot_bytes: 262144", "snapshot_bytes: 9223372036854775807"},
		"probes_connections":         {"parallel_probes: 2", "parallel_probes: 33"},
	} {
		t.Run(name, func(t *testing.T) { rejectConfig(t, strings.Replace(validYAML, pair[0], pair[1], 1)) })
	}
	for _, pair := range [][2]string{{"inbound_connections: 64", "inbound_connections: 2147483647"}, {"connect_seconds: 10", "connect_seconds: 9223372036"}, {"snapshot_bytes: 262144", "snapshot_bytes: 2097152"}} {
		if _, err := loadTest(strings.Replace(validYAML, pair[0], pair[1], 1)); err != nil {
			t.Fatal("boundary valid ditolak", err)
		}
	}
}
