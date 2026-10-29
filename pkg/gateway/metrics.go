package gateway

import (
	"net/http"
)

// HandleMetrics returns operational metrics summary.
func (gw *Gateway) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	summary := gw.tracker.GetSummary()
	writeJSON(w, http.StatusOK, summary)
}

// HandlePrometheusMetrics returns standard Prometheus exposition format text.
func (gw *Gateway) HandlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(gw.tracker.FormatPrometheus()))
}
