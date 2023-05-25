package server

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/instrument"
	"github.com/weaveworks/common/middleware"
	"time"
)

type serverMetrics struct {
	registerer          prometheus.Registerer
	tcpConnections      *prometheus.GaugeVec
	tcpConnectionsLimit *prometheus.GaugeVec
	requestDuration     *prometheus.HistogramVec
	receivedMessageSize *prometheus.HistogramVec
	sentMessageSize     *prometheus.HistogramVec
	inflightRequests    *prometheus.GaugeVec
}

func newServerMetrics(cfg Config) *serverMetrics {
	// If user doesn't supply a registerer/gatherer, use Prometheus' by default.
	reg := cfg.Registerer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	return &serverMetrics{
		registerer: reg,
		tcpConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cfg.MetricsNamespace,
			Name:      "tcp_connections",
			Help:      "Current number of accepted TCP connections.",
		}, []string{"protocol"}),
		tcpConnectionsLimit: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cfg.MetricsNamespace,
			Name:      "tcp_connections_limit",
			Help:      "The max number of TCP connections that can be accepted (0 means no limit).",
		}, []string{"protocol"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                       cfg.MetricsNamespace,
			Name:                            "request_duration_seconds",
			Help:                            "Time (in seconds) spent serving HTTP requests.",
			Buckets:                         instrument.DefBuckets,
			NativeHistogramBucketFactor:     cfg.MetricsNativeHistogramFactor,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		}, []string{"method", "route", "status_code", "ws"}),
		receivedMessageSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.MetricsNamespace,
			Name:      "request_message_bytes",
			Help:      "Size (in bytes) of messages received in the request.",
			Buckets:   middleware.BodySizeBuckets,
		}, []string{"method", "route"}),
		sentMessageSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.MetricsNamespace,
			Name:      "response_message_bytes",
			Help:      "Size (in bytes) of messages sent in response.",
			Buckets:   middleware.BodySizeBuckets,
		}, []string{"method", "route"}),
		inflightRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cfg.MetricsNamespace,
			Name:      "inflight_requests",
			Help:      "Current number of inflight requests.",
		}, []string{"method", "route"}),
	}
}

func (s *serverMetrics) mustRegister() {
	s.registerer.MustRegister(
		s.tcpConnections,
		s.tcpConnectionsLimit,
		s.requestDuration,
		s.receivedMessageSize,
		s.sentMessageSize,
		s.inflightRequests,
	)
}

func (s *serverMetrics) mustRegisterOrReuse() {
	s.tcpConnections = s.mustRegisterOrReuseGauge(s.tcpConnections)
	s.tcpConnectionsLimit = s.mustRegisterOrReuseGauge(s.tcpConnectionsLimit)
	s.requestDuration = s.mustRegisterOrReuseHistogram(s.requestDuration)
	s.receivedMessageSize = s.mustRegisterOrReuseHistogram(s.receivedMessageSize)
	s.sentMessageSize = s.mustRegisterOrReuseHistogram(s.sentMessageSize)
	s.inflightRequests = s.mustRegisterOrReuseGauge(s.inflightRequests)
}

func (s *serverMetrics) mustRegisterOrReuseHistogram(c *prometheus.HistogramVec) *prometheus.HistogramVec {
	err := s.registerer.Register(c)
	if err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
			panic(fmt.Sprintf("previously existing collector must be a HistogramVec, "+
				"but was %[1]T: %[1]v", alreadyRegistered.ExistingCollector))
		}
		panic(fmt.Sprintf("failed to register metrics: %v", err))
	}
	return c
}

func (s *serverMetrics) mustRegisterOrReuseGauge(c *prometheus.GaugeVec) *prometheus.GaugeVec {
	err := s.registerer.Register(c)
	if err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
			panic(fmt.Sprintf("previously existing collector must be a GaugeVec, "+
				"but was %[1]T: %[1]v", alreadyRegistered.ExistingCollector))
		}
		panic(fmt.Sprintf("failed to register metrics: %v", err))
	}
	return c
}
