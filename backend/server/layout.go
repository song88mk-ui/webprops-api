package server

import (
	"context"
	"net/http"
	"strings"
)

const (
	AZURE_STORAGE_ACCOUNT = "https://song88webconfig.blob.core.windows.net"
)

func MaybeLayout(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		xLayout := r.Header.Get("X-Layout")
		if xLayout == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyLayout, strings.ToUpper(xLayout))
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func LayoutFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyLayout)
	if v == nil {
		return "", false
	}

	layout, ok := v.(string)
	return layout, ok
}
