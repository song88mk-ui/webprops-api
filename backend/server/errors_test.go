package server

import (
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "known code without details fills message from map",
			err:  APIError{Code: ErrMissingClientId},
			want: "Missing X-Client-Id (code: 4002)",
		},
		{
			name: "explicit message overrides the map",
			err:  APIError{Code: ErrBadRequest, Message: "custom message"},
			want: "custom message (code: 400)",
		},
		{
			name: "details are appended when present",
			err:  APIError{Code: ErrNotFound, Details: "client xyz"},
			want: "Not found (code: 404) (client xyz)",
		},
		{
			// Unmapped codes have no entry in errCodeMessages, so the message
			// renders empty. This documents current behavior, not desired behavior.
			name: "unmapped code renders an empty message",
			err:  APIError{Code: 9999},
			want: " (code: 9999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_prepare(t *testing.T) {
	tests := []struct {
		name       string
		err        APIError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "fills status and message from code maps when unset",
			err:        APIError{Code: ErrForbidden},
			wantStatus: http.StatusForbidden,
			wantMsg:    "Forbidden",
		},
		{
			name:       "does not overwrite an explicitly set status",
			err:        APIError{Code: ErrForbidden, Status: http.StatusTeapot},
			wantStatus: http.StatusTeapot,
			wantMsg:    "Forbidden",
		},
		{
			name:       "does not overwrite an explicitly set message",
			err:        APIError{Code: ErrRequestTooBig, Message: "custom"},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "custom",
		},
		{
			// Unmapped codes have no entry in errCodeStatuses, so Status is left
			// at Go's zero value (0) rather than falling back to e.g. 500. This
			// documents current behavior, not desired behavior.
			name:       "unmapped code leaves status at zero value",
			err:        APIError{Code: 9999},
			wantStatus: 0,
			wantMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.err
			e.prepare()

			if e.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", e.Status, tt.wantStatus)
			}
			if e.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", e.Message, tt.wantMsg)
			}
		})
	}
}
