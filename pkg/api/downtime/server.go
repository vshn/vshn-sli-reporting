package downtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/api/handler"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type downtimeServer struct {
	store DowntimeStore
}

type DowntimeStore interface {
	StoreNewWindow(types.DowntimeWindow) (types.DowntimeWindow, error)
	ListWindows(from time.Time, to time.Time) ([]types.DowntimeWindow, error)
	ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]types.DowntimeWindow, error)
	UpdateWindow(types.DowntimeWindow) (types.DowntimeWindow, error)
	PatchWindow(types.DowntimeWindow) (types.DowntimeWindow, error)
}

func (s *downtimeServer) ListDowntime(r *http.Request) (any, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not parse `from` time: %w", err), http.StatusBadRequest)
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not parse `to` time: %w", err), http.StatusBadRequest)
	}

	ws, err := s.store.ListWindows(ft, tt)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not list downtime windows: %w", err), http.StatusBadRequest)
	}

	return ws, nil
}

func (s *downtimeServer) ListDowntimeForCluster(r *http.Request) (any, error) {
	clusterId := r.PathValue("clusterid")

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not parse `from` time: %w", err), http.StatusBadRequest)
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not parse `to` time: %w", err), http.StatusBadRequest)
	}

	ws, err := s.store.ListWindowsMatchingClusterFacts(r.Context(), ft, tt, clusterId)

	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not list downtime windows: %w", err), http.StatusBadRequest)
	}

	return ws, nil
}

func (s *downtimeServer) CreateDowntime(r *http.Request) (any, error) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("invalid downtime window: %w", err), http.StatusBadRequest)
	}

	ws, err := s.store.StoreNewWindow(window)

	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not store downtime window: %w", err), http.StatusBadRequest)
	}

	return handler.ResponseWithCode{Data: ws, Code: http.StatusCreated}, nil
}

func (s *downtimeServer) UpdateDowntime(r *http.Request) (any, error) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("invalid downtime window: %w", err), http.StatusBadRequest)
	}

	window.ID = r.PathValue("id")
	ws, err := s.store.UpdateWindow(window)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not update downtime window: %w", err), http.StatusBadRequest)
	}

	return ws, nil
}

func (s *downtimeServer) PatchDowntime(r *http.Request) (any, error) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("invalid downtime window: %w", err), http.StatusBadRequest)
	}

	window.ID = r.PathValue("id")
	ws, err := s.store.PatchWindow(window)
	if err != nil {
		return nil, handler.NewErrWithCode(fmt.Errorf("could not patch downtime window: %w", err), http.StatusBadRequest)
	}

	return ws, nil
}

func Setup(mux *http.ServeMux, store DowntimeStore) {
	s := downtimeServer{store}
	mux.Handle("GET /downtime", handler.JSONFunc(s.ListDowntime))
	mux.Handle("GET /downtime/cluster/{clusterid}", handler.JSONFunc(s.ListDowntimeForCluster))
	mux.Handle("POST /downtime", handler.JSONFunc(s.CreateDowntime))
	mux.Handle("POST /downtime/{id}", handler.JSONFunc(s.UpdateDowntime))
	mux.Handle("PATCH /downtime/{id}", handler.JSONFunc(s.PatchDowntime))
}
