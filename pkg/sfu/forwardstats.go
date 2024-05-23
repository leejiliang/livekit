package sfu

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/livekit/livekit-server/pkg/telemetry/prometheus"
	"github.com/livekit/protocol/utils"
)

type ForwardStats struct {
	lock       sync.Mutex
	lastLeftMs atomic.Int64

	lastTransit time.Duration
	jitter      time.Duration

	latency *utils.LatencyAggregate

	closeCh chan struct{}
}

func NewForwardStats(latencyUpdateInterval, reportInterval, latencyWindowLength time.Duration) *ForwardStats {
	s := &ForwardStats{
		latency: utils.NewLatencyAggregate(latencyUpdateInterval, latencyWindowLength),
		closeCh: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.closeCh:
				return
			case <-ticker.C:
				latency, jitter, _ := s.GetLastStats(reportInterval)
				prometheus.RecordForwardJitter(float64(jitter / time.Millisecond))
				prometheus.RecordForwardLatency(float64(latency / time.Millisecond))
			}
		}
	}()
	return s
}

func (s *ForwardStats) Update(arrival, left time.Time) {
	leftMs := left.UnixMilli()
	lastMs := s.lastLeftMs.Load()
	if leftMs < lastMs || !s.lastLeftMs.CompareAndSwap(lastMs, leftMs) {
		return
	}

	transit := left.Sub(arrival)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.latency.Update(time.Duration(arrival.UnixNano()), float64(transit))
	d := transit - s.lastTransit
	s.lastTransit = transit
	if d < 0 {
		d = -d
	}
	s.jitter += (d - s.jitter) / 16
}

func (s *ForwardStats) GetStats() (latency, jitter, jitter2 time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	w := s.latency.Summarize()
	return time.Duration(w.Mean()), time.Duration(w.StdDev()), s.jitter
}

func (s *ForwardStats) GetLastStats(duration time.Duration) (latency, jitter, jitter2 time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	w := s.latency.SummarizeLast(duration)
	return time.Duration(w.Mean()), time.Duration(w.StdDev()), s.jitter
}

func (s *ForwardStats) Stop() {
	close(s.closeCh)
}
