package properties

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	clientId := chi.URLParam(r, "clientId")

	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})

	cacheKey := fmt.Sprintf("%s", clientId)
	filePath := fmt.Sprintf("%s.json", clientId)

	v := c.Get(cacheKey)
	if v != nil {
		render.JSON(w, r, v.(interface{}))
		return nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Errorf("unable to load SDK config: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to load AWS config"}
	}

	s3Client := s3.NewFromConfig(cfg)

	bucket := os.Getenv("BUCKET_NAME")

	log.Infof("retrieving: %s:%s", bucket, filePath)

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &filePath,
	})

	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, server.APIError{Code: http.StatusNotFound, Message: fmt.Sprintf("no properties for client: %s", clientId)})
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read from stream: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "error reading file"}
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)

	if err != nil {
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to parse json"}
	}

	c.Put(cacheKey, data, 60*60*24)

	render.JSON(w, r, data)
	return nil
}
