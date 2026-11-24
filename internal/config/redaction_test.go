package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.yaml.in/yaml/v4"
	"strings"
	"sync"
	"testing"
)

func TestSecretRedaction(t *testing.T) {
	c, err := loadTest(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	key, ok := c.APIKey("primary")
	if !ok || key.Reveal() != "synthetic-provider-credential" {
		t.Fatal("key terpisah tidak dimuat")
	}
	if c.GatewayToken().Reveal() != "synthetic-gateway-credential-12345" || c.AdminToken().Reveal() != "synthetic-admin-credential-67890" {
		t.Fatal("token role salah")
	}
	for _, value := range []any{c, *c, c.GatewayToken(), c.AdminToken(), key, c.Upstreams(), c.Relays()} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			out := fmt.Sprintf(format, value)
			if strings.Contains(out, "synthetic-") {
				t.Fatal("nilai secret bocor lewat fmt")
			}
		}
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "synthetic-") {
			t.Fatal("nilai secret bocor lewat JSON")
		}
		data, err = yaml.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "synthetic-") {
			t.Fatal("nilai secret bocor lewat YAML")
		}
	}
}
func TestErrorsRedacted(t *testing.T) {
	sentinel := "private-sentinel-DO-NOT-ECHO"
	cases := []string{
		validYAML + sentinel + ": [\n",
		strings.Replace(validYAML, "https://api.provider.example/v1", "https://user:"+sentinel+"@api.example/v1", 1),
		strings.Replace(validYAML, "gateway_token_env: GATEWAY_TOKEN", "gateway_token_file: /nonexistent/"+sentinel, 1),
		validYAML + "unknown: " + sentinel + "\n",
	}
	for _, s := range cases {
		c, err := loadTest(s)
		if err == nil || c != nil {
			t.Fatal("input invalid tidak ditolak")
		}
		for e := err; e != nil; e = errors.Unwrap(e) {
			for _, f := range []string{"%s", "%v", "%+v", "%#v"} {
				if strings.Contains(fmt.Sprintf(f, e), sentinel) {
					t.Fatal("input sensitif bocor lewat error")
				}
			}
		}
	}
}

func TestNestedPrivateSecretRedaction(t *testing.T) {
	c, err := loadTest(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := c.APIKey("primary")
	for _, value := range []any{
		struct{ token Secret }{c.GatewayToken()},
		struct{ token Secret }{c.AdminToken()},
		struct{ token Secret }{key},
		struct{ cfg Config }{*c},
		struct{ cfg *Config }{c},
	} {
		for _, format := range []string{"%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(format, value), "synthetic-") {
				t.Fatal("secret bocor lewat fmt field privat pembungkus")
			}
		}
	}
}

func TestConcurrentSnapshotReads(t *testing.T) {
	c, err := loadTest(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				key, ok := c.APIKey("primary")
				if !ok || key.Reveal() != "synthetic-provider-credential" || c.GatewayToken().Reveal() != "synthetic-gateway-credential-12345" {
					t.Error("secret snapshot berubah")
					return
				}
				if strings.Contains(fmt.Sprintf("%#v", struct{ cfg Config }{*c}), "synthetic-") {
					t.Error("secret concurrent bocor")
					return
				}
				us := c.Upstreams()
				us[0].Models[0].Alias = "changed"
			}
			next, err := loadTest(validYAML)
			if err != nil || !next.Validated() {
				t.Error("load independen gagal")
				return
			}
			bad, err := loadTest(strings.Replace(validYAML, "api_key_env: UPSTREAM_KEY", "api_key_env: ABSENT", 1))
			if err == nil || bad != nil {
				t.Error("load invalid concurrent diterima")
			}
		}()
	}
	wg.Wait()
}
