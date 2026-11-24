package access

import (
	"encoding/json"
	"fmt"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync"
	"testing"
)

const gatewayToken = "synthetic-gateway-credential-12345"
const adminToken = "synthetic-admin-credential-67890"
const upstreamKey = "synthetic-upstream-credential"

func snapshot(t *testing.T, gateway string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(strings.NewReader(string(data)), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": gateway, "INFINITY_ADMIN_TOKEN": adminToken, "INFINITY_UPSTREAM_KEY": upstreamKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal("fixture konfigurasi tidak valid", err)
	}
	return c
}
func TestRoleVerification(t *testing.T) {
	p, err := New(snapshot(t, gatewayToken))
	if err != nil || p == nil {
		t.Fatal("verifier valid ditolak", err)
	}
	if !p.Gateway(gatewayToken) || !p.Admin(adminToken) {
		t.Fatal("role valid ditolak")
	}
	for _, v := range []string{"", adminToken, upstreamKey, gatewayToken[:20], gatewayToken + "x", strings.ToUpper(gatewayToken)} {
		if p.Gateway(v) {
			t.Fatal("credential bukan gateway diterima")
		}
	}
	for _, v := range []string{"", gatewayToken, upstreamKey, adminToken[:20], adminToken + "x"} {
		if p.Admin(v) {
			t.Fatal("credential bukan admin diterima")
		}
	}
	for _, c := range []*config.Config{nil, {}} {
		if p, err := New(c); err == nil || p != nil {
			t.Fatal("snapshot tidak tervalidasi diterima")
		}
	}
	var nilPolicy *Tokens
	if nilPolicy.Gateway(gatewayToken) || (&Tokens{}).Admin(adminToken) {
		t.Fatal("verifier kosong menerima akses")
	}
}

func TestOpaqueTokensAndPrivacy(t *testing.T) {
	for _, value := range []string{gatewayToken, `synthetic,opaque:with"quote\and+/=`, strings.Repeat("x", 4096)} {
		p, err := New(snapshot(t, value))
		if err != nil {
			t.Fatal(err)
		}
		if !p.Gateway(value) || p.Gateway(value+"x") {
			t.Fatal("token opaque/cap berubah")
		}
		for _, object := range []any{p, *p, struct{ policy Tokens }{*p}} {
			for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
				out := fmt.Sprintf(format, object)
				for _, secret := range []string{value, adminToken, upstreamKey} {
					if strings.Contains(out, secret) {
						t.Fatal("verifier membocorkan credential")
					}
				}
			}
			data, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), value) {
				t.Fatal("JSON verifier bocor")
			}
		}
	}
}
func TestConcurrentRoleChecks(t *testing.T) {
	p, err := New(snapshot(t, gatewayToken))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				if !p.Gateway(gatewayToken) || p.Gateway(adminToken) || !p.Admin(adminToken) || p.Admin(upstreamKey) {
					t.Error("verifikasi serentak salah")
					return
				}
			}
		}()
	}
	wg.Wait()
}
