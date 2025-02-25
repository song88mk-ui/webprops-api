package properties

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"s88/wp/backend/log"
	"s88/wp/backend/server"

	"code.forgejo.org/go-chi/cache"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/go-chi/render"
)

func (s *Server) getProperties(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	clientId, _ := server.ClientIdFromContext(ctx)
	layout, _ := server.LayoutFromContext(ctx)

	c, _ := cache.NewCacher(cache.Options{Adapter: "memory"})

	cacheKey := fmt.Sprintf("%s", clientId)
	filePath := fmt.Sprintf("%s.json", clientId)

	if layout != "" {
		cacheKey = fmt.Sprintf("%s-%s", layout, clientId)
		filePath = fmt.Sprintf("%s/%s.json", layout, clientId)
	}

	v := c.Get(cacheKey)
	if v != nil {
		render.JSON(w, r, v.(interface{}))
		return nil
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

	resp, err := client.DownloadStream(ctx, container, filePath, nil)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, server.APIError{Code: http.StatusNotFound, Message: fmt.Sprintf("no properties for client: %s", clientId)})
		return nil
	}

	buf := bytes.Buffer{}
	retryReader := resp.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = buf.ReadFrom(retryReader)

	if err != nil {
		log.Errorf("unable to read from stream: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "error reading file"}
	}

	err = retryReader.Close()
	if err != nil {
		log.Errorf("unable to close stream: %v", err)
		return server.APIError{Code: http.StatusInternalServerError, Message: "error closing file"}
	}

	var data map[string]interface{}

	err = json.Unmarshal(buf.Bytes(), &data)

	if err != nil {
		return server.APIError{Code: http.StatusInternalServerError, Message: "failed to parse json"}
	}

	c.Put(cacheKey, data, 60*60*24)

	render.JSON(w, r, data)
	return nil
}
