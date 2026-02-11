package backend

import (
	"s88/wp/backend/routes"
	"s88/wp/backend/routes/v1/admin"
	"s88/wp/backend/routes/v1/maintenance"
	"s88/wp/backend/routes/v1/properties"
	"s88/wp/backend/server"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func mountRoutes(srv *server.Server) {
	srv.Router.Route("/api/v1", func(router chi.Router) {
		maintenance.Mount(srv, router)
		properties.Mount(srv, router)
		admin.Mount(srv, router)
	})
	srv.Router.Get("/healthz", routes.HealthCheckHandler)
	srv.Router.Handle("/metrics", promhttp.Handler())
}
