package admin

import (
	"net/http"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"github.com/go-chi/render"
)

type flushResponse struct {
	Status string `json:"status"`
}

func (s *Server) invalidateCache(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	isAdmin := server.AdminFromContext(ctx)

	if !isAdmin {
		render.JSON(w, r, flushResponse{Status: "nok"})
		return nil
	}

	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})
	log.Info("flush cache")
	c.Flush()

	render.JSON(w, r, flushResponse{Status: "ok"})
	return nil
}
