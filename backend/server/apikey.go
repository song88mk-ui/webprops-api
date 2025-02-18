package server

import (
	"context"
	"net/http"
	"os"
)

func MustApiKey(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("CACHE_API_KEY")
		xApiKey := r.Header.Get("X-Api-Key")
		if xApiKey == "" || xApiKey != apiKey {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyApiKey, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func AdminFromContext(ctx context.Context) bool {
	v := ctx.Value(ctxKeyApiKey)
	if v == nil {
		return false
	}

	isAdmin, ok := v.(bool)
	return isAdmin && ok
}
