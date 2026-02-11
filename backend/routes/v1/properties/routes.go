package properties

import (
	"s88/wp/backend/server"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	*server.Server
}

func Mount(srv *server.Server, r chi.Router) {
	s := &Server{Server: srv}

	r.Get("/config/{clientId}", server.WrapHandler(s.getConfig))

	r.With(server.MustClient).
		With(server.MaybeLayout).
		Get("/properties", server.WrapHandler(s.getProperties))
}
