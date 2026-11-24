package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSecretSources(t *testing.T) {
	for _, role := range []struct{ env, field string }{{"GATEWAY_TOKEN", "gateway_token"}, {"ADMIN_TOKEN", "admin_token"}, {"UPSTREAM_KEY", "api_key"}} {
		for _, value := range []string{"", " ", "changeme", "CHANGE-ME", "replace_me", "example", "your-api-key", "${UNRESOLVED_CREDENTIAL}", "<operator-must-replace-this>", "bad\nvalue", strings.Repeat("x", 4097)} {
			t.Run(role.env+"/invalid/"+fmt.Sprint(len(value)), func(t *testing.T) {
				src := testSources()
				lookup := src.LookupEnv
				src.LookupEnv = func(name string) (string, bool) {
					if name == role.env {
						return value, true
					}
					return lookup(name)
				}
				if c, err := Load(strings.NewReader(validYAML), src); err == nil || c != nil {
					t.Fatal("secret invalid diterima")
				}
			})
		}
		src := testSources()
		lookup := src.LookupEnv
		src.LookupEnv = func(name string) (string, bool) {
			if name == role.env {
				return "", false
			}
			return lookup(name)
		}
		if c, err := Load(strings.NewReader(validYAML), src); err == nil || c != nil {
			t.Fatal("secret absent diterima")
		}
		field := role.field + "_env: " + role.env
		indent := "  "
		if role.field == "api_key" {
			indent = "    "
		}
		both := strings.Replace(validYAML, field, role.field+"_file: /tmp/unavailable-secret\n"+indent+field, 1)
		if c, err := loadTest(both); err == nil || c != nil || err.Error() != "config_invalid_secret_source" {
			t.Fatal("sumber ganda tidak ditolak oleh validasi secret")
		}

		rejectConfig(t, strings.Replace(validYAML, field, "", 1))
	}
	if c, err := Load(strings.NewReader(validYAML), Sources{}); err == nil || c != nil {
		t.Fatal("lookup nil memakai environment host")
	}
	for _, pair := range [][2]string{{"ADMIN_TOKEN", "GATEWAY_TOKEN"}, {"UPSTREAM_KEY", "GATEWAY_TOKEN"}, {"UPSTREAM_KEY", "ADMIN_TOKEN"}, {"GATEWAY_TOKEN", "INVALID/ENV"}} {
		rejectConfig(t, strings.Replace(validYAML, pair[0], pair[1], 1))
	}
	src := testSources()
	lookup := src.LookupEnv
	src.LookupEnv = func(name string) (string, bool) {
		if name == "UPSTREAM_KEY" {
			return "x", true
		}
		return lookup(name)
	}
	if _, err := Load(strings.NewReader(validYAML), src); err != nil {
		t.Fatal("API key nonempty singkat ditolak", err)
	}
	if _, err := Load(strings.NewReader(withSecondUpstream(validYAML, "secondary")), testSources()); err != nil {
		t.Fatal("reuse key antar-upstream ditolak", err)
	}
}

func TestSecretFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "secret")
	for _, role := range []struct{ env, field string }{{"GATEWAY_TOKEN", "gateway_token"}, {"ADMIN_TOKEN", "admin_token"}, {"UPSTREAM_KEY", "api_key"}} {
		for _, ending := range []string{"", "\n", "\r\n"} {
			value := "synthetic-file-credential-12345678" + ending
			if err := os.WriteFile(file, []byte(value), 0600); err != nil {
				t.Fatal(err)
			}
			cfg := strings.Replace(validYAML, role.field+"_env: "+role.env, role.field+"_file: "+file, 1)
			c, err := loadTest(cfg)
			if err != nil || c == nil {
				t.Fatal("file secret valid ditolak", err)
			}
		}
	}
	cfg := strings.Replace(validYAML, "gateway_token_env: GATEWAY_TOKEN", "gateway_token_file: "+file, 1)
	for _, v := range []string{"", "changeme", "synthetic-file-credential-12345678\n\n", "synthetic-file-credential-12345678\r", strings.Repeat("x", 4097)} {
		if err := os.WriteFile(file, []byte(v), 0600); err != nil {
			t.Fatal(err)
		}
		rejectConfig(t, cfg)
	}
	if err := os.WriteFile(file, []byte(strings.Repeat("x", 4096)+"\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTest(cfg); err != nil {
		t.Fatal("secret tepat cap+CRLF ditolak", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTest(strings.Replace(cfg, file, link, 1)); err != nil {
		t.Fatal("symlink ke file reguler ditolak", err)
	}
	for _, path := range []string{filepath.Join(dir, "missing"), dir, "relative-secret"} {
		rejectConfig(t, strings.Replace(cfg, file, path, 1))
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	rejectConfig(t, strings.Replace(cfg, file, fifo, 1))
	src := testSources()
	src.OpenFile = func(string) (io.ReadCloser, error) { return nil, errors.New("sentinel-private-os-error") }
	if c, err := Load(strings.NewReader(cfg), src); err == nil || c != nil || strings.Contains(err.Error(), "sentinel-private") {
		t.Fatal("error file bocor/diterima")
	}
	src.OpenFile = func(string) (io.ReadCloser, error) { return io.NopCloser(errorReader{}), nil }
	if c, err := Load(strings.NewReader(cfg), src); err == nil || c != nil {
		t.Fatal("read failure diterima")
	}
	r := strings.NewReader(strings.Repeat("x", 5000))
	src.OpenFile = func(string) (io.ReadCloser, error) { return io.NopCloser(r), nil }
	if c, err := Load(strings.NewReader(cfg), src); err == nil || c != nil {
		t.Fatal("oversize reader diterima")
	}
	if r.Len() != 5000-4099 {
		t.Fatal("pembacaan file tidak dibatasi cap+newline+1")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("sentinel-private-reader-error") }
