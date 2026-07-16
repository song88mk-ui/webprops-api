package properties

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	isAdmin := server.AdminFromContext(ctx)

	if !isAdmin {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	clientId := chi.URLParam(r, "clientId")
	layout, _ := server.LayoutFromContext(ctx)

	cacheKey := clientId
	filePath := fmt.Sprintf("%s.json", clientId)

	if layout != "" {
		clientId = strings.ToUpper(clientId)
		cacheKey = fmt.Sprintf("%s-%s", layout, clientId)
		filePath = fmt.Sprintf("%s/%s.json", layout, clientId)
	}

	var data map[string]interface{}
	err := render.Decode(r, &data)
	if err != nil {
		return server.APIError{Code: http.StatusBadRequest, Message: "failed to decode payload"}
	}

	if len(data) == 0 {
		return server.APIError{Code: http.StatusBadRequest, Message: "empty config payload"}
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Errorf("unable to load SDK config: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to load AWS config"}
	}

	s3Client := s3.NewFromConfig(cfg)

	bucket := os.Getenv("BUCKET_NAME")

	// preserve the maintenance flag unless the payload sets it explicitly
	if _, ok := data["isMaintenance"]; !ok {
		if v, ok := currentMaintenanceFlag(r, s3Client, bucket, filePath); ok {
			data["isMaintenance"] = v
		}
	}

	updatedJson, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to marshal json"}
	}

	log.Infof("updating: %s:%s", bucket, filePath)

	contentType := "application/json"

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &filePath,
		Body:        bytes.NewReader(updatedJson),
		ContentType: &contentType,
	})
	if err != nil {
		log.Errorf("unable to put object %s: %v", filePath, err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to upload config"}
	}

	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})
	_ = c.Delete(cacheKey)

	render.JSON(w, r, "ok")
	return nil
}

func currentMaintenanceFlag(r *http.Request, s3Client *s3.Client, bucket string, key string) (bool, bool) {
	ctx := r.Context()

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, false
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(body, &existing); err != nil {
		return false, false
	}

	v, ok := existing["isMaintenance"].(bool)
	return v, ok
}
