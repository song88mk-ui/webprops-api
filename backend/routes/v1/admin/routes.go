package admin

import (
	"s88/wp/backend/server"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	*server.Server
}

func Mount(srv *server.Server, r chi.Router) {
	s := &Server{Server: srv}

	r.Handle("/metrics", promhttp.Handler())

	// cache
	r.With(server.MustApiKey).With(server.MaybeLayout).Route("/cache", func(r chi.Router) {
		r.Post("/flush", server.WrapHandler(s.invalidateCache))
		r.Delete("/flush/{clientId}", server.WrapHandler(s.invalidateClientCache))
	})
}
