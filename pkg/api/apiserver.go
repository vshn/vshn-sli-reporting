package api

import (
	"crypto/sha256"
	"crypto/subtle"
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
	store  store.DowntimeStore
}

func NewApiServer(config ApiServerConfig, store store.DowntimeStore) ApiServer {
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
	return http.ListenAndServe(hostport, s.basicAuth(s.mux))
}

func (s *ApiServer) basicAuth(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(s.config.AuthUser))
			expectedPasswordHash := sha256.Sum256([]byte(s.config.AuthPass))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
