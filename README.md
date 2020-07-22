# metric

[![GoDoc](https://godoc.org/github.com/yum-install-brains/metric?status.svg)](https://godoc.org/github.com/yum-install-brains/metric)
[![Go Report Card](https://goreportcard.com/badge/github.com/yum-install-brains/metric)](https://goreportcard.com/report/github.com/yum-install-brains/metric)

Package provides simple uniform interface for counters. 

It is compatible with [expvar](https://golang.org/pkg/expvar/) package, that is
also commonly used for monitoring.

**Fork notes**: this fork removes gauges and histograms and slightly changes the behaviour of frame roll logic.

## Usage

```go
// Create new metric. All metrics may take time frames if you want them to keep
// history. If no time frames are given the metric only keeps track of a single
// current value.
c := metric.NewCounter(15 * time.Minute, 10 * time.Second) // 15 minutes of history with 10 second precision
// Increment counter
c.Add(1)
// Return JSON with all recorded counter values
c.String() // Or json.Marshal(c)
```

Metrics are thread-safe and can be updated from background goroutines.

## License

Code is distributed under MIT license, feel free to use it in your proprietary
projects as well.
