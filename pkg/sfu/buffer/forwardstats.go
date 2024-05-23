package buffer

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	jitterUpdateInterval = 1 * time.Millisecond
	rttUpdateInterval    = time.Second
	rttWindowLength      = 5 * time.Second
)

type ForwardStats struct {
	lock       sync.Mutex
	lastLeftMs atomic.Int64

	// lastPktTimeStamp time.Time
	lastTransit time.Duration
	jitter      time.Duration

	delay *LatencyAggregate
}

func NewForwardStats() *ForwardStats {
	return &ForwardStats{
		delay: NewLatencyAggregate(50*time.Millisecond, time.Second),
	}
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
	s.delay.Update(time.Duration(arrival.UnixNano()), float64(transit))
	// if arrival.Sub(s.lastPktTimeStamp) > jitterUpdateInterval {
	d := transit - s.lastTransit
	s.lastTransit = transit
	if d < 0 {
		d = -d
	}
	s.jitter += (d - s.jitter) / 16
	// s.lastPktTimeStamp = arrival
	// }
}

func (s *ForwardStats) GetJitter() time.Duration {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.jitter
}

func (s *ForwardStats) GetLatency() time.Duration {
	s.lock.Lock()
	defer s.lock.Unlock()
	return time.Duration(s.delay.Summarize().Mean())
}
