package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	_ "net/http/pprof" // used for debug pprof at the default path.
	"strings"
	"time"

	"github.com/influxdata/influxdb/kit/prom"
	"github.com/influxdata/influxdb/kit/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	// MetricsPath exposes the prometheus metrics over /metrics.
	MetricsPath = "/metrics"
	// ReadyPath exposes the readiness of the service over /ready.
	ReadyPath = "/ready"
	// HealthPath exposes the health of the service over /health.
	HealthPath = "/health"
	// DebugPath exposes /debug/pprof for go debugging.
	DebugPath = "/debug"
)

// Handler provides basic handling of metrics, health and debug endpoints.
// All other requests are passed down to the sub handler.
type Handler struct {
	name string
	// MetricsHandler handles metrics requests
	MetricsHandler http.Handler
	// ReadyHandler handles readiness checks
	ReadyHandler http.Handler
	// HealthHandler handles health requests
	HealthHandler http.Handler
	// DebugHandler handles debug requests
	DebugHandler http.Handler
	// Handler handles all other requests
	Handler http.Handler

	requests   *prometheus.CounterVec
	requestDur *prometheus.HistogramVec

	// Logger if set will log all HTTP requests as they are served
	Logger *zap.Logger
}

// NewHandler creates a new handler with the given name.
// The name is used to tag the metrics produced by this handler.
//
// The MetricsHandler is set to the default prometheus handler.
// It is the caller's responsibility to call prometheus.MustRegister(h.PrometheusCollectors()...).
// In most cases, you want to use NewHandlerFromRegistry instead.
func NewHandler(name string) *Handler {
	h := &Handler{
		name:           name,
		MetricsHandler: promhttp.Handler(),
		DebugHandler:   http.DefaultServeMux,
	}
	h.initMetrics()
	return h
}

// NewHandlerFromRegistry creates a new handler with the given name,
// and sets the /metrics endpoint to use the metrics from the given registry,
// after self-registering h's metrics.
func NewHandlerFromRegistry(name string, reg *prom.Registry) *Handler {
	h := &Handler{
		name:           name,
		MetricsHandler: reg.HTTPHandler(),
		ReadyHandler:   http.HandlerFunc(ReadyHandler),
		HealthHandler:  http.HandlerFunc(HealthHandler),
		DebugHandler:   http.DefaultServeMux,
	}
	h.initMetrics()
	reg.MustRegister(h.PrometheusCollectors()...)
	return h
}

// ServeHTTP delegates a request to the appropriate subhandler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var span opentracing.Span
	span, r = tracing.ExtractFromHTTPRequest(r, h.name)
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	defer span.Finish()

	// TODO: better way to do this?
	statusW := newStatusResponseWriter(w)
	w = statusW

	// TODO: This could be problematic eventually. But for now it should be fine.
	defer func(start time.Time) {
		duration := time.Since(start)
		statusClass := statusW.statusCodeClass()
		statusCode := statusW.code()
		h.requests.With(prometheus.Labels{
			"handler":    h.name,
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     statusClass,
			"user_agent": userAgent,
		}).Inc()
		h.requestDur.With(prometheus.Labels{
			"handler":    h.name,
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     statusClass,
			"user_agent": userAgent,
		}).Observe(duration.Seconds())
		if h.Logger != nil {
			errField := zap.Skip()
			if errStr := w.Header().Get(PlatformErrorCodeHeader); errStr != "" {
				errField = zap.Error(errors.New(errStr))
			}
			errReferenceField := zap.Skip()
			if errReference := w.Header().Get(PlatformErrorCodeHeader); errReference != "" {
				errReferenceField = zap.String("error_code", PlatformErrorCodeHeader)
			}

			h.Logger.Debug("Request",
				zap.String("handler", h.name),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", statusCode),
				zap.Int("duration_ns", int(duration)),
				errField,
				errReferenceField,
			)
		}
	}(time.Now())

	switch {
	case r.URL.Path == MetricsPath:
		h.MetricsHandler.ServeHTTP(w, r)
	case r.URL.Path == ReadyPath:
		h.ReadyHandler.ServeHTTP(w, r)
	case r.URL.Path == HealthPath:
		h.HealthHandler.ServeHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, DebugPath):
		h.DebugHandler.ServeHTTP(w, r)
	default:
		h.Handler.ServeHTTP(w, r)
	}
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, code int, res interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	return json.NewEncoder(w).Encode(res)
}

// PrometheusCollectors satisifies prom.PrometheusCollector.
func (h *Handler) PrometheusCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		h.requests,
		h.requestDur,
	}
}

func (h *Handler) initMetrics() {
	const namespace = "http"
	const handlerSubsystem = "api"

	h.requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: handlerSubsystem,
		Name:      "requests_total",
		Help:      "Number of http requests received",
	}, []string{"handler", "method", "path", "status", "user_agent"})

	h.requestDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: handlerSubsystem,
		Name:      "request_duration_seconds",
		Help:      "Time taken to respond to HTTP request",
	}, []string{"handler", "method", "path", "status", "user_agent"})
}

func logEncodingError(logger *zap.Logger, r *http.Request, err error) {
	// If we encounter an error while encoding the response to an http request
	// the best thing we can do is log that error, as we may have already written
	// the headers for the http request in question.
	logger.Info("error encoding response",
		zap.String("path", r.URL.Path),
		zap.String("method", r.Method),
		zap.Error(err))
}
