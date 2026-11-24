package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opencode-infinity/packaging-fixture/internal/webui"
)

func loadSecret(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("admin_secret_unavailable")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 257))
	if err != nil || len(data) < 24 || len(data) > 256 {
		return nil, errors.New("invalid_admin_secret")
	}
	for _, b := range data {
		if b < 33 || b > 126 {
			return nil, errors.New("invalid_admin_secret")
		}
	}
	return data, nil
}

func run() error {
	secret, err := loadSecret(os.Getenv("T05_ADMIN_SECRET_FILE"))
	if err != nil {
		return err
	}
	handler, err := webui.EmbeddedHandler(secret)
	if err != nil {
		return errors.New("invalid_ui_assets")
	}
	address := os.Getenv("T05_LISTEN")
	if address == "" {
		address = "127.0.0.1:8788"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New("listener_unavailable")
	}
	defer listener.Close()
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		deadline, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(deadline)
	}()
	fmt.Printf("fixture_listen %s\n", listener.Addr())
	// shortcut: delapan koneksi untuk spike statis; batas aplikasi ditetapkan pada M0/EP-02.
	err = server.Serve(newBoundedListener(listener, 8))
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return errors.New("server_failed")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(78)
	}
}
