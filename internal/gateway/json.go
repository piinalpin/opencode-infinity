// Package gateway mengoordinasikan katalog dan generasi dari snapshot server.
package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

type chatJSON struct {
	alias      string
	start, end int
}

// Validasi sintaks stdlib mendahului pencarian offset. Tidak membuat tree/map JSON.
func parseChat(body []byte) (chatJSON, error) {
	invalid := errors.New("invalid_json")
	if !utf8.Valid(body) || !json.Valid(body) {
		return chatJSON{}, invalid
	}
	first := skipSpace(body, 0)
	if body[first] != '{' {
		return chatJSON{}, invalid
	}
	var result chatJSON
	seenModel, seenStream := false, false
	for i := skipSpace(body, first+1); body[i] != '}'; {
		keyEnd := stringEnd(body, i)
		var key string
		// Nama kontrol maksimal enam karakter; escape terpanjang enam byte per karakter.
		if keyEnd-i <= 38 {
			_ = json.Unmarshal(body[i:keyEnd], &key)
		}
		start := skipSpace(body, skipSpace(body, keyEnd)+1)
		end := valueEnd(body, start)
		if key == "model" {
			if seenModel {
				return chatJSON{}, invalid
			}
			seenModel = true
			value := bytes.TrimSpace(body[start:end])
			if len(value) > 770 || len(value) == 0 || value[0] != '"' || json.Unmarshal(value, &result.alias) != nil || result.alias == "" || len(result.alias) > 128 {
				return chatJSON{}, invalid
			}
			result.start, result.end = start, end
		}
		if key == "stream" {
			if seenStream {
				return chatJSON{}, invalid
			}
			seenStream = true
			value := bytes.TrimSpace(body[start:end])
			if bytes.Equal(value, []byte("true")) {
				return chatJSON{}, errors.New("unsupported_api")
			}
			if !bytes.Equal(value, []byte("false")) {
				return chatJSON{}, invalid
			}
		}
		i = skipSpace(body, end)
		if body[i] == ',' {
			i = skipSpace(body, i+1)
		}
	}
	if result.alias == "" {
		return chatJSON{}, invalid
	}
	return result, nil
}

func (p chatJSON) remap(body, replacement []byte) []byte {
	out := make([]byte, len(body)-(p.end-p.start)+len(replacement))
	n := copy(out, body[:p.start])
	n += copy(out[n:], replacement)
	copy(out[n:], body[p.end:])
	return out
}

func skipSpace(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return i
}
func stringEnd(b []byte, i int) int {
	for i++; i < len(b); i++ {
		if b[i] == '\\' {
			i++
			continue
		}
		if b[i] == '"' {
			return i + 1
		}
	}
	return len(b)
}

// Dipanggil hanya setelah json.Valid; delimiter/string sudah tervalidasi.
func valueEnd(b []byte, i int) int {
	if b[i] == '"' {
		return stringEnd(b, i)
	}
	depth := 0
	for ; i < len(b); i++ {
		switch b[i] {
		case '"':
			i = stringEnd(b, i) - 1
		case '{', '[':
			depth++
		case '}', ']':
			if depth == 0 {
				return i
			}
			depth--
			if depth == 0 {
				return i + 1
			}
		case ',':
			if depth == 0 {
				return i
			}
		}
	}
	return i
}
