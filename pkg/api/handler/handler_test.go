package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonglil/buflogr"
)

func TestJSONFunc(t *testing.T) {
	tests := []struct {
		name string

		handlerResult any
		handlerError  error

		matchBody  string
		wantStatus string
		matchLog   string
	}{
		{
			name:          "valid request",
			handlerResult: "test",

			wantStatus: "200 OK",
			matchBody:  "\"test\"\n",
			matchLog:   "Request completed",
		},
		{
			name:         "handler error without code",
			handlerError: fmt.Errorf("some error"),

			wantStatus: "500 Internal Server Error",
			matchBody:  "some error",
			matchLog:   "Failed to process request",
		},
		{
			name:         "handler error with code",
			handlerError: NewErrWithCode(fmt.Errorf("some error"), http.StatusTeapot),

			wantStatus: "418 I'm a teapot",
			matchBody:  "some error",
			matchLog:   "Failed to process request",
		},
		{
			name:          "handler returns ResponseWithCode",
			handlerResult: ResponseWithCode{Data: "test", Code: http.StatusCreated},

			wantStatus: "201 Created",
			matchBody:  "\"test\"\n",
			matchLog:   "Request completed",
		},
		{
			name:          "handler returns *ResponseWithCode",
			handlerResult: &ResponseWithCode{Data: "test", Code: http.StatusCreated},

			wantStatus: "201 Created",
			matchBody:  "\"test\"\n",
			matchLog:   "Request completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			var buf bytes.Buffer
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(logr.NewContext(t.Context(), buflogr.NewWithBuffer(&buf)))
			JSONFunc(func(r *http.Request) (any, error) {
				return tt.handlerResult, tt.handlerError
			}).ServeHTTP(rr, req)
			res := rr.Result()
			assert.Equal(t, tt.wantStatus, res.Status)
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.matchBody)
			assert.Contains(t, buf.String(), tt.matchLog)
		})
	}
}
