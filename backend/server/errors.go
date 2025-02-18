package server

import (
	"context"
	"fmt"
	"net/http"

	"s88/wp/backend/log"

	"emperror.dev/errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func WrapHandler(hn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := hn(w, r)
		if err != nil {

			if apiErr, ok := err.(APIError); ok {
				apiErr.prepare()

				render.Status(r, apiErr.Status)
				render.JSON(w, r, apiErr)
				return
			}

			rctx := chi.RouteContext(r.Context())

			var eventId string
			eventId = "0"

			if isExpectedError(err) {
				log.Infof("expected error in handler for %v %v, ignoring", rctx.RouteMethod, rctx.RoutePattern())
			} else {
				log.Errorf("error in handler for %v %v: %v", rctx.RouteMethod, rctx.RoutePattern(), err)
			}

			apiErr := APIError{ID: eventId, Code: ErrInternalServerError}
			apiErr.prepare()

			render.Status(r, apiErr.Status)
			render.JSON(w, r, apiErr)
		}
	}
}

func isExpectedError(err error) bool {
	return errors.Is(err, context.Canceled)
}

type APIError struct {
	Code int    `json:"code"`
	ID   string `json:"id,omitempty"`

	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`

	Status int `json:"-"`
}

func (e APIError) Error() string {
	if e.Message == "" {
		e.Message = errCodeMessages[e.Code]
	}

	if e.Details != "" {
		return fmt.Sprintf("%s (code: %d) (%s)", e.Message, e.Code, e.Details)
	}

	return fmt.Sprintf("%s (code: %d)", e.Message, e.Code)
}

func (e *APIError) prepare() {
	if e.Status == 0 {
		e.Status = errCodeStatuses[e.Code]
	}

	if e.Message == "" {
		e.Message = errCodeMessages[e.Code]
	}
}

const (
	ErrBadRequest          = 400
	ErrForbidden           = 403
	ErrNotFound            = 404
	ErrMethodNotAllowed    = 405
	ErrTooManyRequests     = 429
	ErrInternalServerError = 500

	ErrRequestTooBig   = 4001
	ErrMissingClientId = 4002
)

var errCodeMessages = map[int]string{
	ErrBadRequest:          "Bad Request",
	ErrForbidden:           "Forbidden",
	ErrNotFound:            "Not found",
	ErrMethodNotAllowed:    "Method not allowed",
	ErrTooManyRequests:     "Rate limit reached",
	ErrInternalServerError: "Internal server error",

	ErrRequestTooBig:   "Request too big (max 2MB)",
	ErrMissingClientId: "Missing X-Client-Id",
}

var errCodeStatuses = map[int]int{
	ErrBadRequest:          http.StatusBadRequest,
	ErrForbidden:           http.StatusForbidden,
	ErrNotFound:            http.StatusNotFound,
	ErrMethodNotAllowed:    http.StatusMethodNotAllowed,
	ErrTooManyRequests:     http.StatusTooManyRequests,
	ErrInternalServerError: http.StatusInternalServerError,

	ErrRequestTooBig:   http.StatusBadRequest,
	ErrMissingClientId: http.StatusNotFound,
}
