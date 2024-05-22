package buffer

import (
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/livekit/protocol/utils"
)

// a ring buffer of welford mean/var summaries used to aggregate jitter and rtt.
type LatencyAggregate struct {
	summary []utils.Welford
	ivl     time.Duration
	cap     uint64
	head    uint64
}

func NewLatencyAggregate(interval, windowLength time.Duration) *LatencyAggregate {
	c := uint64((windowLength + interval - 1) / interval)
	return &LatencyAggregate{
		summary: make([]utils.Welford, c),
		ivl:     interval,
		cap:     uint64(c),
	}
}

// extend the ring to contain ts then merge the value into the interval summary.
func (a *LatencyAggregate) Update(ts time.Duration, v float64) {
	i := uint64(ts / a.ivl)
	if i+a.cap < a.head {
		return
	}

	if i > a.head {
		k := a.head + 1
		if k+a.cap < i {
			k = i - a.cap
		}
		for ; k <= i; k++ {
			a.summary[k%a.cap].Reset()
		}
		a.head = i
	}

	a.summary[i%a.cap].Update(v)
}

func (a *LatencyAggregate) Get(ts time.Duration) (utils.Welford, bool) {
	i := uint64(ts / a.ivl)
	if i+a.cap < a.head || i > a.head {
		return utils.Welford{}, false
	}
	return a.summary[i%a.cap], true
}

// aggregate the interval summaries
func (a *LatencyAggregate) Summarize() utils.Welford {
	return utils.WelfordMerge(a.summary...)
}

func (a *LatencyAggregate) SummarizeLast(d time.Duration) utils.Welford {
	n := min(a.head, a.cap, uint64((d+a.ivl-1)/a.ivl))
	l := (a.head - n) % a.cap
	r := (a.head % a.cap)
	if l < r {
		return utils.WelfordMerge(a.summary[l:r]...)
	}
	return utils.WelfordMerge(
		utils.WelfordMerge(a.summary[:r]...),
		utils.WelfordMerge(a.summary[l:]...),
	)
}

func (a *LatencyAggregate) MarshalLogObject(e zapcore.ObjectEncoder) error {
	summary := a.Summarize()
	e.AddFloat64("count", summary.Count())
	e.AddFloat64("mean", summary.Mean())
	e.AddFloat64("stddev", summary.StdDev())
	return nil
}
