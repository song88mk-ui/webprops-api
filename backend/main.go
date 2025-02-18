package backend

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"github.com/go-chi/render"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "web",
	Usage:  "Run the API server",
	Action: run,
}

func run(c *cli.Context) error {

	port := ":" + os.Getenv("PORT")
	s, err := server.New()
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
		return nil
	}

	render.Decode = decode

	mountRoutes(s)

	e := make(chan error)

	go func() {
		e <- http.ListenAndServe(port, s.Router)
	}()

	log.Infof("api server started on port %s", port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	select {
	case <-ctx.Done():
		log.Info("interrupt signal received, shutting down...")
		return nil
	case err := <-e:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("error running server: %v", err)
		}
		return err
	}
}

const MaxContentLength = 2 * 1024 * 1024 // 2mb

func decode(r *http.Request, v any) error {
	if r.ContentLength > MaxContentLength {
		return server.APIError{
			Code: server.ErrRequestTooBig,
		}
	}

	var err error
	lr := io.LimitReader(r.Body, MaxContentLength)

	switch render.GetRequestContentType(r) {
	case render.ContentTypeJSON:
		err = render.DecodeJSON(lr, v)
	default:
		err = server.APIError{
			Code: server.ErrBadRequest,
		}
	}

	return err
}
