package handler

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthOutput is the response body of the health endpoint.
type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"Overall health status"`
	}
}

// RegisterHealth attaches the /healthz liveness endpoint to the API.
func RegisterHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Liveness check",
		Tags:        []string{"system"},
	}, func(_ context.Context, _ *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})
}
