// Package testutil menyediakan fixture lokal; hanya diimpor dari berkas test.
package testutil

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

const MaxConnections = 16
const MaxRecords = 256
const FixtureTimeout = 5 * time.Second

type Connect struct {
	Relay, Hostname, Port, TargetPeer string
	Refused                           bool
}
type ProxySnapshot struct {
	Accepted, Active int
	Connects         []Connect
}

// SOCKS5 hanya mengizinkan satu authority yang dipetakan ke listener loopback.
// shortcut: SOCKS5 CONNECT domain/no-auth; tambah mode lain saat integrasi memerlukannya.
type SOCKS5 struct {
	listener                 net.Listener
	relay, authority, target string
	refuse                   bool
	mu                       sync.Mutex
	accepted                 int
	closed                   bool
	live                     map[net.Conn]struct{}
	connects                 []Connect
	workers                  sync.WaitGroup
	closeOnce                sync.Once
	done                     chan struct{}
}

func NewSOCKS5(relay, authority, target string, refuse bool) (*SOCKS5, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("target fixture harus alamat IP loopback")
	}
	if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
		return nil, errors.New("port target tidak valid")
	}
	if h, p, err := net.SplitHostPort(authority); err != nil || h == "" || p != port {
		return nil, errors.New("authority fixture tidak valid")
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &SOCKS5{listener: l, relay: relay, authority: authority, target: target, refuse: refuse, live: make(map[net.Conn]struct{}), done: make(chan struct{})}
	s.workers.Add(1)
	go s.accept()
	return s, nil
}
func (s *SOCKS5) Addr() string { return s.listener.Addr().String() }
func (s *SOCKS5) Snapshot() ProxySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ProxySnapshot{Accepted: s.accepted, Active: len(s.live), Connects: append([]Connect(nil), s.connects...)}
}
func (s *SOCKS5) accept() {
	defer s.workers.Done()
	for {
		c, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed || len(s.live) >= MaxConnections || s.accepted >= MaxRecords {
			s.mu.Unlock()
			c.Close()
			continue
		}
		s.accepted++
		s.live[c] = struct{}{}
		s.workers.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.workers.Done()
			defer func() { c.Close(); s.mu.Lock(); delete(s.live, c); s.mu.Unlock() }()
			s.serve(c)
		}()
	}
}
func (s *SOCKS5) serve(c net.Conn) {
	_ = c.SetDeadline(time.Now().Add(FixtureTimeout))
	var greeting [2]byte
	if _, err := io.ReadFull(c, greeting[:]); err != nil || greeting[0] != 5 || greeting[1] == 0 {
		return
	}
	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(c, methods); err != nil {
		return
	}
	noauth := false
	for _, m := range methods {
		if m == 0 {
			noauth = true
		}
	}
	if !noauth {
		_, _ = c.Write([]byte{5, 255})
		return
	}
	if _, err := c.Write([]byte{5, 0}); err != nil {
		return
	}
	var head [5]byte
	if _, err := io.ReadFull(c, head[:]); err != nil || head[0] != 5 || head[1] != 1 || head[2] != 0 || head[3] != 3 || head[4] == 0 {
		return
	}
	addr := make([]byte, int(head[4])+2)
	if _, err := io.ReadFull(c, addr); err != nil {
		return
	}
	host := string(addr[:len(addr)-2])
	port := strconv.Itoa(int(binary.BigEndian.Uint16(addr[len(addr)-2:])))
	if net.JoinHostPort(host, port) != s.authority {
		_, _ = c.Write([]byte{5, 2, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	if s.refuse {
		s.mu.Lock()
		s.connects = append(s.connects, Connect{Relay: s.relay, Hostname: host, Port: port, Refused: true})
		s.mu.Unlock()
		_, _ = c.Write([]byte{5, 2, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	upstream, err := net.DialTimeout("tcp4", s.target, time.Second)
	if err != nil {
		_, _ = c.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer upstream.Close()
	_ = upstream.SetDeadline(time.Now().Add(FixtureTimeout))
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.connects = append(s.connects, Connect{Relay: s.relay, Hostname: host, Port: port, TargetPeer: upstream.LocalAddr().String()})
	s.mu.Unlock()
	if _, err := c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	copied := make(chan struct{})
	go func() {
		_, _ = io.CopyBuffer(upstream, c, make([]byte, 16*1024))
		upstream.Close()
		c.Close()
		close(copied)
	}()
	_, _ = io.CopyBuffer(c, upstream, make([]byte, 16*1024))
	c.Close()
	upstream.Close()
	<-copied
}
func (s *SOCKS5) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		_ = s.listener.Close()
		for c := range s.live {
			_ = c.Close()
		}
		s.mu.Unlock()
		go func() { s.workers.Wait(); close(s.done) }()
	})
	select {
	case <-s.done:
		return nil
	case <-time.After(FixtureTimeout):
		return fmt.Errorf("cleanup relay %s melewati deadline", s.relay)
	}
}
