package metric

import (
	"encoding/json"
	"expvar"
	"fmt"
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
		fmt.Println(result)
		fmt.Println(expect)
		t.Fatal(result, expect)
	}
}

func TestCounter(t *testing.T) {
	c := NewCounter()
	assertJSON(t, c, h{"type": "c", "count": 0})
	c.Add(1)
	assertJSON(t, c, h{"type": "c", "count": 1})
	c.Add(10)
	assertJSON(t, c, h{"type": "c", "count": 11})
}

func TestMetricReset(t *testing.T) {
	c := &counter{}
	c.Add(5)
	assertJSON(t, c, h{"type": "c", "count": 5})
	c.Reset()
	assertJSON(t, c, h{"type": "c", "count": 0})
}

func TestMetricString(t *testing.T) {
	c := NewCounter()
	c.Add(1)
	c.Add(3)
	if s := c.String(); s != "4" {
		t.Fatal(s)
	}
}

func TestCounterTimeline(t *testing.T) {
	now = mockTime(0)
	c := NewCounter(1*time.Second, 3*time.Second)
	expect := func(total float64, samples ...float64) h {
		timeline := v{}
		for _, s := range samples {
			timeline = append(timeline, h{"type": "c", "count": s})
		}
		return h{
			"interval": 1,
			"total":    h{"type": "c", "count": total},
			"samples":  timeline,
		}
	}
	assertJSON(t, c, expect(0, 0, 0, 0))
	c.Add(1)
	assertJSON(t, c, expect(1, 1, 0, 0))
	now = mockTime(1)
	assertJSON(t, c, expect(1, 0, 1, 0))
	c.Add(5)
	assertJSON(t, c, expect(6, 5, 1, 0))
	now = mockTime(3)
	assertJSON(t, c, expect(5, 0, 0, 5))
	now = mockTime(10)
	assertJSON(t, c, expect(0, 0, 0, 0))
}

func TestExpVar(t *testing.T) {
	expvar.Publish("test:count", NewCounter())
	expvar.Get("test:count").(Metric).Add(1)
	if s := expvar.Get("test:count").String(); s != `1` {
		t.Fatal(s)
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
		c := NewCounter(1*time.Second, 10*time.Second)
		for i := 0; i < b.N; i++ {
			c.Add(rand.Float64())
		}
	})
}
