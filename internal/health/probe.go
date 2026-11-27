// Package health menjadwalkan observasi, tanpa memiliki keputusan eligibility.
package health

import (
	"context"
	"errors"
	"opencode-infinity/internal/pool"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// Schedule wajib eksplisit; angka fixture bukan default produksi.
type Schedule struct{ Interval, Jitter, Timeout, Freshness, MaxBackoff time.Duration }

// Prober merupakan adapter server tepercaya. Wajib menghormati Context, tidak
// melakukan generasi/retry/direct fallback, dan menutup seluruh socket sebelum return.
// shortcut: adapter SOCKS5/HTTPS runtime menunggu perakitan; S02 memakai fixture lokal.
type Prober interface {
	Probe(context.Context, string, pool.HealthLayer) (pool.HealthState, error)
}
type Scheduler struct {
	owner    *pool.Manager
	schedule Schedule
	prober   Prober
	jitter   func(time.Duration) time.Duration
	running  atomic.Bool
}

func New(owner *pool.Manager, schedule Schedule, prober Prober, jitter func(time.Duration) time.Duration) (*Scheduler, error) {
	if owner == nil || owner.ProbeConcurrency() == 0 || prober == nil || nilProber(prober) || schedule.Interval <= 0 || schedule.Timeout <= 0 || schedule.Freshness <= schedule.Timeout || schedule.Jitter < 0 || schedule.Jitter >= schedule.Interval || schedule.MaxBackoff < schedule.Interval || schedule.MaxBackoff > time.Duration(1<<62) || schedule.Freshness > time.Duration(1<<62) || (schedule.Jitter > 0 && jitter == nil) {
		return nil, errors.New("probe_invalid_component")
	}
	return &Scheduler{owner: owner, schedule: schedule, prober: prober, jitter: jitter}, nil
}

type job struct {
	relay   string
	layer   pool.HealthLayer
	next    time.Time
	backoff time.Duration
	active  bool
}
type work struct {
	index   int
	relay   string
	layer   pool.HealthLayer
	release func()
}
type completion struct {
	observedAt time.Time
	index      int
	state      pool.HealthState
}

// Run hanya dijalankan eksplisit oleh perakitan server, tidak oleh pembacaan status.
func (s *Scheduler) Run(parent context.Context) error {
	if s == nil || s.owner == nil || parent == nil || !s.running.CompareAndSwap(false, true) {
		return errors.New("probe_invalid_run")
	}
	defer s.running.Store(false)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	workers := s.owner.ProbeConcurrency()
	jobs := make([]job, 0, 2*len(s.owner.View().Relays))
	for _, r := range s.owner.View().Relays {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			jobs = append(jobs, job{relay: r.ID, layer: layer, next: time.Now(), backoff: s.schedule.Interval})
		}
	}
	queue := make(chan work)
	finished := make(chan completion, workers)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	for range workers {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-queue:
					observedAt := time.Now()
					probeCtx, stop := context.WithTimeout(ctx, s.schedule.Timeout)
					state, err := s.prober.Probe(probeCtx, task.relay, task.layer)
					if err != nil || probeCtx.Err() != nil || (state != pool.HealthHealthy && state != pool.HealthUnhealthy) {
						state = pool.HealthUnhealthy
					}
					stop()
					task.release()
					finished <- completion{observedAt: observedAt, index: task.index, state: state}
				}
			}
		})
	}
	timer := time.NewTimer(s.schedule.Interval)
	defer timer.Stop()
	active := 0
	for {
		if ctx.Err() != nil {
			return nil
		}
		now := time.Now()
		for i := range jobs {
			j := &jobs[i]
			if j.active || j.next.After(now) || active >= workers {
				continue
			}
			release, ok := s.owner.TryProbe(j.relay, j.layer)
			if !ok {
				j.next = now.Add(s.schedule.Interval)
				continue
			}
			task := work{i, j.relay, j.layer, release}
			select {
			case queue <- task:
				j.active = true
				active++
			case <-ctx.Done():
				release()
				return nil
			}
		}
		delay := s.schedule.Interval
		for _, j := range jobs {
			if !j.active {
				d := time.Until(j.next)
				if d < delay {
					delay = d
				}
			}
		}
		// Pada semua worker sibuk, tunggu completion, bukan putar timer nol untuk job tertunda.
		if active >= workers {
			delay = s.schedule.Interval
		}
		if delay < time.Nanosecond {
			delay = time.Nanosecond
		}
		timer.Reset(delay)
		select {
		case <-ctx.Done():
			return nil
		case result := <-finished:
			active--
			j := &jobs[result.index]
			j.active = false
			now = time.Now()
			observation := pool.ObservationView{State: result.state, ObservedAt: result.observedAt, ValidUntil: result.observedAt.Add(s.schedule.Freshness)}
			// Observe menolak hasil lama/konflik; error tidak diteruskan sebagai data mentah.
			// Hasil yang kalah oleh observasi lebih baru/expired tidak menimpa owner.
			_ = s.owner.Observe(j.relay, j.layer, observation)
			if result.state == pool.HealthHealthy {
				j.backoff = s.schedule.Interval
			} else {
				if j.backoff > s.schedule.MaxBackoff/2 {
					j.backoff = s.schedule.MaxBackoff
				} else {
					j.backoff *= 2
				}
			}
			offset := time.Duration(0)
			if s.jitter != nil {
				offset = s.jitter(s.schedule.Jitter)
			}
			offset = max(-s.schedule.Jitter, min(s.schedule.Jitter, offset))
			delay := j.backoff
			if offset > 0 && offset > s.schedule.MaxBackoff-delay {
				delay = s.schedule.MaxBackoff
			} else {
				delay += offset
			}
			j.next = now.Add(delay)
		case <-timer.C:
		}
	}
}

func nilProber(p Prober) bool {
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Map, reflect.Chan, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}
