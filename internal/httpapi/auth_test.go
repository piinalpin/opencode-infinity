package httpapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const gatewayToken = "synthetic-gateway-credential-12345"
const adminToken = "synthetic-admin-credential-67890"
const upstreamKey = "synthetic-upstream-credential"

func testPolicy(t *testing.T) *access.Tokens { return policyForToken(t, gatewayToken) }
func policyForToken(t *testing.T, token string) *access.Tokens {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(strings.NewReader(string(data)), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": token, "INFINITY_ADMIN_TOKEN": adminToken, "INFINITY_UPSTREAM_KEY": upstreamKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := access.New(c)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestGatewayGuard(t *testing.T) {
	calls := 0
	guard, err := RequireGateway(testPolicy(t), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) }))
	if err != nil {
		t.Fatal("guard valid ditolak", err)
	}
	for _, path := range []string{"/v1/models", "/providers/primary/v1/models", "/v1/chat/completions", "/providers/primary/v1/chat/completions"} {
		for _, token := range []string{gatewayToken, "", adminToken, upstreamKey, "wrong"} {
			before := calls
			r := httptest.NewRequest(http.MethodPost, path, nil)
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			guard.ServeHTTP(w, r)
			want := 401
			step := 0
			if token == gatewayToken {
				want = 204
				step = 1
			}
			if w.Code != want || calls-before != step {
				t.Fatal("status/next-call role model salah")
			}
		}
	}
	if h, err := RequireGateway(nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})); err == nil || h != nil {
		t.Fatal("policy nil diterima")
	}
	if h, err := RequireGateway(&access.Tokens{}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})); err == nil || h != nil {
		t.Fatal("policy zero diterima")
	}
	if h, err := RequireGateway(testPolicy(t), nil); err == nil || h != nil {
		t.Fatal("handler nil diterima")
	}
}

func TestBearerHeaderPolicy(t *testing.T) {
	cases := []struct {
		header http.Header
		valid  bool
	}{
		{nil, false}, {http.Header{"Authorization": nil}, false}, {http.Header{"Authorization": {}}, false},
		{http.Header{"Authorization": {""}}, false},
		{http.Header{"Authorization": {"Bearer " + gatewayToken, "Bearer " + gatewayToken}}, false},
		{http.Header{"Authorization": {"Bearer " + gatewayToken}, "authorization": {"Bearer " + gatewayToken}}, false},
		{http.Header{"Authorization": nil, "authorization": {"Bearer " + gatewayToken}}, false},
		{http.Header{"authorization": {"bEaReR " + gatewayToken}}, true},
		{http.Header{"aUtHoRiZaTiOn": {"Bearer " + gatewayToken}}, true},
	}
	for _, value := range []string{"Bearer", "Bearer ", gatewayToken, "Basic " + gatewayToken, " Bearer " + gatewayToken, "Bearer  " + gatewayToken, "Bearer\t" + gatewayToken, "Bearer " + gatewayToken + " ", "Bearer " + gatewayToken + ", Bearer other", "Bearer " + gatewayToken + "\n", "Bearer " + gatewayToken + "\r", "Bearer " + gatewayToken + "\x00", "Bearer café", "Bearer " + strings.Repeat("x", 4097)} {
		cases = append(cases, struct {
			header http.Header
			valid  bool
		}{http.Header{"Authorization": {value}}, false})
	}
	for _, path := range []string{"/v1/models", "/providers/primary/v1/models", "/v1/chat/completions", "/providers/primary/v1/chat/completions"} {
		for _, tc := range cases {
			calls := 0
			guard, err := RequireGateway(testPolicy(t), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) }))
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, path+"?access_token="+gatewayToken, nil)
			r.Header = tc.header.Clone()
			if r.Header == nil {
				r.Header = make(http.Header)
			}
			r.Header.Set("Cookie", "token="+gatewayToken)
			r.Header.Set("X-API-Key", gatewayToken)
			r.Header.Set("X-Role", "admin")
			r.Header.Set("Proxy-Authorization", "Bearer "+gatewayToken)
			w := httptest.NewRecorder()
			guard.ServeHTTP(w, r)
			want := 401
			step := 0
			if tc.valid {
				want = 204
				step = 1
			}
			if w.Code != want || calls != step {
				t.Fatal("kebijakan header ambigu/alternatif salah")
			}
		}
	}
}
func TestOpaqueBearerParsing(t *testing.T) {
	for _, token := range []string{"opaque,token:with\"quote\\and+slash/=", strings.Repeat("x", 4096)} {
		got, ok := bearerToken(http.Header{"Authorization": {"Bearer " + token}})
		if !ok || got != token {
			t.Fatal("token opaque/cap valid ditolak atau dinormalisasi")
		}
	}
}

func TestAdminRoleGuard(t *testing.T) {
	for _, path := range []string{"/admin/status", "/metrics", "/admin/relay/disable"} {
		calls := 0
		guard, err := RequireAdmin(testPolicy(t), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) }))
		if err != nil {
			t.Fatal("guard admin valid ditolak", err)
		}
		for _, token := range []string{"", gatewayToken, upstreamKey, "wrong", adminToken} {
			r := httptest.NewRequest(http.MethodPost, path+"?admin_token="+adminToken, nil)
			r.Header.Set("Cookie", "admin="+adminToken)
			r.Header.Set("X-Role", "admin")
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			before := calls
			w := httptest.NewRecorder()
			guard.ServeHTTP(w, r)
			want := 401
			step := 0
			if token == adminToken {
				want = 204
				step = 1
			}
			if w.Code != want || calls-before != step {
				t.Fatal("akses role admin salah")
			}
		}
	}
	for _, constructor := range []func(*access.Tokens, http.Handler) (http.Handler, error){RequireGateway, RequireAdmin} {
		var nilFunction http.HandlerFunc
		for _, next := range []http.Handler{nil, nilFunction} {
			if h, err := constructor(testPolicy(t), next); err == nil || h != nil {
				t.Fatal("handler nil diterima constructor")
			}
		}
	}
}

type unreadBody struct{ reads, closes int }

func (b *unreadBody) Read([]byte) (int, error) { b.reads++; panic("body tidak boleh dibaca guard") }
func (b *unreadBody) Close() error             { b.closes++; return nil }
func TestAuthenticationErrors(t *testing.T) {
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(old) })
	ids := map[string]bool{}
	for _, admin := range []bool{false, true} {
		var guard http.Handler
		var err error
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("dispatch terjadi pada auth invalid") })
		if admin {
			guard, err = RequireAdmin(testPolicy(t), next)
		} else {
			guard, err = RequireGateway(testPolicy(t), next)
		}
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?access_token="+gatewayToken, nil)
			body := &unreadBody{}
			r.Body = body
			r.Header.Set("Authorization", "Bearer wrong-"+gatewayToken)
			r.Header.Set("X-Request-ID", gatewayToken)
			r.Header.Set("X-Infinity-Session", upstreamKey)
			w := httptest.NewRecorder()
			guard.ServeHTTP(w, r)
			var payload struct {
				Error     struct{ Message, Type, Code string }
				RequestID string `json:"request_id"`
			}
			if json.Unmarshal(w.Body.Bytes(), &payload) != nil {
				t.Fatal("envelope autentikasi tidak valid")
			}
			code := "invalid_gateway_token"
			realm := "gateway"
			if admin {
				code = "invalid_admin_token"
				realm = "admin"
			}
			if w.Code != 401 || payload.Error.Code != code || payload.Error.Type != "authentication_error" || payload.Error.Message != "Token tidak valid." {
				t.Fatal("status/envelope autentikasi salah")
			}
			if len(payload.RequestID) != 36 || !strings.HasPrefix(payload.RequestID, "req_") || ids[payload.RequestID] {
				t.Fatal("request ID tidak dibuat server")
			}
			if _, err := hex.DecodeString(strings.TrimPrefix(payload.RequestID, "req_")); err != nil {
				t.Fatal("request ID bukan hex")
			}
			ids[payload.RequestID] = true
			if w.Header().Get("X-Request-ID") != payload.RequestID || w.Header().Get("WWW-Authenticate") != `Bearer realm="`+realm+`"` || w.Header().Get("X-Infinity-Error-Source") != "gateway" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("header error tidak lengkap")
			}
			if body.reads != 0 || body.closes != 0 {
				t.Fatal("guard mengambil ownership body")
			}
			for _, v := range []string{gatewayToken, adminToken, upstreamKey} {
				if strings.Contains(w.Body.String(), v) || strings.Contains(logs.String(), v) {
					t.Fatal("credential bocor pada response/log")
				}
			}
		}
	}
}

func TestOpaqueHeaderWithConfiguredToken(t *testing.T) {
	for _, token := range []string{`synthetic,opaque:with"quote\and+/=`, strings.Repeat("x", 4096)} {
		guard, err := RequireGateway(policyForToken(t, token), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
		if err != nil {
			t.Fatal(err)
		}
		in := httptest.NewRequest("POST", "/v1/chat/completions", nil)
		in.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		guard.ServeHTTP(w, in)
		if w.Code != 204 {
			t.Fatal("token opaque/cap konfigurasi tidak bisa dipakai")
		}
	}
}
func FuzzBearerHeader(f *testing.F) {
	for _, value := range []string{"", "Bearer opaque", "bEaReR opaque,token", "Bearer one, Bearer two", "Bearer\tvalue"} {
		f.Add(value, false)
	}
	f.Fuzz(func(t *testing.T, value string, duplicate bool) {
		header := http.Header{"Authorization": {value}}
		if duplicate {
			header["authorization"] = []string{value}
		}
		token, ok := bearerToken(header)
		if ok {
			if duplicate || token == "" || len(token) > 4096 {
				t.Fatal("parser menerima bentuk ambigu/oversize")
			}
			for _, b := range []byte(token) {
				if b < 33 || b > 126 {
					t.Fatal("parser menerima whitespace/control")
				}
			}
		}
	})
}

func TestConcurrentGuards(t *testing.T) {
	var calls atomic.Int32
	h, err := RequireGateway(testPolicy(t), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				in := httptest.NewRequest("POST", "/v1/chat/completions", nil)
				token := gatewayToken
				want := 204
				if j%2 == 0 {
					token = adminToken
					want = 401
				}
				in.Header.Set("Authorization", "Bearer "+token)
				out := httptest.NewRecorder()
				h.ServeHTTP(out, in)
				if out.Code != want {
					t.Error("guard concurrent salah")
					return
				}
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 200 {
		t.Fatal("jumlah admission concurrent salah")
	}
}
