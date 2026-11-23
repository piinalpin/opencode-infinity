package warp_request_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"opencode-infinity/internal/testutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type nativeResult struct {
	Exit, OutputBytes, ErrorBytes, Events, TextEvents int
	Code                                              string
	ConfigSeen                                        bool
	ProbeExit                                         int
}

// Native stdout/stderr dikonsumsi terbatas; tidak disimpan atau diteruskan ke artefak.
type nativeCollector struct {
	mu                  sync.Mutex
	bytes, events, text int
	tail                []byte
	hints               map[string]bool
	cancel              context.CancelFunc
	stdout              bool
}

func (n *nativeCollector) Write(p []byte) (int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.bytes += len(p)
	if n.bytes > maxResponse {
		n.cancel()
		return 0, errors.New("native_output_limit")
	}
	if !n.stdout {
		if len(n.tail)+len(p) <= 64*1024 {
			n.tail = append(n.tail, p...)
		}
		text := strings.ToLower(string(n.tail))
		for _, hint := range []string{"config", "invalid", "provider", "apikey", "model", "baseurl", "package", "permission", "network", "connection", "error", "decode", "not found", "token", "home", "enoent", "eacces", "erofs", "path", "missing", "validation", "parse", "failure", "unknown", "notfound", "uid", "agent", "fixture", "disabled", "fetch", "failed", "git", "workspace", "project", "directory", "auth", "key", "api", "required", "missing", "bytes", "expected", "string", "encrypt", "unauthorized", "denied", "provided", "credential", "get", "resolve", "loading"} {
			if strings.Contains(text, hint) {
				n.hints[hint] = true
			}
		}
		return len(p), nil
	}
	for _, b := range p {
		n.tail = append(n.tail, b)
		if len(n.tail) > 64*1024 {
			n.cancel()
			return 0, errors.New("native_line_limit")
		}
		if b != '\n' {
			continue
		}
		var event struct{ Type string }
		if json.Unmarshal(bytes.TrimSpace(n.tail), &event) == nil {
			if event.Type == "error" {
				text := strings.ToLower(string(n.tail))
				for _, hint := range []string{"agent", "fixture", "model", "provider", "notfound", "not found", "disabled", "permission", "invalid", "config", "connection", "fetch", "failed", "home", "enoent", "schema", "git", "workspace", "project", "directory", "auth", "key", "unknown", "api", "required", "missing", "bytes", "expected", "string", "encrypt", "unauthorized", "denied", "provided", "credential", "get", "resolve", "loading"} {
					if strings.Contains(text, hint) {
						n.hints[hint] = true
					}
				}
			}
			n.events++
			if event.Type == "text" || event.Type == "reasoning" {
				n.text++
			}
		}
		n.tail = n.tail[:0]
	}
	return len(p), nil
}
func runNative(parent context.Context, binary, root, base, nonce string) nativeResult {
	if os.MkdirAll("/tmp/t04-native", 0700) != nil {
		return nativeResult{Exit: -1, Code: "native_home_directory"}
	}
	for _, d := range []string{"config/opencode", "data", "state", "cache", "work"} {
		if os.MkdirAll(filepath.Join(root, d), 0700) != nil {
			return nativeResult{Exit: -1, Code: "native_directory"}
		}
	}
	encoded, err := json.Marshal(nativeConfiguration(base, nonce))
	if err != nil {
		return nativeResult{Exit: -1, Code: "native_config"}
	}
	if os.WriteFile(filepath.Join(root, "config/opencode/opencode.json"), encoded, 0600) != nil {
		return nativeResult{Exit: -1, Code: "native_config_write"}
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "run", "--standalone", "--log-level", "none", "--title", "T04-fixture", "--agent", "fixture", "--model", "opencode/"+selectedModel, "--format", "json", "Print integers one through sixty separated by spaces. No tools.")
	cmd.Dir = filepath.Join(root, "work")
	cmd.Env = []string{"PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C.UTF-8", "OPENCODE_DB=:memory:", "XDG_CONFIG_HOME=" + filepath.Join(root, "config"), "XDG_DATA_HOME=" + filepath.Join(root, "data"), "XDG_STATE_HOME=" + filepath.Join(root, "state"), "XDG_CACHE_HOME=" + filepath.Join(root, "cache")}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	cmd.WaitDelay = 2 * time.Second
	out := &nativeCollector{stdout: true, cancel: cancel, hints: map[string]bool{}}
	stderr := &nativeCollector{cancel: cancel, hints: map[string]bool{}}
	cmd.Stdout = out
	cmd.Stderr = stderr
	probe := exec.CommandContext(ctx, binary, "debug", "config")
	probe.Env = cmd.Env
	probe.Dir = cmd.Dir
	probeOutput := &nativeCollector{cancel: cancel, hints: map[string]bool{}}
	probe.Stdout = probeOutput
	probe.Stderr = &nativeCollector{cancel: cancel, hints: map[string]bool{}}
	probeErr := probe.Run()
	probeCode := 0
	if probeErr != nil {
		probeCode = -1
	}
	seen := bytes.Contains(probeOutput.tail, []byte(filepath.Join(root, "config/opencode/opencode.json")))
	probeOutput.tail = nil
	err = cmd.Run()
	result := nativeResult{ConfigSeen: seen, ProbeExit: probeCode, Exit: 0, OutputBytes: out.bytes, ErrorBytes: stderr.bytes, Events: out.events, TextEvents: out.text}
	if err != nil {
		result.Exit = -1
		result.Code = "native_exit"
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.Exit = exit.ExitCode()
		}
	}
	if ctx.Err() != nil {
		result.Code = "native_timeout"
	}
	plain := strings.ToLower(regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(string(stderr.tail), ""))
	for _, hint := range []string{"config", "invalid", "provider", "apikey", "model", "baseurl", "package", "permission", "network", "connection", "error", "decode", "not found", "token", "home", "enoent", "eacces", "erofs", "path", "missing", "validation", "parse", "failure", "unknown", "notfound", "uid", "agent", "fixture", "disabled", "fetch", "failed", "git", "workspace", "project", "directory", "auth", "key", "api", "required", "missing", "bytes", "expected", "string", "encrypt", "unauthorized", "denied", "provided", "credential", "get", "resolve", "loading"} {
		if strings.Contains(plain, hint) || out.hints[hint] {
			result.Code += "_" + strings.ReplaceAll(hint, " ", "-")
		}
	}
	stderr.tail = nil

	// Semua field laporan adalah hitungan/kategori; buang buffer parsing terakhir.
	out.tail = nil
	return result
}

func TestNativeFixture(t *testing.T) {
	binary := os.Getenv("T04_NATIVE_BIN")
	if binary == "" {
		t.Skip("opt-in native local smoke; no provider")
	}
	_, _, c, s := setupControl(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_ = testutil.WriteFragments(r.Context(), w, []byte("data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":\""+selectedModel+"\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"fixture\"},\"finish_reason\":null}]}\n\n"+terminalFrames), []int{7, 3})
	})
	result := runNative(t.Context(), binary, t.TempDir(), s.URL, "fixture-nonce")
	if result.Exit != 0 || result.TextEvents == 0 || c.trace.snapshot().Admitted != 1 || !c.trace.snapshot().EOF {
		t.Fatalf("native local smoke: result=%+v observation=%+v", result, c.trace.snapshot())
	}
}

func nativeConfiguration(base, nonce string) map[string]any {
	policies := []any{}
	for _, action := range []string{"provider.use", "tool.use", "integration.connect", "integration.use", "mcp.use", "skill.use", "websearch.use"} {
		policies = append(policies, map[string]any{"action": action, "resource": "*", "effect": "deny"})
	}
	// OpenCode 2.3 menerapkan juga integration.use pada provider tanpa credential.
	for _, action := range []string{"provider.use", "integration.use"} {
		policies = append(policies, map[string]any{"action": action, "resource": "opencode", "effect": "allow"})
	}
	return map[string]any{
		"model": "opencode/" + selectedModel, "default_agent": "fixture", "update": "disable", "share": "disabled", "snapshots": false, "warming": false, "automations": false,
		"compaction": map[string]any{"auto": false, "background": false}, "plugins": []any{}, "policies": policies,
		"agents": map[string]any{"fixture": map[string]any{"mode": "primary", "description": "Synthetic T04 transport check", "system": "Respond in plain text only. Do not use tools or inspect files.", "steps": 1, "permissions": []any{map[string]any{"action": "*", "resource": "*", "effect": "deny"}}}},
		"providers": map[string]any{"opencode": map[string]any{
			"name": "OpenCode Free", "package": "@opencode/ai/providers/openai-compatible",
			"settings": map[string]any{"baseURL": base + "/inference/openai/v1", "timeout": 90000, "headerTimeout": 30000, "chunkTimeout": 15000},
			"headers":  map[string]any{"X-T04-Run": nonce, "x-opencode-client": "desktop"},
			"body":     map[string]any{"max_tokens": 512, "stream": true},
			"models":   map[string]any{selectedModel: map[string]any{"modelID": selectedModel, "name": selectedModel, "type": "chat", "compatibility": map[string]any{"maxTokensField": "max_tokens"}, "limit": map[string]any{"context": 32768, "output": 512}, "capabilities": map[string]any{"tools": false}}},
		}},
	}
}

func TestNativeConfigurationIsolation(t *testing.T) {
	c := nativeConfiguration("http://127.0.0.1:1234", "fixture-nonce")
	encoded, _ := json.Marshal(c)
	if c["model"] != "opencode/"+selectedModel || c["update"] != "disable" || c["warming"] != false || c["snapshots"] != false {
		t.Fatalf("native isolation not configured: %s", encoded)
	}
	p, ok := c["providers"].(map[string]any)
	if !ok {
		t.Fatal("native provider belum ada")
	}
	op := p["opencode"].(map[string]any)
	if op["package"] != "@opencode/ai/providers/openai-compatible" {
		t.Fatal("package native tidak sesuai")
	}
}
