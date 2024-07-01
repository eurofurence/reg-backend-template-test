package examplectl

import (
	"context"
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/apimodel"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"net/http"
	"net/url"
	"strconv"
)

const minValueParam = "min_value"

type RequestGetExample struct {
	minValue int64
}

func (c *Controller) GetExample(ctx context.Context, req *RequestGetExample, w http.ResponseWriter) (*apimodel.Example, error) {
	val, err := c.svc.ObtainNextValue(ctx, req.minValue)
	if err != nil {
		web.SendErrorResponse(ctx, w, err)
		return nil, err
	}

	return &apimodel.Example{
		Value: val,
	}, nil
}

func (c *Controller) GetExampleRequest(r *http.Request, w http.ResponseWriter) (*RequestGetExample, error) {
	minValue, err := parseIntQueryParam(r, minValueParam)
	if err != nil {
		web.SendErrorResponse(r.Context(), w, err)
		return nil, err
	}

	return &RequestGetExample{
		minValue: minValue,
	}, nil
}

func (c *Controller) GetExampleResponse(ctx context.Context, res *apimodel.Example, w http.ResponseWriter) error {
	return web.EncodeWithStatus(http.StatusOK, res, w)
}

func parseIntQueryParam(r *http.Request, name string) (int64, error) {
	valueStr := r.URL.Query().Get(name)
	if valueStr != "" {
		val, err := strconv.Atoi(valueStr)
		if err != nil {
			return 0, common.NewBadRequest(r.Context(), common.RequestParseFailed, url.Values{"request": []string{fmt.Sprintf("parameter %s invalid - must be a valid integer", name)}})
		}
		return int64(val), nil
	} else {
		return 0, nil
	}
}
