package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Secret menyimpan nilai yang hanya boleh dibuka secara eksplisit oleh komponen server.
// Closure menyembunyikan representasi nilai saat fmt melewati formatter pada field privat pembungkus.
// Ini bukan perlindungan terhadap kode tepercaya yang memanggil Reveal atau inspeksi memory.
type Secret struct{ reveal func() string }

func (s Secret) Reveal() string {
	if s.reveal == nil {
		return ""
	}
	return s.reveal()
}
func (c *Config) GatewayToken() Secret            { return c.gateway }
func (c *Config) AdminToken() Secret              { return c.admin }
func (c *Config) APIKey(id string) (Secret, bool) { v, ok := c.keys[id]; return v, ok }
func environmentName(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b == '_' || i > 0 && b >= '0' && b <= '9' {
			continue
		}
		return false
	}
	return true
}
func placeholder(s string) bool {
	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") || strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		return true
	}
	switch strings.ToLower(s) {
	case "changeme", "change-me", "change_me", "replace-me", "replace_me", "your-api-key", "your-token", "gateway-token", "admin-token", "upstream-key", "example", "placeholder", "todo", "tbd":
		return true
	}
	return false
}
func resolveSecret(env, file string, min int, s Sources) (Secret, error) {
	if (env == "") == (file == "") {
		return Secret{}, errors.New("config_invalid_secret_source")
	}
	var value string
	if file != "" {
		if !filepath.IsAbs(file) {
			return Secret{}, errors.New("config_invalid_secret_source")
		}
		open := s.OpenFile
		if open == nil {
			open = openSecretFile
		}
		f, err := open(file)
		if err != nil {
			return Secret{}, errors.New("config_secret_unavailable")
		}
		data, readErr := io.ReadAll(io.LimitReader(f, 4099))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil {
			return Secret{}, errors.New("config_secret_unavailable")
		}
		value = string(data)
		if strings.HasSuffix(value, "\r\n") {
			value = strings.TrimSuffix(value, "\r\n")
		} else {
			value = strings.TrimSuffix(value, "\n")
		}
	} else {
		if !environmentName(env) || s.LookupEnv == nil {
			return Secret{}, errors.New("config_invalid_secret_source")
		}
		var ok bool
		value, ok = s.LookupEnv(env)
		if !ok {
			return Secret{}, errors.New("config_secret_unavailable")
		}
	}
	if !printable(value, 4096) || len(value) < min || placeholder(value) {
		return Secret{}, errors.New("config_invalid_secret")
	}
	return Secret{reveal: func() string { return value }}, nil
}
func loadSecrets(d *document, c *Config, s Sources) error {
	var err error
	c.gateway, err = resolveSecret(d.Server.GatewayEnv, d.Server.GatewayFile, 24, s)
	if err != nil {
		return err
	}
	c.admin, err = resolveSecret(d.Server.AdminEnv, d.Server.AdminFile, 24, s)
	if err != nil {
		return err
	}
	if c.gateway.Reveal() == c.admin.Reveal() {
		return errors.New("config_secret_roles_overlap")
	}
	c.keys = make(map[string]Secret, len(d.Upstreams))
	for _, u := range d.Upstreams {
		key, err := resolveSecret(u.KeyEnv, u.KeyFile, 1, s)
		if err != nil {
			return err
		}
		if key.Reveal() == c.gateway.Reveal() || key.Reveal() == c.admin.Reveal() {
			return errors.New("config_secret_roles_overlap")
		}
		c.keys[u.ID] = key
	}
	return nil
}

// O_NONBLOCK menjaga penggantian file menjadi FIFO tidak menggantung startup pada host Linux.
func openSecretFile(path string) (io.ReadCloser, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("config_secret_unavailable")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("config_secret_unavailable")
	}
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("config_secret_unavailable")
	}
	return f, nil
}

func (s Secret) Format(state fmt.State, verb rune) { io.WriteString(state, "[REDACTED]") }
func (s Secret) MarshalJSON() ([]byte, error)      { return []byte(`"[REDACTED]"`), nil }
func (s Secret) MarshalText() ([]byte, error)      { return []byte("[REDACTED]"), nil }
func (s Secret) MarshalYAML() (any, error)         { return "[REDACTED]", nil }

// Format melindungi juga field privat saat fmt menelusuri keseluruhan Config.
func (c Config) Format(state fmt.State, verb rune) { io.WriteString(state, "config.Config{redacted}") }
