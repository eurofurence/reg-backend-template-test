package infoctl

import (
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Controller struct{}

func InitRoutes(router chi.Router) {
	ctl := &Controller{}

	router.Route("/", func(sr chi.Router) {
		initGetRoutes(sr, ctl)
	})
}

func initGetRoutes(router chi.Router, c *Controller) {
	router.Method(
		http.MethodGet,
		"/",
		web.CreateHandler(
			c.Health,
			c.HealthRequest,
			c.HealthResponse,
		),
	)
}
