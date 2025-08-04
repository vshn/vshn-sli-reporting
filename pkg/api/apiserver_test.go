package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/vshn-sli-reporting/pkg/store/mock"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

var config = ApiServerConfig{
	AuthUser: "admin",
	AuthPass: "pass",
	Port:     8080,
	Host:     "localhost",
}

func setup(rv *types.DowntimeWindow) (*ApiServer, *mock.MockDowntimeStore) {
	store := &mock.MockDowntimeStore{
		ReturnValue: rv,
	}

	server := NewApiServer(config, store)
	return &server, store
}

func TestBasicAuthSucceeds(t *testing.T) {
	serv, mock := setup(&types.DowntimeWindow{Title: "Test1"})

	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")
	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	req.SetBasicAuth("admin", "pass")
	w := httptest.NewRecorder()

	handler := serv.basicAuth(serv.mux)

	handler(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "200 OK", res.Status)

	windows := []types.DowntimeWindow{}
	err := json.NewDecoder(res.Body).Decode(&windows)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(windows))
	assert.Equal(t, "Test1", windows[0].Title)

	assert.Equal(t, "list", mock.LastCall)
	assert.True(t, mock.LastCallFrom.Equal(time1))
	assert.True(t, mock.LastCallTo.Equal(time2))
}

func TestBasicAuthFailsIfNotSet(t *testing.T) {
	serv, _ := setup(&types.DowntimeWindow{Title: "Test1"})

	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	req.SetBasicAuth("admin", "pasfdasdfass")
	w := httptest.NewRecorder()

	handler := serv.basicAuth(serv.mux)

	handler(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "401 Unauthorized", res.Status)
}

func TestBasicAuthFailsIfInvalidCredentials(t *testing.T) {
	serv, _ := setup(&types.DowntimeWindow{Title: "Test1"})

	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	handler := serv.basicAuth(serv.mux)

	handler(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "401 Unauthorized", res.Status)
}
