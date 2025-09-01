package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/perses/promql-builder/label"
	"github.com/perses/promql-builder/vector"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type DowntimeLister interface {
	ListWindows(from time.Time, to time.Time) ([]types.DowntimeWindow, error)
	ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]types.DowntimeWindow, error)
}

type PrometheusQuerier interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...prometheusv1.Option) (model.Value, prometheusv1.Warnings, error)
	QueryRange(ctx context.Context, query string, r prometheusv1.Range, options ...prometheusv1.Option) (model.Value, prometheusv1.Warnings, error)
}

type queryServer struct {
	lister DowntimeLister
	prom   PrometheusQuerier
}

func (s *queryServer) QueryCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clusterID := r.PathValue("clusterid")

	from := r.URL.Query().Get("from")
	fromT, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `from` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}
	fromT = fromT.Truncate(time.Hour)
	to := r.URL.Query().Get("to")
	toT, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `to` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}
	toT = toT.Truncate(time.Hour)

	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = ".*"
	}

	downtimes, err := s.lister.ListWindowsMatchingClusterFacts(ctx, fromT, toT, clusterID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not list downtimes (%s)", err.Error()), http.StatusInternalServerError)
		return
	}

	hours := int(toT.Sub(fromT).Hours())
	if hours <= 0 {
		http.Error(w, "`to` must be at least 1 hour after `from`", http.StatusBadRequest)
		return
	}

	rawSamples, _, err := s.prom.QueryRange(
		ctx,
		vector.New(
			vector.WithMetricName("slo:sli_error:ratio_rate1h"),
			vector.WithLabelMatchers(
				label.New("cluster_id").Equal(clusterID),
				label.New("sloth_service").EqualRegexp(filter),
			)).String(),
		prometheusv1.Range{
			Start: fromT,
			End:   toT,
			Step:  time.Hour,
		})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not query Prometheus (%s)", err.Error()), http.StatusInternalServerError)
		return
	}
	samples, ok := rawSamples.(model.Matrix)
	if !ok {
		http.Error(w, fmt.Sprintf("Error: Unexpected result type from Prometheus (expected model.Matrix, got %T)", rawSamples), http.StatusInternalServerError)
		return
	}

	rawObjective, _, err := s.prom.Query(
		ctx,
		vector.New(
			vector.WithMetricName("slo:objective:ratio"),
			vector.WithLabelMatchers(
				label.New("cluster_id").Equal(clusterID),
				label.New("sloth_service").EqualRegexp(filter),
			)).String(), toT)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not query Prometheus for objective (%s)", err.Error()), http.StatusInternalServerError)
		return
	}
	objectives, ok := rawObjective.(model.Vector)
	if !ok {
		http.Error(w, fmt.Sprintf("Error: Unexpected result type from Prometheus for objective (expected model.Vector, got %T)", rawObjective), http.StatusInternalServerError)
		return
	}
	objectiveMap := make(map[string]float64)
	for _, sample := range objectives {
		name := string(sample.Metric["sloth_service"])
		if name == "" {
			fmt.Println("Warning: Found objective sample without sloth_service label, skipping", sample.Metric)
			continue
		}
		objectiveMap[name] = float64(sample.Value)
	}

	response := QueryClusterResponse{
		ClusterID: clusterID,
		SLIData:   make(map[string]QueryClusterResponseSLIData),
	}

	for _, sample := range samples {
		name := string(sample.Metric["sloth_service"])
		if name == "" {
			fmt.Println("Warning: Found sample without sloth_service label, skipping", sample.Metric)
			continue
		}
		d := response.SLIData[name]
		for _, pair := range sample.Values {
			val := float64(pair.Value)
			if timeMatchesDowntimeWindow(pair.Timestamp.Time(), downtimes) {
				val = 0
			}

			d.DataPoints = append(d.DataPoints, SLIDataPoint{
				Timestamp:       pair.Timestamp.Time(),
				ErrorRate1h:     val,
				RealErrorRate1h: float64(pair.Value),
			})
		}
		response.SLIData[name] = d
	}

	for name, d := range response.SLIData {
		obj, ok := objectiveMap[name]
		if !ok {
			fmt.Println("Warning: Could not find objective for service", name)
			continue
		}
		d.Objective = obj
		var sum float64
		for _, dp := range d.DataPoints {
			sum += dp.ErrorRate1h
		}
		d.ErrorRateWindow = sum / float64(hours)
		d.ErrorBudgetWindow = 1.0 - d.Objective - d.ErrorRateWindow
		d.ErrorBudgetWindowPercentage = d.ErrorBudgetWindow / (1.0 - d.Objective)
		response.SLIData[name] = d
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type QueryClusterResponse struct {
	ClusterID string `json:"cluster_id"`

	SLIData map[string]QueryClusterResponseSLIData `json:"sli_data"`
}

type QueryClusterResponseSLIData struct {
	// Objective is the SLO objective for this service, e.g. 0.98 for 98%
	Objective float64 `json:"objective"`
	// ErrorRateWindow is the average error rate over the entire window.
	// Null time points are treated as 0 error rate.
	ErrorRateWindow float64 `json:"error_rate_window"`
	// ErrorBudgetWindow is the remaining error budget over the entire window.
	// It is calculated as (1 - objective) - ErrorRateWindow.
	// It can be negative if the error rate exceeded the objective.
	ErrorBudgetWindow float64 `json:"error_budget_window"`
	// ErrorBudgetWindowPercentage is the percentage of the error budget remaining calculated over the entire window.
	// It is calculated as ErrorBudgetWindow / (1 - objective).
	// It can be negative if the error rate exceeded the objective.
	ErrorBudgetWindowPercentage float64 `json:"error_budget_window_percent"`
	// DataPoints contains the error rate for each hour in the window.
	DataPoints []SLIDataPoint `json:"data_points"`
}

type SLIDataPoint struct {
	// Timestamp is the time of the data point as provided by Prometheus.
	Timestamp time.Time `json:"timestamp"`
	// ErrorRate1h is the error rate for the past hour adjusted for downtimes.
	ErrorRate1h float64 `json:"error_rate_1h"`
	// RealErrorRate1h is the raw error rate for the past hour as reported by Prometheus.
	RealErrorRate1h float64 `json:"real_error_rate_1h"`
}

func Setup(mux *http.ServeMux, lister DowntimeLister, prom PrometheusQuerier) {
	s := queryServer{lister: lister, prom: prom}
	log.Println("Registering endpoints")
	mux.HandleFunc("GET /query/cluster/{clusterid}", s.QueryCluster)
}

// as Prometheus looks back in time from T the window is matched as such: (windows.start, windows.end]
func timeMatchesDowntimeWindow(ts time.Time, windows []types.DowntimeWindow) bool {
	for _, w := range windows {
		if (w.StartTime == nil || ts.After(*w.StartTime)) && (w.EndTime == nil || !ts.After(*w.EndTime)) {
			return true
		}
	}
	return false
}
