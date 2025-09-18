package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/vshn/vshn-sli-reporting/pkg/api/downtime"
	"github.com/vshn/vshn-sli-reporting/pkg/api/handler"
	"github.com/vshn/vshn-sli-reporting/pkg/api/query"
)

type ApiServerConfig struct {
	AuthUser string
	AuthPass string
	Port     int
	Host     string

	Logger *logr.Logger
}

type ApiServer struct {
	config ApiServerConfig
	mux    *http.ServeMux
	store  downtime.DowntimeStore
	server *http.Server
}

func NewApiServer(config ApiServerConfig, store downtime.DowntimeStore, prom query.PrometheusQuerier) ApiServer {
	if config.Logger == nil {
		l := logr.Discard()
		config.Logger = &l
	}
	var mux = http.NewServeMux()
	mux.Handle("/", handler.JSONFunc(func(r *http.Request) (any, error) {
		return nil, handler.NewErrWithCode(errors.New("not found"), http.StatusNotFound)
	}))
	downtime.Setup(mux, store)
	query.Setup(mux, store, prom)
	return ApiServer{
		config: config,
		mux:    mux,
		store:  store,
	}
}

func (s *ApiServer) Start() error {
	var hostport = fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	var server = http.Server{
		Addr:    hostport,
		Handler: s.logInject(s.basicAuth(s.mux)),
	}
	s.server = &server
	s.config.Logger.Info("Listening on", "addr", hostport)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *ApiServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *ApiServer) logInject(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := s.config.Logger.WithValues(
			"method", r.Method, "url", r.URL.String(), "remote", r.RemoteAddr,
			"request_id", r.Header.Get("X-Request-ID"), "internal_request_id", uuid.NewString(),
			"user_agent", r.UserAgent(),
		)
		ctx := logr.NewContext(r.Context(), logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		logr.FromContextOrDiscard(r.Context()).Info("Unauthorized request", "username", username)
	})
}
