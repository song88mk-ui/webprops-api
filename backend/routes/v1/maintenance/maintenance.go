package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
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

	container := azContainer()

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
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to decode payload"}
	}

	if payload.ClientIds == "all" {
		clientIds := getClientIds(ctx, xLayout)

		for i, _ := range clientIds {
			err := setJsonMaintenance(ctx, xLayout, clientIds[i], payload.IsMaintenance)
			if err != nil {
				return server.APIError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to set maintenance file: %v", err)}
			}
		}
	} else {

		clientIds := strings.Split(payload.ClientIds, ",")

		for i, _ := range clientIds {
			err := setJsonMaintenance(ctx, xLayout, clientIds[i], payload.IsMaintenance)
			if err != nil {
				return server.APIError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to set maintenance file: %v", err)}
			}
		}

	}

	// flush
	render.JSON(w, r, "ok")
	return nil
}

func setJsonMaintenance(ctx context.Context, layout string, id string, isMaintenance bool) error {
	container := azContainer()
	blobPath := fmt.Sprintf("%s/%s.json", layout, id)

	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Errorf("unable to initialize azure id: %v", err)
		return errors.Sentinel("failed to auth")
	}

	client, err := azblob.NewClient(server.AZURE_STORAGE_ACCOUNT, creds, nil)
	if err != nil {
		log.Errorf("unable to create azure blob client: %v", err)
		return errors.Sentinel("failed to connect to storage")
	}

	localFile, err := os.CreateTemp("", "id-*.json")
	if err != nil {
		log.Errorf("failed to create temp file: %v", err)
		return errors.Sentinel("failed to create temporary buffer")
	}
	defer localFile.Close()

	_, err = client.DownloadFile(ctx, container, blobPath, localFile, nil)

	if err != nil {
		log.Errorf("unable to download file: %v", err)
		return errors.Sentinel("failed to download file")
	}

	err = updateLocalFile(localFile, isMaintenance)
	if err != nil {
		log.Errorf("failed to update fileu: %v", err)
		return errors.Sentinel("failed to update file")
	}

	uploadFile, err := os.OpenFile(localFile.Name(), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("failed to open file for upload: %v", err)
		return errors.Sentinel("failed to open file for upload")
	}
	defer uploadFile.Close()

	_, err = client.UploadFile(ctx, container, blobPath, uploadFile, nil)
	if err != nil {
		log.Errorf("failed to upload file: %v", err)
		return errors.Sentinel("failed to upload file")
	}

	_ = os.Remove(localFile.Name())
	_ = os.Remove(uploadFile.Name())

	return nil
}

func updateLocalFile(file *os.File, isMaintenance bool) error {

	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return errors.Sentinel("failed to read file")
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return errors.Sentinel("failed to marshal json")
	}

	jsonData["isMaintenance"] = isMaintenance

	updatedJson, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return errors.Sentinel("failed to marshal json")
	}

	err = ioutil.WriteFile(file.Name(), updatedJson, 0644)
	if err != nil {
		return errors.Sentinel("failed to write file")
	}

	return nil
}

func getClientIds(ctx context.Context, layout string) []string {
	var result []string

	container := azContainer()

	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Errorf("unable to initialize azure id: %v", err)
		return result
	}

	client, err := azblob.NewClient(server.AZURE_STORAGE_ACCOUNT, creds, nil)
	if err != nil {
		log.Errorf("unable to create azure blob client: %v", err)
		return result
	}

	pager := client.NewListBlobsFlatPager(container, nil)

	for pager.More() {
		pResp, err := pager.NextPage(ctx)
		if err != nil {
			log.Errorf("failed to get the list: %v", err)
			return result
		}

		for _, b := range pResp.Segment.BlobItems {
			if strings.HasPrefix(*b.Name, layout) {
				fPaths := strings.Split(*b.Name, "/")
				id := strings.TrimSuffix(fPaths[1], ".json")
				result = append(result, id)
			}
		}
	}

	return result
}

func azContainer() string {
	container := "staging"
	cEnv := os.Getenv("CONTAINER")
	if cEnv != "" {
		container = cEnv
	}
	return container
}
