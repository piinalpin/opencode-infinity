package main

import (
	"net"
	"sync"
)

type boundedListener struct {
	net.Listener
	slots chan struct{}
}
type boundedConn struct {
	net.Conn
	once  sync.Once
	slots chan struct{}
}
type overload struct{}

func (overload) Error() string   { return "fixture_connection_limit" }
func (overload) Temporary() bool { return true }
func (overload) Timeout() bool   { return false }

func newBoundedListener(base net.Listener, capacity int) net.Listener {
	return &boundedListener{Listener: base, slots: make(chan struct{}, capacity)}
}
func (l *boundedListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	select {
	case l.slots <- struct{}{}:
		return &boundedConn{Conn: c, slots: l.slots}, nil
	default:
		_ = c.Close()
		return nil, overload{}
	}
}
func (c *boundedConn) Close() error {
	var err error
	c.once.Do(func() { err = c.Conn.Close(); <-c.slots })
	return err
}
