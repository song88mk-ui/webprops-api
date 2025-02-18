package maintenance

import (
	"net/http"

	"s88/wp/backend/server"

	"github.com/go-chi/render"
)

type Maintenance struct {
	ClientId      string `json:"clientId"`
	IsMaintenance bool   `json:"isMaintenance"`
}

func (s *Server) getMaintenance(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	clientId, ok := server.ClientIdFromContext(ctx)
	if !ok {
		return server.APIError{
			Code:    http.StatusForbidden,
			Message: "Forbidden",
		}
	}

	m := Maintenance{
		ClientId:      clientId,
		IsMaintenance: false,
	}
	render.JSON(w, r, m)
	return nil
}
