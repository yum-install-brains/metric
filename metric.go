package metric

import (
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// To mock time in tests
var now = time.Now

// Metric is a single meter (counter, gauge or histogram, optionally - with history)
type Metric interface {
	Add(n float64)
	Reset()
	String() string
}

// NewCounter returns a counter metric that increments the value with each
// incoming number.
func NewCounter(frame ...time.Duration) Metric {
	return newMetric(func() Metric { return &counter{} }, frame...)
}

type timeseries struct {
	sync.Mutex
	now      time.Time
	size     int
	interval time.Duration
	samples  []Metric
}

func (ts *timeseries) Reset() {
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
	}
}

func (ts *timeseries) Add(n float64) {
	ts.Lock()
	defer ts.Unlock()
	//ts.roll()
	ts.samples[0].Add(n)
}

func (ts *timeseries) MarshalJSON() ([]byte, error) {
	ts.Lock()
	defer ts.Unlock()
	data, err := json.Marshal(struct {
		Interval float64  `json:"interval"`
		Samples  []Metric `json:"samples"`
	}{float64(ts.interval) / float64(time.Second), ts.samples})
	ts.roll()
	return data, err
}

func (ts *timeseries) String() string {
	b, _ := ts.MarshalJSON()
	return string(b)
}

func strjson(x interface{}) string {
	b, _ := json.Marshal(x)
	return string(b)
}

type counter struct {
	count uint64
}

func (c *counter) String() string { return strjson(c) }
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

func newTimeseries(builder func() Metric, frame ...time.Duration) *timeseries {
	var interval time.Duration
	var totalDuration time.Duration

	if frame[0] == 0 {
		interval = time.Minute
	} else {
		interval = frame[0]
	}

	if frame[1] == 0 {
		totalDuration = interval * 15
	} else {
		totalDuration = frame[1]
	}

	n := int(totalDuration / interval)
	samples := make([]Metric, n, n)
	for i := 0; i < n; i++ {
		samples[i] = builder()
	}
	return &timeseries{interval: interval, samples: samples}
}

func newMetric(builder func() Metric, frame ...time.Duration) Metric {
	// We need to provide both interval and totalDuration to be able to use time series
	if len(frame) < 2 {
		return builder()
	}

	return newTimeseries(builder, frame...)
}
