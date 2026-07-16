package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/render"
)

type Maintenance struct {
	ClientId      string `json:"clientId"`
	IsMaintenance bool   `json:"isMaintenance"`
	Site          string `json:"site"`
}

type Payload struct {
	ClientIds     string `json:"clientIds"`
	IsMaintenance bool   `json:"isMaintenance"`
}

func newS3Client(ctx context.Context) (*s3.Client, string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Errorf("unable to load SDK config: %v", err)
		return nil, "", errors.Wrap(err, "failed to load AWS config")
	}

	return s3.NewFromConfig(cfg), os.Getenv("BUCKET_NAME"), nil
}

func (s *Server) getMaintenance(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	isAdmin := server.AdminFromContext(ctx)

	if !isAdmin {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	xLayout, _ := server.LayoutFromContext(ctx)

	if xLayout == "" {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	s3Client, bucket, err := newS3Client(ctx)
	if err != nil {
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to load AWS config"}
	}

	prefix := xLayout + "/"

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	mm := []Maintenance{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return server.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
		}

		for _, obj := range page.Contents {
			key := *obj.Key
			if !strings.HasSuffix(key, ".json") {
				continue
			}

			m, ok := getJsonMaintenance(ctx, s3Client, bucket, key)
			if ok {
				if m.ClientId == "" {
					m.ClientId = clientIdFromKey(key, prefix)
				}
				mm = append(mm, m)
			}
		}
	}

	render.JSON(w, r, mm)
	return nil
}

func getJsonMaintenance(ctx context.Context, s3Client *s3.Client, bucket string, key string) (Maintenance, bool) {
	var m Maintenance

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Errorf("unable to get object %s: %v", key, err)
		return m, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read object %s: %v", key, err)
		return m, false
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Errorf("unable to parse object %s: %v", key, err)
		return m, false
	}

	if v, ok := data["clientId"].(string); ok {
		m.ClientId = v
	}

	if v, ok := data["site"].(string); ok {
		m.Site = v
	}

	if v, ok := data["isMaintenance"].(bool); ok {
		m.IsMaintenance = v
	}

	return m, true
}

func (s *Server) setMaintenance(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	isAdmin := server.AdminFromContext(ctx)

	if !isAdmin {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	xLayout, _ := server.LayoutFromContext(ctx)

	if xLayout == "" {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	var payload Payload
	err := render.Decode(r, &payload)
	if err != nil {
		return server.APIError{Code: http.StatusBadRequest, Message: "failed to decode payload"}
	}

	s3Client, bucket, err := newS3Client(ctx)
	if err != nil {
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to load AWS config"}
	}

	var clientIds []string

	if payload.ClientIds == "all" {
		clientIds, err = getClientIds(ctx, s3Client, bucket, xLayout)
		if err != nil {
			return server.APIError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to list clients: %v", err)}
		}
	} else {
		for _, id := range strings.Split(payload.ClientIds, ",") {
			id = strings.ToUpper(strings.TrimSpace(id))
			if id != "" {
				clientIds = append(clientIds, id)
			}
		}
	}

	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})

	for _, clientId := range clientIds {
		err := setJsonMaintenance(ctx, s3Client, bucket, xLayout, clientId, payload.IsMaintenance)
		if err != nil {
			return server.APIError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to set maintenance for %s: %v", clientId, err)}
		}
		// remove from cache so GET /properties serves the new flag
		_ = c.Delete(fmt.Sprintf("%s-%s", xLayout, clientId))
	}

	render.JSON(w, r, "ok")
	return nil
}

func setJsonMaintenance(ctx context.Context, s3Client *s3.Client, bucket string, layout string, id string, isMaintenance bool) error {
	key := fmt.Sprintf("%s/%s.json", layout, id)

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Errorf("unable to get object %s: %v", key, err)
		return errors.Wrap(err, "failed to download file")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read object %s: %v", key, err)
		return errors.Wrap(err, "failed to read file")
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		log.Errorf("unable to parse object %s: %v", key, err)
		return errors.Wrap(err, "failed to parse json")
	}

	jsonData["isMaintenance"] = isMaintenance

	updatedJson, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}

	contentType := "application/json"

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(updatedJson),
		ContentType: &contentType,
	})
	if err != nil {
		log.Errorf("unable to put object %s: %v", key, err)
		return errors.Wrap(err, "failed to upload file")
	}

	return nil
}

func getClientIds(ctx context.Context, s3Client *s3.Client, bucket string, layout string) ([]string, error) {
	var result []string

	prefix := layout + "/"

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Errorf("failed to get the list: %v", err)
			return nil, errors.Wrap(err, "failed to list objects")
		}

		for _, obj := range page.Contents {
			key := *obj.Key
			if !strings.HasSuffix(key, ".json") {
				continue
			}
			result = append(result, clientIdFromKey(key, prefix))
		}
	}

	return result, nil
}

func clientIdFromKey(key string, prefix string) string {
	return strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".json")
}
