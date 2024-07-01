package examplectl

import (
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"github.com/eurofurence/reg-backend-template-test/internal/service/example"
	"github.com/go-chi/chi/v5"
	"net/http"
)

const categoryParam = "category"

type Controller struct {
	svc example.Example
}

func InitRoutes(router chi.Router, svc example.Example) {
	h := &Controller{
		svc: svc,
	}

	router.Route("/api/rest/v1/example", func(sr chi.Router) {
		initGetRoutes(sr, h)
		initPostRoutes(sr, h)
	})
}

func initGetRoutes(router chi.Router, h *Controller) {
	router.Method(
		http.MethodGet,
		"/",
		web.CreateHandler(
			h.GetExample,
			h.GetExampleRequest,
			h.GetExampleResponse,
		),
	)
}

func initPostRoutes(router chi.Router, h *Controller) {
	router.Method(
		http.MethodPost,
		fmt.Sprintf("/{%s}", categoryParam),
		web.CreateHandler(
			h.SetExample,
			h.SetExampleRequest,
			h.SetExampleResponse,
		),
	)
}
