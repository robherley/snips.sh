package stats

import (
	"net/url"
	"os"
	"time"

	"github.com/armon/go-metrics"
)

var cfg = func() *metrics.Config {
	c := &metrics.Config{
		ServiceName:          "snips",
		EnableHostname:       true,
		EnableRuntimeMetrics: true,
		EnableTypePrefix:     false,
		TimerGranularity:     time.Millisecond,
		ProfileInterval:      time.Second,
		FilterDefault:        true,
	}

	c.HostName, _ = os.Hostname()
	return c
}()

// Initialize sets the global metrics sink, if url is not specified, it will use the blackhole sink
func Initialize(statsdURL *url.URL) (*metrics.Metrics, error) {
	var (
		sink metrics.MetricSink = &metrics.BlackholeSink{}
		err  error
	)

	if statsdURL != nil && statsdURL.String() != "" {
		sink, err = metrics.NewStatsdSinkFromURL(statsdURL)
	}

	if err != nil {
		return nil, err
	}

	return metrics.NewGlobal(cfg, sink)
}

// Default returns the default shared metrics instance
func Default() *metrics.Metrics {
	return metrics.Default()
}
