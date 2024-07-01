package acceptance

import (
	"context"
	"github.com/eurofurence/reg-backend-template-test/internal/application/app"
	"github.com/eurofurence/reg-backend-template-test/internal/application/server"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/configuration"
	"github.com/eurofurence/reg-backend-template-test/test/mocks/idpmock"
	"net/http/httptest"
	"testing"
)

var ts *httptest.Server
var application *app.Application

func tstSetup(t *testing.T) {
	t.Helper()

	ctx := context.TODO()

	application = app.New()
	if err := configuration.Setup(); err != nil {
		t.Error("failed to read acceptance test configuration")
		t.FailNow()
	}

	// pre-populate component mocks here

	application.IDPClient = idpmock.New()

	// now duplicating application setup (see app.Application.Run()) with required changes for test server

	if err := application.SetupRepositories(ctx); err != nil {
		t.Error("failed to set up unmocked repositories")
		t.FailNow()
	}

	if err := application.SetupServices(ctx); err != nil {
		t.Error("failed to set up services")
		t.FailNow()
	}

	router, err := server.Router(ctx, application.IDPClient)
	if err != nil {
		t.Error("failed to create router")
		t.FailNow()
	}

	if err := application.SetupControllers(ctx, router); err != nil {
		t.Error("failed to set up controllers")
		t.FailNow()
	}

	ts = httptest.NewServer(router)
}

func tstShutdown() {
	ts.Close()
}
