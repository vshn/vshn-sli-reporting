package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

type JSONFunc func(r *http.Request) (any, error)

func (f JSONFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l := logr.FromContextOrDiscard(r.Context())
	start := time.Now()
	statusCode := http.StatusOK
	defer func() {
		l.Info("Request completed", "duration", time.Since(start), "status", statusCode)
	}()

	result, err := f(r)
	if err != nil {
		statusCode = http.StatusInternalServerError
		cr := ErrWithCode{}
		if errors.As(err, &cr) {
			statusCode = cr.Code
		}
		http.Error(w, err.Error(), statusCode)
		l.Error(err, "Failed to process request", "status", statusCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if rwc, ok := result.(ResponseWithCode); ok {
		statusCode = rwc.Code
		w.WriteHeader(rwc.Code)
		result = rwc.Data
	} else if rwc, ok := result.(*ResponseWithCode); ok {
		statusCode = rwc.Code
		w.WriteHeader(rwc.Code)
		result = rwc.Data
	}

	// We can't return any error as the response might be already partially written
	if err := json.NewEncoder(w).Encode(result); err != nil {
		l.Error(err, "Failed to write response")
	}
}

type ErrWithCode struct {
	Err  error
	Code int
}

func (e ErrWithCode) Unwrap() error {
	return e.Err
}

func (e ErrWithCode) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Err.Error())
}

func NewErrWithCode(err error, code int) ErrWithCode {
	return ErrWithCode{
		Err:  err,
		Code: code,
	}
}

type ResponseWithCode struct {
	Data any
	Code int
}
