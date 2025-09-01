package query

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/vshn/vshn-sli-reporting/pkg/store/mock"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

func setup(rv []types.DowntimeWindow, q, qr staticPrometheusQuerierResponse) (*http.ServeMux, *mock.MockDowntimeStore) {
	store := &mock.MockDowntimeStore{
		ReturnValues: rv,
	}
	mux := http.NewServeMux()

	Setup(mux, store, staticPrometheusQuerier{queryRangeResponse: qr, queryResponse: q})
	return mux, store
}

func TestQueryWithDowntime(t *testing.T) {
	from := mustTimeFromRFC3339(t, "2020-01-01T00:00:00Z")
	to := mustTimeFromRFC3339(t, "2020-02-01T00:00:00Z")

	t.Logf("from: %s, to: %s, hours: %d", from, to, int(to.Sub(from).Hours()))

	mux, _ := setup(
		[]types.DowntimeWindow{
			{
				Title:     "Test1",
				StartTime: ptr.To(from.Add(6 * time.Hour)),
				EndTime:   ptr.To(from.Add((6 + 6) * time.Hour)),
			},
			{
				Title:     "Test2",
				StartTime: ptr.To(from.Add(9 * time.Hour)),
				EndTime:   ptr.To(from.Add((9 + 6) * time.Hour)),
			},
			{
				Title:     "Test3",
				StartTime: ptr.To(from.Add(20 * time.Hour)),
				EndTime:   ptr.To(from.Add((20 + 1) * time.Hour)),
			},
		}, staticPrometheusQuerierResponse{
			value: model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"__name__":      "slo:objective:ratio",
						"sloth_service": "full",
					},
					Value:     0.98,
					Timestamp: model.TimeFromUnixNano(to.UnixNano()),
				},
				&model.Sample{
					Metric: model.Metric{
						"__name__":      "slo:objective:ratio",
						"sloth_service": "empty",
					},
					Value:     0.98,
					Timestamp: model.TimeFromUnixNano(to.UnixNano()),
				},
			},
		}, staticPrometheusQuerierResponse{
			value: promMatrix(
				promSampleStream(t, sloErrorMetric("full"), mustTimeFromRFC3339(t, "2020-01-01T00:00:00Z"), "24x1"),
				promSampleStream(t, sloErrorMetric("empty"), mustTimeFromRFC3339(t, "2020-01-01T00:00:00Z"), ""),
			),
		})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/query/cluster/blub?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339)), nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	var queryResponse QueryClusterResponse
	err := json.NewDecoder(res.Body).Decode(&queryResponse)
	require.NoError(t, err)

	serviceFullDowntime := queryResponse.SLIData["full"]
	assert.Equal(t, explodeValues(t, "6x1 9x0 5x1 0 3x1"), collectErrorRate(serviceFullDowntime.DataPoints), "from is exclusive, to is inclusive")
	assert.Equal(t, explodeValues(t, "24x1"), collectRealErrorRate(serviceFullDowntime.DataPoints))
	assert.Equal(t, 0.98, serviceFullDowntime.Objective)
	// (6+5+3) / 744 hours (window width) = 0.0188
	assert.InDelta(t, 0.0188, serviceFullDowntime.ErrorRateWindow, 0.0001)
	// 98% objective allows 2% error budget, minus 1.88% used ~ 0.0012 remaining
	assert.InDelta(t, 0.0012, serviceFullDowntime.ErrorBudgetWindow, 0.0001)
	assert.InDelta(t, 0.059, serviceFullDowntime.ErrorBudgetWindowPercentage, 0.001)

	serviceEmptyDowntime := queryResponse.SLIData["empty"]
	assert.Equal(t, []float64{}, collectErrorRate(serviceEmptyDowntime.DataPoints))
	assert.Equal(t, []float64{}, collectRealErrorRate(serviceEmptyDowntime.DataPoints))
	assert.Equal(t, 0.98, serviceEmptyDowntime.Objective)
	assert.Equal(t, 0.0, serviceEmptyDowntime.ErrorRateWindow)
	assert.InDelta(t, 0.02, serviceEmptyDowntime.ErrorBudgetWindow, 0.001)
	assert.Equal(t, 1.0, serviceEmptyDowntime.ErrorBudgetWindowPercentage)
}

func collectErrorRate(dp []SLIDataPoint) []float64 {
	values := make([]float64, len(dp))
	for i := range dp {
		values[i] = dp[i].ErrorRate1h
	}
	return values
}
func collectRealErrorRate(dp []SLIDataPoint) []float64 {
	values := make([]float64, len(dp))
	for i := range dp {
		values[i] = dp[i].RealErrorRate1h
	}
	return values
}

type staticPrometheusQuerierResponse struct {
	value model.Value
	err   error
}

type staticPrometheusQuerier struct {
	queryRangeResponse staticPrometheusQuerierResponse
	queryResponse      staticPrometheusQuerierResponse
}

func (s staticPrometheusQuerier) Query(ctx context.Context, query string, ts time.Time, options ...prometheusv1.Option) (model.Value, prometheusv1.Warnings, error) {
	return s.queryResponse.value, nil, s.queryResponse.err
}

func (s staticPrometheusQuerier) QueryRange(ctx context.Context, query string, r prometheusv1.Range, options ...prometheusv1.Option) (model.Value, prometheusv1.Warnings, error) {
	return s.queryRangeResponse.value, nil, s.queryRangeResponse.err
}

// promMatrix creates a matrix from the given sample streams.
func promMatrix(ss ...model.SampleStream) model.Value {
	m := make(model.Matrix, len(ss))
	for i := range ss {
		m[i] = &ss[i]
	}
	return m
}

// explodeValues explodes a string of the form "3x0.5 0.7 2x0.1" into a slice of float64
// [0.5, 0.5, 0.5, 0.7, 0.1, 0.1]
func explodeValues(t *testing.T, s string) []float64 {
	t.Helper()

	parts := strings.Split(s, " ")
	values := make([]float64, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		repeat, val, err := parseValue(p)
		require.NoErrorf(t, err, "failed to parse value %q", p)
		values = append(values, slices.Repeat([]float64{val}, repeat)...)
	}
	return values
}

// parseValue parses a value of the form "3x0.5" or "0.5" into (3, 0.5) or (1, 0.5)
func parseValue(s string) (int, float64, error) {
	before, after, found := strings.Cut(s, "x")
	if !found {
		val, err := strconv.ParseFloat(s, 64)
		return 1, val, err
	}
	r, err := strconv.Atoi(before)
	if err != nil {
		return 0, 0, err
	}
	f, err := strconv.ParseFloat(after, 64)
	return r, f, err
}

// promSampleStream creates a sample stream with samples starting at `from`
// and increasing by 1 hour for each value in `values`.
func promSampleStream(t *testing.T, m model.Metric, from time.Time, values string) model.SampleStream {
	t.Helper()

	exploded := explodeValues(t, values)
	sps := make([]model.SamplePair, len(exploded))
	for i := range exploded {
		sps[i] = model.SamplePair{
			Value:     model.SampleValue(exploded[i]),
			Timestamp: model.TimeFromUnixNano(from.Add(time.Duration(i+1) * time.Hour).UnixNano()),
		}
	}
	t.Logf("Exploded values for %v: %v", m, sps)
	return model.SampleStream{
		Metric: m,
		Values: sps,
	}
}

func mustTimeFromRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return v
}

func sloErrorMetric(slothService string) model.Metric {
	return model.Metric{
		"__name__":      "slo:sli_error:ratio_rate1h",
		"sloth_service": model.LabelValue(slothService),
	}
}
