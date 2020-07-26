package metric

import (
	"encoding/json"
	"fmt"
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
	Get() []float64
}

type Syncronizer interface {
	GetTime() time.Time
	// Sync one metric frame start with another
	Sync(m Metric)
}

// NewCounter returns a counter metric that increments the value with each
// incoming number.
func NewCounter(frameStart time.Time, frames ...string) Metric {
	return newMetric(func() Metric { return &counter{} }, frameStart, frames...)
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
	val, err := json.Marshal(struct {
		Interval float64  `json:"interval"`
		Samples  []Metric `json:"samples"`
	}{float64(ts.interval) / float64(time.Second), ts.samples})
	ts.roll()
	return val, err
}

func (ts *timeseries) String() string {
	b, _ := ts.MarshalJSON()
	return string(b)
}

func (ts *timeseries) Get() []float64 {
	ts.Lock()
	defer ts.Unlock()

	values := make([]float64, len(ts.samples), len(ts.samples))

	for i, sample := range ts.samples {
		values[i] = sample.(*counter).value()
	}
	ts.roll()
	return values
}

func (ts *timeseries) GetTime() time.Time {
	ts.Lock()
	defer ts.Unlock()

	return ts.now
}

func (ts *timeseries) Sync(syncMetric Metric) {
	ts.Lock()
	defer ts.Unlock()

	ts.now = syncMetric.(*timeseries).now
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
func (c *counter) Get() []float64 { return []float64{c.value()} }
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

func newTimeseries(builder func() Metric, frameStart time.Time, frame string) *timeseries {
	var (
		totalNum, intervalNum   int
		totalUnit, intervalUnit rune
	)
	units := map[rune]time.Duration{
		's': time.Second,
		'm': time.Minute,
		'h': time.Hour,
		'd': time.Hour * 24,
		'w': time.Hour * 24 * 7,
		'M': time.Hour * 24 * 7 * 30,
		'y': time.Hour * 24 * 7 * 365,
	}
	fmt.Sscanf(frame, "%d%c%d%c", &totalNum, &totalUnit, &intervalNum, &intervalUnit)
	interval := units[intervalUnit] * time.Duration(intervalNum)
	if interval == 0 {
		interval = time.Minute
	}
	totalDuration := units[totalUnit] * time.Duration(totalNum)
	if totalDuration == 0 {
		totalDuration = interval * 15
	}
	n := int(totalDuration / interval)
	samples := make([]Metric, n, n)
	for i := 0; i < n; i++ {
		samples[i] = builder()
	}
	return &timeseries{interval: interval, samples: samples, now: frameStart}
}

func newMetric(builder func() Metric, frameStart time.Time, frames ...string) Metric {
	if len(frames) == 0 {
		return builder()
	}

	return newTimeseries(builder, frameStart, frames[0])
}
