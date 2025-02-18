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

	r.With(server.MustClient).
		Get("/maintenance", server.WrapHandler(s.getMaintenance))
}
