package metric

import (
	"encoding/json"
	"expvar"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

type (
	h map[string]interface{}
	v []interface{}
)

func mockTime(sec int) func() time.Time {
	return func() time.Time {
		return time.Date(2017, 8, 11, 9, 0, sec, 0, time.UTC)
	}
}

func assertJSON(t *testing.T, o1, o2 interface{}) {
	var result, expect interface{}
	if reflect.TypeOf(o2).Kind() == reflect.Slice {
		result, expect = v{}, v{}
	} else {
		result, expect = h{}, h{}
	}
	if b1, err := json.Marshal(o1); err != nil {
		t.Fatal(o1, err)
	} else if err := json.Unmarshal(b1, &result); err != nil {
		t.Fatal(err)
	} else if b2, err := json.Marshal(o2); err != nil {
		t.Fatal(o2, err)
	} else if err := json.Unmarshal(b2, &expect); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(result, expect) {
		t.Fatal(result, expect)
	}
}

func TestCounter(t *testing.T) {
	c := &counter{}
	assertJSON(t, c, h{"type": "c", "count": 0})
	c.Add(1)
	assertJSON(t, c, h{"type": "c", "count": 1})
	c.Add(10)
	assertJSON(t, c, h{"type": "c", "count": 11})
	c.Reset()
	assertJSON(t, c, h{"type": "c", "count": 0})
}

func TestTimeline(t *testing.T) {
	now = mockTime(0)
	c := NewCounter("3s1s")
	count := func(x float64) h { return h{"type": "c", "count": x} }
	assertJSON(t, c, h{"interval": 1, "samples": v{count(0), count(0), count(0)}})
	c.Add(1)
	assertJSON(t, c, h{"interval": 1, "samples": v{count(1), count(0), count(0)}})
	now = mockTime(1)
	// We want to keep values of recent frame until they were read
	assertJSON(t, c, h{"interval": 1, "samples": v{count(1), count(0), count(0)}})
	c.Add(5)
	assertJSON(t, c, h{"interval": 1, "samples": v{count(5), count(1), count(0)}})
	now = mockTime(3)
	assertJSON(t, c, h{"interval": 1, "samples": v{count(5), count(1), count(0)}})
	assertJSON(t, c, h{"interval": 1, "samples": v{count(0), count(0), count(5)}})
}

func TestExpVar(t *testing.T) {
	now = mockTime(0)
	expvar.Publish("test:count", NewCounter())
	expvar.Publish("test:timeline", NewCounter("3s1s"))
	expvar.Get("test:count").(Metric).Add(1)
	expvar.Get("test:timeline").(Metric).Add(1)
	if expvar.Get("test:count").String() != `{"type":"c","count":1}` {
		t.Fatal(expvar.Get("test:count"))
	}
	if expvar.Get("test:timeline").String() != `{"interval":1,"samples":[{"type":"c","count":1},{"type":"c","count":0},{"type":"c","count":0}]}` {
		t.Fatal(expvar.Get("test:timeline"))
	}
	now = mockTime(1)
	if expvar.Get("test:count").String() != `{"type":"c","count":1}` {
		t.Fatal(expvar.Get("test:count"))
	}
	if expvar.Get("test:timeline").String() != `{"interval":1,"samples":[{"type":"c","count":1},{"type":"c","count":0},{"type":"c","count":0}]}` {
		t.Fatal(expvar.Get("test:timeline"))
	}
}

func BenchmarkMetrics(b *testing.B) {
	b.Run("counter", func(b *testing.B) {
		c := &counter{}
		for i := 0; i < b.N; i++ {
			c.Add(rand.Float64())
		}
	})
	b.Run("timeline/counter", func(b *testing.B) {
		c := NewCounter("10s1s")
		for i := 0; i < b.N; i++ {
			c.Add(rand.Float64())
		}
	})
}
