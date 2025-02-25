package maintenance

import (
	"s88/wp/backend/server"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	*server.Server
}

func Mount(srv *server.Server, r chi.Router) {
	s := &Server{Server: srv}

	r.With(server.MustApiKey).
		With(server.MaybeLayout).Route("/maintenance", func(r chi.Router) {
		r.Get("/", server.WrapHandler(s.getMaintenance))
		r.Post("/", server.WrapHandler(s.setMaintenance))
	})
}
