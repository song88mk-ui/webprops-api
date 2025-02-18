package server

import (
	"context"
	"net/http"
	"strings"

	"s88/wp/backend/log"

	"github.com/go-chi/render"
)

func MustClient(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		clientId := r.Header.Get("X-Client-Id")
		if clientId == "" {
			log.Errorf("invalid client id: %s", clientId)
			render.Status(r, errCodeStatuses[ErrMissingClientId])
			render.JSON(w, r, APIError{
				Code:    ErrMissingClientId,
				Message: errCodeMessages[ErrMissingClientId],
			})
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyClientId, strings.ToUpper(clientId))
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

func ClientIdFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyClientId)
	if v == nil {
		return "", false
	}

	c, ok := v.(string)
	return c, ok
}
