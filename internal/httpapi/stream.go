package httpapi

import (
	"context"
	"io"
	"net/http"
	"opencode-infinity/internal/gateway"
	"sync"
	"time"
)

type streamForwarder struct {
	writer       http.ResponseWriter
	controller   *http.ResponseController
	context      context.Context
	idle         time.Duration
	end          time.Time
	mu           sync.Mutex
	stop         func() bool
	callbackDone chan struct{}
}

func supportsFlush(w http.ResponseWriter) bool {
	for {
		switch v := w.(type) {
		case interface{ FlushError() error }:
			return true
		case http.Flusher:
			return true
		case interface{ Unwrap() http.ResponseWriter }:
			w = v.Unwrap()
		default:
			return false
		}
	}
}

func (h modelHandler) prepareStream(w http.ResponseWriter, r *http.Request, started time.Time) (*streamForwarder, error) {
	if !supportsFlush(w) {
		return nil, http.ErrNotSupported
	}
	end := started.Add(h.total)
	if deadline, ok := r.Context().Deadline(); ok && deadline.Before(end) {
		end = deadline
	}
	s := &streamForwarder{writer: w, controller: http.NewResponseController(w), context: r.Context(), idle: h.idle, end: end, callbackDone: make(chan struct{})}
	if err := s.writeDeadline(); err != nil {
		return nil, err
	}
	s.stop = context.AfterFunc(r.Context(), func() {
		defer close(s.callbackDone)
		s.mu.Lock()
		defer s.mu.Unlock()
		_ = s.controller.SetWriteDeadline(time.Now())
	})
	return s, nil
}

func (s *streamForwarder) deadline(when time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.context.Err(); err != nil {
		return err
	}
	if !time.Now().Before(s.end) {
		return context.DeadlineExceeded
	}
	if when.After(s.end) {
		when = s.end
	}
	return s.controller.SetWriteDeadline(when)
}
func (s *streamForwarder) writeDeadline() error { return s.deadline(time.Now().Add(s.idle)) }
func (s *streamForwarder) close() {
	// Stop(false) berarti callback mungkin masih berjalan. Join sebelum koneksi dapat digunakan ulang.
	if !s.stop() {
		<-s.callbackDone
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.controller.SetWriteDeadline(time.Time{})
}

func (s *streamForwarder) copy(body io.Reader) error {
	if err := s.controller.Flush(); err != nil {
		return err
	}
	buffer := make([]byte, gateway.StreamCopyBytes)
	for {
		// Jangan biarkan deadline idle Write kedaluwarsa ketika menunggu Read upstream.
		if err := s.deadline(s.end); err != nil {
			return err
		}
		n, readErr := body.Read(buffer)
		if n > 0 {
			if err := s.writeDeadline(); err != nil {
				return err
			}
			written, err := s.writer.Write(buffer[:n])
			if err != nil {
				return err
			}
			if written != n {
				return io.ErrShortWrite
			}
			if err := s.controller.Flush(); err != nil {
				return err
			}
		}
		if readErr != nil {
			if err := s.context.Err(); err != nil {
				return err
			}
			if !time.Now().Before(s.end) {
				return context.DeadlineExceeded
			}
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}
