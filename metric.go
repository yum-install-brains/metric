package metric

import (
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// To mock time in tests
var now = time.Now

// Metric is a single meter (counter, gauge or histogram, optionally - with history)
type Metric interface {
	Add(n float64)
	String() string
}

// metric is an extended private interface
// Counters implement it.
type metric interface {
	Metric
	Reset()
	Aggregate(samples []metric)
}

// NewCounter returns a counter metric that increments the value with each
// incoming number.
func NewCounter(frame ...time.Duration) Metric {
	return newMetric(func() metric { return &counter{} }, frame...)
}

type timeseries struct {
	sync.Mutex
	now      time.Time
	size     int
	interval time.Duration
	total    metric
	samples  []metric
}

func (ts *timeseries) Reset() {
	ts.total.Reset()
	for _, s := range ts.samples {
		s.Reset()
	}
}

func (ts *timeseries) roll() {
	t := now()
	roll := int((t.Round(ts.interval).Sub(ts.now.Round(ts.interval))) / ts.interval)
	ts.now = t
	n := len(ts.samples)
	if roll <= 0 {
		return
	}
	if roll >= len(ts.samples) {
		ts.Reset()
	} else {
		for i := 0; i < roll; i++ {
			tmp := ts.samples[n-1]
			for j := n - 1; j > 0; j-- {
				ts.samples[j] = ts.samples[j-1]
			}
			ts.samples[0] = tmp
			ts.samples[0].Reset()
		}
		ts.total.Aggregate(ts.samples)
	}
}

func (ts *timeseries) Add(n float64) {
	ts.Lock()
	defer ts.Unlock()
	ts.total.Add(n)
	ts.samples[0].Add(n)
}

func (ts *timeseries) MarshalJSON() ([]byte, error) {
	ts.Lock()
	defer ts.Unlock()
	ts.roll()
	return json.Marshal(struct {
		Interval float64  `json:"interval"`
		Total    Metric   `json:"total"`
		Samples  []metric `json:"samples"`
	}{float64(ts.interval) / float64(time.Second), ts.total, ts.samples})
}

func (ts *timeseries) String() string {
	ts.Lock()
	defer ts.Unlock()
	value := ts.total.String()
	ts.roll()
	return value
}

type counter struct {
	count uint64
}

func (c *counter) String() string { return strconv.FormatFloat(c.value(), 'g', -1, 64) }
func (c *counter) Reset()         { atomic.StoreUint64(&c.count, math.Float64bits(0)) }
func (c *counter) value() float64 { return math.Float64frombits(atomic.LoadUint64(&c.count)) }
func (c *counter) Add(n float64) {
	for {
		old := math.Float64frombits(atomic.LoadUint64(&c.count))
		new := old + n
		if atomic.CompareAndSwapUint64(&c.count, math.Float64bits(old), math.Float64bits(new)) {
			return
		}
	}
}
func (c *counter) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string  `json:"type"`
		Count float64 `json:"count"`
	}{"c", c.value()})
}

func (c *counter) Aggregate(samples []metric) {
	c.Reset()
	for _, s := range samples {
		c.Add(s.(*counter).value())
	}
}

func newTimeseries(builder func() metric, frame ...time.Duration) *timeseries {
	var interval time.Duration
	var totalDuration time.Duration

	if frame[0] == 0 {
		interval = time.Minute
	}

	if frame[1] == 0 {
		totalDuration = interval * 15
	}

	n := int(totalDuration / interval)

	samples := make([]metric, n, n)
	for i := 0; i < n; i++ {
		samples[i] = builder()
	}

	totalMetric := builder()

	return &timeseries{interval: interval, total: totalMetric, samples: samples}
}

func newMetric(builder func() metric, frame ...time.Duration) Metric {
	// We need to provide both interval and totalDuration to be able to use time series
	if len(frame) < 2 {
		return builder()
	}
	return newTimeseries(builder, frame...)

}
