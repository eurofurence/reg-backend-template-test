package infoctl

import (
	"context"
	"github.com/eurofurence/reg-backend-template-test/internal/apimodel"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"net/http"
)

type HealthRequest struct{}

func (c *Controller) Health(ctx context.Context, req *HealthRequest, w http.ResponseWriter) (*apimodel.Health, error) {
	return &apimodel.Health{Status: "OK"}, nil
}

func (c *Controller) HealthRequest(r *http.Request, w http.ResponseWriter) (*HealthRequest, error) {
	return &HealthRequest{}, nil
}

func (c *Controller) HealthResponse(ctx context.Context, res *apimodel.Health, w http.ResponseWriter) error {
	return web.EncodeWithStatus(http.StatusOK, res, w)
}
