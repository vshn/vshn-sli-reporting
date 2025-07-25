package api

import (
	"fmt"
	"net/http"

	"github.com/vshn/vshn-sli-reporting/pkg/api/downtime"
	"github.com/vshn/vshn-sli-reporting/pkg/store"
)

type ApiServerConfig struct {
	AuthUser string
	AuthPass string
	Port     int
	Host     string
}

type ApiServer struct {
	config ApiServerConfig
	mux    *http.ServeMux
	store  *store.DowntimeStore
}

func NewApiServer(config ApiServerConfig, store *store.DowntimeStore) ApiServer {
	var mux = http.NewServeMux()
	downtime.Setup(mux, store)
	return ApiServer{
		config: config,
		mux:    mux,
		store:  store,
	}
}

func (s *ApiServer) Start() error {
	var hostport = fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	return http.ListenAndServe(hostport, s.mux)
}
