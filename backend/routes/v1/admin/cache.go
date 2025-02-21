package admin

import (
	"fmt"
	"net/http"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"github.com/go-chi/chi/v5"
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

func (s *Server) invalidateClientCache(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	isAdmin := server.AdminFromContext(ctx)
	xLayout, _ := server.LayoutFromContext(ctx)

	if !isAdmin {
		render.JSON(w, r, flushResponse{Status: "nok"})
		return nil
	}

	if xLayout == "" {
		render.JSON(w, r, flushResponse{Status: "nok"})
		return nil
	}

	clientId := chi.URLParam(r, "clientId")
	cacheKey := fmt.Sprintf("%s-%s", xLayout, clientId)
	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})

	if !c.IsExist(cacheKey) {
		render.JSON(w, r, flushResponse{Status: "ok"})
		return nil
	}

	log.Info("flush client cache")
	c.Delete(cacheKey)

	render.JSON(w, r, flushResponse{Status: "ok"})
	return nil
}
