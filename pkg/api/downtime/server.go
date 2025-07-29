package downtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/store"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type downtimeServer struct {
	store      *store.DowntimeStore
}

func (s *downtimeServer) ListDowntime(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws, err := s.store.ListWindows(ft, tt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (s *downtimeServer) ListDowntimeForCluster(w http.ResponseWriter, r *http.Request) {
	clusterId := r.PathValue("clusterid")

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ft, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tt, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws, err := s.store.ListWindowsMatchingClusterFacts(r.Context(), ft, tt, clusterId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)

}

func (s *downtimeServer) CreateDowntime(w http.ResponseWriter, r *http.Request) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws, err := s.store.StoreNewWindow(&window)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	window.ID = r.PathValue("id")

	ws, err := s.store.UpdateWindow(&window)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (s *downtimeServer) PatchDowntime(w http.ResponseWriter, r *http.Request) {
	window := types.DowntimeWindow{}
	err := json.NewDecoder(r.Body).Decode(&window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	window.ID = r.PathValue("id")

	ws, err := s.store.PatchWindow(&window)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func Setup(mux *http.ServeMux, store *store.DowntimeStore) {
	s := downtimeServer{store}
	fmt.Println("Registering endpoints")
	mux.HandleFunc("GET /downtime", s.ListDowntime)
	mux.HandleFunc("GET /downtime/cluster/{clusterid}", s.ListDowntimeForCluster)
	mux.HandleFunc("POST /downtime", s.CreateDowntime)
	mux.HandleFunc("POST /downtime/{id}", s.UpdateDowntime)
	mux.HandleFunc("PATCH /downtime/{id}", s.PatchDowntime)
}
