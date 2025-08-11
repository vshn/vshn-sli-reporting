package downtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type downtimeServer struct {
	store DowntimeStore
}

type DowntimeStore interface {
	InitializeDB() error
	CloseDB() error
	StoreNewWindow(*types.DowntimeWindow) (*types.DowntimeWindow, error)
	ListWindows(from time.Time, to time.Time) ([]*types.DowntimeWindow, error)
	ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]*types.DowntimeWindow, error)
	UpdateWindow(*types.DowntimeWindow) (*types.DowntimeWindow, error)
	PatchWindow(*types.DowntimeWindow) (*types.DowntimeWindow, error)
}

func (s *downtimeServer) ListDowntime(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `from` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `to` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	ws, err := s.store.ListWindows(ft, tt)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not list downtime windows (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ws)
}

func (s *downtimeServer) ListDowntimeForCluster(w http.ResponseWriter, r *http.Request) {
	clusterId := r.PathValue("clusterid")

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `from` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not parse `to` time (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	ws, err := s.store.ListWindowsMatchingClusterFacts(r.Context(), ft, tt, clusterId)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not list downtime windows (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ws)

}

func (s *downtimeServer) CreateDowntime(w http.ResponseWriter, r *http.Request) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Invalid downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	ws, err := s.store.StoreNewWindow(&window)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		http.Error(w, fmt.Sprintf("Error: Could not store downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (s *downtimeServer) UpdateDowntime(w http.ResponseWriter, r *http.Request) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Invalid downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	window.ID = r.PathValue("id")

	ws, err := s.store.UpdateWindow(&window)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not update downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ws)
}

func (s *downtimeServer) PatchDowntime(w http.ResponseWriter, r *http.Request) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Invalid downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	window.ID = r.PathValue("id")

	ws, err := s.store.PatchWindow(&window)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: Could not patch downtime window (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ws)
}

func Setup(mux *http.ServeMux, store DowntimeStore) {
	s := downtimeServer{store}
	log.Println("Registering endpoints")
	mux.HandleFunc("GET /downtime", s.ListDowntime)
	mux.HandleFunc("GET /downtime/cluster/{clusterid}", s.ListDowntimeForCluster)
	mux.HandleFunc("POST /downtime", s.CreateDowntime)
	mux.HandleFunc("POST /downtime/{id}", s.UpdateDowntime)
	mux.HandleFunc("PATCH /downtime/{id}", s.PatchDowntime)
}
