package v1health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eurofurence/reg-backend-template-test/internal/logging"
	"github.com/eurofurence/reg-backend-template-test/internal/restapi/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
)

func Create(server chi.Router) {
	server.Get("/", healthGet)
}

func healthGet(w http.ResponseWriter, r *http.Request) {
	logging.Ctx(r.Context()).Info("health")

	dto := HealthResultDto{Status: "up"}

	w.Header().Add(headers.ContentType, media.ContentTypeApplicationJson)
	w.WriteHeader(http.StatusOK)
	writeJson(r.Context(), w, dto)
}

func writeJson(ctx context.Context, w http.ResponseWriter, v interface{}) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	if err != nil {
		logging.Ctx(ctx).Warn(fmt.Sprintf("error while encoding json response: %v", err))
	}
}
