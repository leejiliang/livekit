package buffer

import "time"

const (
	jitterUpdateInterval = 10 * time.Millisecond
)

type ForwardStats struct {
	latency uint32

	lastPktTimeStamp time.Time
	lastTransit      time.Time
	jitter           time.Duration
}

func NewForwardStats() *ForwardStats {
	return &ForwardStats{}
}

func (s *ForwardStats) Update(arrival, left time.Time) {
	delay := left.Sub(arrival)
	if arrival.Sub(s.lastPktTimeStamp) > jitterUpdateInterval {
		// s.latency = uint32(delay.Milliseconds())
		s.jitter += (delay - s.jitter) / 16
		s.lastPktTimeStamp = arrival
	}
}

func (s *ForwardStats) GetJitter() time.Duration {
	return s.jitter
}
