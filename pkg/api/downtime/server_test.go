package downtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/vshn-sli-reporting/pkg/store/mock"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

func setup(rv *types.DowntimeWindow) (*http.ServeMux, *mock.MockDowntimeStore) {
	store := &mock.MockDowntimeStore{
		ReturnValue: rv,
	}
	mux := http.NewServeMux()

	Setup(mux, store)
	return mux, store
}

func setupError(rv *types.DowntimeWindow) (*http.ServeMux, *mock.MockDowntimeStore) {
	mux, store := setup(rv)
	store.DoError = true
	return mux, store
}

func TestListDowntime(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{
		Title: "Test1",
	})

	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")
	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

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

func TestListDowntimeError(t *testing.T) {
	mux, mock := setupError(&types.DowntimeWindow{})
	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "list", mock.LastCall)
}

func TestListDowntimeParseError(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{})
	req := httptest.NewRequest(http.MethodGet, "/downtime?from=2020-bogsu", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	req = httptest.NewRequest(http.MethodGet, "/downtime?from=2020-01-01T00:00:00Z&to=sadfasdf", nil)
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	req = httptest.NewRequest(http.MethodGet, "/downtime", nil)
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)
}

func TestListDowntimeByCluster(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{
		Title: "Test1",
	})

	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-02-02T00:00:00Z")
	req := httptest.NewRequest(http.MethodGet, "/downtime/cluster/c-sdf?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "200 OK", res.Status)

	windows := []types.DowntimeWindow{}
	err := json.NewDecoder(res.Body).Decode(&windows)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(windows))
	assert.Equal(t, "Test1", windows[0].Title)

	assert.Equal(t, "listcluster", mock.LastCall)
	assert.True(t, mock.LastCallFrom.Equal(time1))
	assert.True(t, mock.LastCallTo.Equal(time2))
	assert.Equal(t, "c-sdf", mock.LastCallCluster)
}

func TestListDowntimeByClusterError(t *testing.T) {
	mux, mock := setupError(&types.DowntimeWindow{})
	req := httptest.NewRequest(http.MethodGet, "/downtime/cluster/c-sdf?from=2020-01-01T00:00:00Z&to=2020-02-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "listcluster", mock.LastCall)
}

func TestListDowntimeByClusterParseError(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{})
	req := httptest.NewRequest(http.MethodGet, "/downtime/cluster/c-sdf?from=2020-bogsu", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	req = httptest.NewRequest(http.MethodGet, "/downtime/cluster/c-sdf?from=2020-01-01T00:00:00Z&to=sadfasdf", nil)
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	req = httptest.NewRequest(http.MethodGet, "/downtime/cluster/c-sdf", nil)
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)
}

func TestCreateUpdatePatchDowntime(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{})

	jsonstr, err := json.Marshal(types.DowntimeWindow{
		Title: "Test1",
	})
	assert.NoError(t, err)

	// Create
	req := httptest.NewRequest(http.MethodPost, "/downtime", strings.NewReader(string(jsonstr)))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "201 Created", res.Status)
	assert.Equal(t, "create", mock.LastCall)

	window := types.DowntimeWindow{}
	err = json.NewDecoder(res.Body).Decode(&window)
	assert.NoError(t, err)

	assert.Equal(t, "Test1", window.Title)

	// Update
	req = httptest.NewRequest(http.MethodPost, "/downtime/asdf", strings.NewReader(string(jsonstr)))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "200 OK", res.Status)
	assert.Equal(t, "update", mock.LastCall)

	window = types.DowntimeWindow{}
	err = json.NewDecoder(res.Body).Decode(&window)
	assert.NoError(t, err)

	assert.Equal(t, "Test1", window.Title)
	assert.Equal(t, "asdf", window.ID)

	// Patch
	req = httptest.NewRequest(http.MethodPatch, "/downtime/asdf2", strings.NewReader(string(jsonstr)))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "200 OK", res.Status)
	assert.Equal(t, "patch", mock.LastCall)

	window = types.DowntimeWindow{}
	err = json.NewDecoder(res.Body).Decode(&window)
	assert.NoError(t, err)

	assert.Equal(t, "Test1", window.Title)
	assert.Equal(t, "asdf2", window.ID)
}

func TestCreateUpdatePatchDowntimeErrors(t *testing.T) {
	mux, mock := setupError(&types.DowntimeWindow{})

	jsonstr, err := json.Marshal(types.DowntimeWindow{
		Title: "Test1",
	})
	assert.NoError(t, err)

	// Create
	req := httptest.NewRequest(http.MethodPost, "/downtime", strings.NewReader(string(jsonstr)))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.NotEqual(t, "201 Created", res.Status)
	assert.Equal(t, "create", mock.LastCall)

	// Update
	req = httptest.NewRequest(http.MethodPost, "/downtime/asfd", strings.NewReader(string(jsonstr)))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.NotEqual(t, "200 OK", res.Status)
	assert.Equal(t, "update", mock.LastCall)

	// Patch
	req = httptest.NewRequest(http.MethodPatch, "/downtime/asfd", strings.NewReader(string(jsonstr)))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.NotEqual(t, "200 OK", res.Status)
	assert.Equal(t, "patch", mock.LastCall)

}

func TestCreateUpdatePatchDowntimeParseErrors(t *testing.T) {
	mux, mock := setup(&types.DowntimeWindow{})

	// Create
	req := httptest.NewRequest(http.MethodPost, "/downtime", strings.NewReader("NotValidJson[]"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	// Update
	req = httptest.NewRequest(http.MethodPost, "/downtime/sdf", strings.NewReader("NotValidJson[]"))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)

	// Patch
	req = httptest.NewRequest(http.MethodPatch, "/downtime/sdf", strings.NewReader("NotValidJson[]"))
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, req)
	res = w.Result()
	defer res.Body.Close()

	assert.Equal(t, "400 Bad Request", res.Status)
	assert.Equal(t, "", mock.LastCall)
}
