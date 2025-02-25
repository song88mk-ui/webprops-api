package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/go-chi/render"
)

type Maintenance struct {
	ClientId      string `json:"clientId"`
	IsMaintenance bool   `json:"isMaintenance"`
	Site          string `json:"site"`
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

	// flush
	render.JSON(w, r, "ok")
	return nil
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

	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Errorf("unable to initialize azure id: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to acquire azure id"}
	}

	container := "staging"
	cEnv := os.Getenv("CONTAINER")
	if cEnv != "" {
		container = cEnv
	}

	client, err := azblob.NewClient(server.AZURE_STORAGE_ACCOUNT, creds, nil)
	if err != nil {
		log.Errorf("unable to create azure blob client: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to acquire blob"}
	}

	pager := client.NewListBlobsFlatPager(container, nil)

	var mm []Maintenance

	for pager.More() {
		pResp, err := pager.NextPage(ctx)
		if err != nil {
			return server.APIError{Code: http.StatusInternalServerError, Message: err.Error()}
		}

		for _, b := range pResp.Segment.BlobItems {

			if strings.HasPrefix(*b.Name, xLayout) {

				m, ok := getJsonMaintenance(ctx, client, container, *b.Name)

				if ok {
					mm = append(mm, m)
				}
			}

			log.Info(*b.Name)
		}
	}

	render.JSON(w, r, mm)
	return nil
}

func getJsonMaintenance(ctx context.Context, abc *azblob.Client, container string, path string) (Maintenance, bool) {
	var m Maintenance
	resp, err := abc.DownloadStream(ctx, container, path, nil)
	if err != nil {
		return m, false
	}

	buf := bytes.Buffer{}
	retryReader := resp.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = buf.ReadFrom(retryReader)

	if err != nil {
		return m, false
	}

	err = retryReader.Close()
	if err != nil {
		return m, false
	}

	var data map[string]interface{}

	err = json.Unmarshal(buf.Bytes(), &data)

	if err != nil {
		return m, false
	}
	if v, ok := data["clientId"].(string); ok {
		m.ClientId = v
	}

	if v, ok := data["site"].(string); ok {
		m.Site = v
	}

	if v, ok := data["isMaitenancen"].(bool); ok {
		m.IsMaintenance = v
	}

	return m, true
}
