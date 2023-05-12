package stats

import (
	"net/url"
	"os"
	"time"

	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/datadog"
	"github.com/rs/zerolog/log"
)

var cfg = func() *metrics.Config {
	c := &metrics.Config{
		ServiceName:          "snips",
		EnableHostname:       false,
		EnableRuntimeMetrics: false,
		EnableTypePrefix:     false,
		TimerGranularity:     time.Millisecond,
		ProfileInterval:      time.Second,
		FilterDefault:        true,
	}

	c.HostName, _ = os.Hostname()
	return c
}()

// Initialize sets the global metrics sink, if url is not specified, it will use the blackhole sink
func Initialize(statsdURL *url.URL, useDogStatsd bool) (*metrics.Metrics, error) {
	var (
		sink metrics.MetricSink = &metrics.BlackholeSink{}
		err  error
	)

	if statsdURL != nil && statsdURL.String() != "" {
		if useDogStatsd {
			log.Info().Str("url", statsdURL.String()).Msg("intializing dogstatsd metrics sink")
			sink, err = datadog.NewDogStatsdSink(statsdURL.Host, cfg.HostName)
		} else {
			log.Info().Str("url", statsdURL.String()).Msg("intializing statsd metrics sink")
			sink, err = metrics.NewStatsdSinkFromURL(statsdURL)
		}
	}

	if err != nil {
		return nil, err
	}

	return metrics.NewGlobal(cfg, sink)
}
