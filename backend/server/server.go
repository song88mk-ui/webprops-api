package server

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	chiprom "github.com/toshi0607/chi-prometheus"
)

var (
	Tag = "latest"
)

type Server struct {
	Router *chi.Mux
}

func New() (*Server, error) {

	s := &Server{
		Router: chi.NewMux(),
	}

	s.Router.Use(middleware.RequestID)

	if os.Getenv("DEBUG") == "true" {
		s.Router.Use(middleware.Logger)
	}

	s.Router.Use(middleware.Recoverer)

	s.Router.Use(middleware.Timeout(60 * time.Second))

	s.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"HEAD", "GET", "POST", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	os.Setenv(chiprom.EnvChiPrometheusLatencyBuckets, "10,25,50,100,300,500,1000,5000")
	prom := chiprom.New("webprops")
	prom.MustRegisterDefault()
	s.Router.Use(prom.Handler)

	s.Router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, errCodeStatuses[ErrNotFound])
		render.JSON(w, r, APIError{
			Code:    ErrNotFound,
			Message: errCodeMessages[ErrNotFound],
		})
	})

	s.Router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, errCodeStatuses[ErrMethodNotAllowed])
		render.JSON(w, r, APIError{
			Code:    ErrMethodNotAllowed,
			Message: errCodeMessages[ErrMethodNotAllowed],
		})
	})

	return s, nil
}

type ctxKey int

const (
	ctxKeyClientId ctxKey = 1
	ctxKeyApiKey   ctxKey = 2
	ctxKeyLayout   ctxKey = 3
)
