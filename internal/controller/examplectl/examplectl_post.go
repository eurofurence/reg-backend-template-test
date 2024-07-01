package examplectl

import (
	"context"
	"encoding/json"
	"github.com/eurofurence/reg-backend-template-test/internal/apimodel"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/url"
)

type RequestSetExample struct {
	category string
	body     apimodel.Example
}

type ResponseEmpty struct{}

func (c *Controller) SetExample(ctx context.Context, req *RequestSetExample, w http.ResponseWriter) (*ResponseEmpty, error) {
	return nil, nil
}

func (c *Controller) SetExampleRequest(r *http.Request, w http.ResponseWriter) (*RequestSetExample, error) {
	category := chi.URLParam(r, categoryParam)

	body, err := parseExampleBody(r)
	if err != nil {
		web.SendErrorResponse(r.Context(), w, err)
		return nil, err
	}

	return &RequestSetExample{
		category: category,
		body:     body,
	}, nil
}

func (c *Controller) SetExampleResponse(ctx context.Context, res *ResponseEmpty, w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func parseExampleBody(r *http.Request) (apimodel.Example, error) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	dto := apimodel.Example{}
	err := decoder.Decode(&dto)
	if err != nil {
		return dto, common.NewBadRequest(r.Context(), common.RequestParseFailed, url.Values{"request": []string{"request body invalid"}})
	}
	return dto, nil
}
