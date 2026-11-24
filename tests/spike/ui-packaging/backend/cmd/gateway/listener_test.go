package main

import (
	"errors"
	"net"
	"sync"
	"testing"
)

type pipeListener struct{ peers []net.Conn }

func (l *pipeListener) Accept() (net.Conn, error) {
	a, b := net.Pipe()
	l.peers = append(l.peers, b)
	return a, nil
}
func (l *pipeListener) Close() error {
	for _, p := range l.peers {
		_ = p.Close()
	}
	return nil
}
func (l *pipeListener) Addr() net.Addr { return nil }

func TestConnectionBoundAndRelease(t *testing.T) {
	base := &pipeListener{}
	t.Cleanup(func() { _ = base.Close() })
	l := newBoundedListener(base, 1)
	first, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	second, err := l.Accept()
	if err == nil {
		_ = second.Close()
		t.Fatal("connection_limit_not_enforced")
	}
	var temporary net.Error
	if !errors.As(err, &temporary) || !temporary.Temporary() {
		t.Fatal("overload_must_not_stop_server")
	}
	var closes sync.WaitGroup
	for i := 0; i < 10; i++ {
		closes.Add(1)
		go func() { defer closes.Done(); _ = first.Close() }()
	}
	closes.Wait()
	third, err := l.Accept()
	if err != nil {
		t.Fatal("slot_not_released")
	}
	_ = third.Close()
}
