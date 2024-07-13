package app

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-backend-template-test/internal/application/server"
	"github.com/eurofurence/reg-backend-template-test/internal/controller/examplectl"
	"github.com/eurofurence/reg-backend-template-test/internal/controller/infoctl"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/configuration"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/idp"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/logging"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/vault"
	"github.com/eurofurence/reg-backend-template-test/internal/service/example"
	"github.com/go-chi/chi/v5"
)

// Application is the main application.
//
// It collects components, but only the ones that it actually needs to keep track of.
//
// Application is responsible for wiring up the application, so this is the place
// where you need to pay attention to dependencies between components.
//
// Note that individual components are expected to take care of logging. This code
// is supposed to be as easy as possible to read.
type Application struct {
	// repositories
	Vault     vault.Vault
	IDPClient idp.IdentityProviderClient

	// services
	Example example.Example

	// controllers

	// servers
}

func New() *Application {
	return &Application{}
}

func (a *Application) Run() int {
	ctx := context.Background()

	if err := a.SetupConfigurationAndLogging(); err != nil {
		return 1
	}

	if err := a.SetupRepositories(ctx); err != nil {
		return 2
	}

	if err := a.SetupServices(ctx); err != nil {
		return 3
	}

	router, err := server.Router(ctx, a.IDPClient)
	if err != nil {
		return 4
	}

	if err := a.SetupControllers(ctx, router); err != nil {
		return 5
	}

	if err := server.Serve(ctx, router); err != nil {
		return 6
	}

	return 0
}

func (a *Application) SetupConfigurationAndLogging() error {
	logging.PreliminarySetup()

	if err := configuration.Setup(); err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Print("failed to obtain configuration - BAILING OUT: %v", err)
		return err
	}
	if err := logging.Setup(); err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Print("failed to configure logging - BAILING OUT: %v", err)
		return err
	}

	aulogging.Logger.NoCtx().Info().Print("successfully obtained configuration and set up logging")
	return nil
}

func (a *Application) SetupRepositories(ctx context.Context) error {
	if a.Vault == nil {
		a.Vault = vault.New()
	}

	if err := a.Vault.Authenticate(ctx); err != nil {
		return err
	}
	if err := a.Vault.ObtainSecrets(ctx); err != nil {
		return err
	}

	if a.IDPClient == nil {
		options := idp.OptionsFromConfig()
		a.IDPClient = idp.New(options)
	}

	if err := a.IDPClient.SetupFromWellKnown(ctx); err != nil {
		return err
	}

	return nil
}

func (a *Application) SetupServices(ctx context.Context) error {
	if a.Example == nil {
		a.Example = example.New()
	}

	return nil
}

func (a *Application) SetupControllers(ctx context.Context, router chi.Router) error {
	examplectl.InitRoutes(router, a.Example)
	infoctl.InitRoutes(router)
	return nil
}
