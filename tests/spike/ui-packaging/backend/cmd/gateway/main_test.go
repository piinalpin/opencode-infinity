package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretFileBound(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"empty", "", false}, {"large", strings.Repeat("x", 257), false},
		{"newline", strings.Repeat("x", 24) + "\n", false},
		{"valid", strings.Repeat("x", 32), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "admin")
			if err := os.WriteFile(path, []byte(tc.value), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := loadSecret(path)
			if (err == nil) != tc.valid {
				t.Fatal("secret_validation_mismatch")
			}
			if tc.valid && string(got) != tc.value {
				t.Fatal("secret_file_changed")
			}
		})
	}
	if _, err := loadSecret(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing_secret_accepted")
	}
}
