package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	RequestTotal      *prometheus.CounterVec
	RequestFailures   *prometheus.CounterVec
	ProviderCallTotal *prometheus.CounterVec
	RequestLatency    *prometheus.HistogramVec
	StreamingTotal    prometheus.Counter
	ProviderHealth    *prometheus.GaugeVec
	ProviderLatency   *prometheus.GaugeVec
	AsyncLogDropped   prometheus.Counter
}

func NewCollector() *Collector {
	return &Collector{
		RequestTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_http_requests_total",
			Help: "Total HTTP requests handled by the gateway.",
		}, []string{"method", "path", "status"}),
		RequestFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_http_request_failures_total",
			Help: "Total failed HTTP requests handled by the gateway.",
		}, []string{"method", "path", "status"}),
		ProviderCallTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_provider_calls_total",
			Help: "Total provider invocations.",
		}, []string{"provider", "model", "stream"}),
		RequestLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_gateway_request_latency_seconds",
			Help:    "Latency of gateway requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		StreamingTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_gateway_streaming_requests_total",
			Help: "Total streaming chat completion requests.",
		}),
		ProviderHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_gateway_provider_health",
			Help: "Provider health status, 1 means healthy.",
		}, []string{"provider"}),
		ProviderLatency: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ai_gateway_provider_latency_ms",
			Help: "Observed provider latency in milliseconds.",
		}, []string{"provider"}),
		AsyncLogDropped: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_gateway_async_log_dropped_total",
			Help: "Number of dropped async usage logs due to a full queue.",
		}),
	}
}

func (c *Collector) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		statusCode := strconv.Itoa(ctx.Writer.Status())
		c.RequestTotal.WithLabelValues(ctx.Request.Method, path, statusCode).Inc()
		c.RequestLatency.WithLabelValues(ctx.Request.Method, path).Observe(time.Since(start).Seconds())
		if ctx.Writer.Status() >= http.StatusBadRequest {
			c.RequestFailures.WithLabelValues(ctx.Request.Method, path, statusCode).Inc()
		}
	}
}

func (c *Collector) ObserveProviderCall(providerName, model string, stream bool) {
	c.ProviderCallTotal.WithLabelValues(providerName, model, strconv.FormatBool(stream)).Inc()
	if stream {
		c.StreamingTotal.Inc()
	}
}

func (c *Collector) SetProviderHealth(providerName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1
	}
	c.ProviderHealth.WithLabelValues(providerName).Set(value)
}

func (c *Collector) SetProviderLatency(providerName string, latencyMS float64) {
	c.ProviderLatency.WithLabelValues(providerName).Set(latencyMS)
}

func (c *Collector) Handler() http.Handler {
	return promhttp.Handler()
}
