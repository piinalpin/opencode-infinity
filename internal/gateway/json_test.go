package gateway

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestModelJSONPreservesBytes(t *testing.T) {
	input := []byte(` { "m\u006fdel" : "coding-default", "stream":false, "messages":[{"model":"nested","content":"日本語"}], "tools":[{"type":"function"}], "extra":{"n":9007199254740993,"huge":1e1000,"same":1,"same":2,"x":null} } `)
	parsed, err := parseChat(input)
	if err != nil || parsed.alias != "coding-default" {
		t.Fatal("alias valid tidak ditemukan")
	}
	output := parsed.remap(input, []byte(`"real-model"`))
	expected := bytes.Replace(input, []byte(`"coding-default"`), []byte(`"real-model"`), 1)
	if !bytes.Equal(output, expected) {
		t.Fatal("byte selain nilai model berubah")
	}
}

func TestGatewayStreamControl(t *testing.T) {
	for _, body := range []string{`{"model":"coding-default","stream":true,"extra":9007199254740993}`, `{"str\u0065am":true,"model":"coding-default","tools":[{"model":"nested"}]}`} {
		input := []byte(body)
		parsed, err := parseChat(input)
		if err != nil {
			t.Fatal("stream true valid ditolak")
		}
		want := bytes.Replace(input, []byte(`"coding-default"`), []byte(`"real-model"`), 1)
		if !bytes.Equal(parsed.remap(input, []byte(`"real-model"`)), want) {
			t.Fatal("field stream/tools berubah setelah mapping")
		}
	}
}

func FuzzModelJSON(f *testing.F) {
	for _, seed := range []string{`{"model":"coding-default"}`, `{"m\u006fdel":"x","array":[{},[],1e1000,"\\\""]}`, `{"model":"x","model":"y"}`, `{"model":"x","stream":true}`, `{}`, `null`, ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64*1024 {
			return
		}
		body := []byte(input)
		parsed, err := parseChat(body)
		if err != nil {
			return
		}
		mapped := parsed.remap(body, []byte(`"server-model"`))
		if !json.Valid(mapped) {
			t.Fatal("remap membuat JSON invalid")
		}
		next, err := parseChat(mapped)
		if err != nil || next.alias != "server-model" {
			t.Fatal("model root salah setelah remap")
		}
	})
}

func TestModelJSONRejectsAmbiguousControls(t *testing.T) {
	for _, input := range []string{
		`{"model":"one","model":"two"}`, `{"model":"one","m\u006fdel":"two"}`,
		`{"model":"one","stream":false,"str\u0065am":false}`,
		`{"model":"one","stream":true,"str\u0065am":false}`, `{"model":"one","stream":null}`,
		`{"model":"one","stream":"false"}`, `{"model":"one","stream":0}`,
		`null`, `[]`, `{}`, `{"model":null}`, `{"model":4}`, `{"model":""}`,
		`{"model":"one"} {}`, `{"model":"one",}`, `{"model":"one"`,
		`{"model":"` + strings.Repeat("a", 129) + `"}`, "{\"model\":\"one\",\"x\":\"\xff\"}",
	} {
		if _, err := parseChat([]byte(input)); err == nil {
			t.Fatal("JSON/control ambigu diterima")
		}
	}
	for _, input := range []string{`{"model":"one"}`, `{"model":"one","stream": false }`, `{"model":"one","x":{"stream":true,"model":42}}`} {
		if _, err := parseChat([]byte(input)); err != nil {
			t.Fatal("kontrol non-stream valid ditolak")
		}
	}
}
