package logging

import (
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/Roshick/go-autumn-slog/pkg/level"
	auslog "github.com/Roshick/go-autumn-slog/pkg/logging"
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"log/slog"
	"os"
)

const (
	LogStyleJSON  = "json"
	LogStylePlain = "plain"
)

// PreliminarySetup provides minimal structured logging before we have read the configuration.
//
// This solves the chicken and egg problem between configuration (which also configures logging)
// and logging, so errors in the configuration can be logged.
//
// After reading the configuration, you should call Setup() with it.
//
// In order to avoid log format differences in normal operation, reading the configuration
// should only write logs if it fails.
func PreliminarySetup() {
	setupJSONWithAllDefaults()
}

// Setup provides fully configured plaintext or structured logging.
//
// It also sets the default logger, so at this point even libraries that neither use slog nor aulogging
// will use our structured logger.
func Setup() error {
	aulogging.RequestIdRetriever = common.GetRequestID

	style := auconfigenv.Get(ConfLogStyle)

	switch style {
	case LogStylePlain:
		lvl, err := level.ParseLogLevel(auconfigenv.Get(ConfLogLevel))
		if err != nil {
			return err
		}

		setupPlain(lvl)
	case LogStyleJSON, "":
		if err := setupJSON(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("failed to parse log style %s, must be one of %s (default if blank), %s", style, LogStyleJSON, LogStylePlain)
	}

	return nil
}

const (
	ConfLogStyle                = "LOG_STYLE"
	ConfLogLevel                = "LOG_LEVEL"
	ConfLogTimeTransformer      = "LOG_TIME_TRANSFORMER"
	ConfLogAttributeKeyMappings = "LOG_ATTRIBUTE_KEY_MAPPINGS"
)

const ECSMapping = `{
  "time": "@timestamp",
  "level": "log.level",
  "msg": "message",
  "error": "error.message"
}`

func ConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         ConfLogStyle,
			Default:     "json",
			Description: "log style, defaults to json if not set",
			Validate:    auconfigapi.ConfigNeedsNoValidation, // validated by logging initialize
		}, {
			Key:     ConfLogLevel,
			Default: "INFO",
			Description: "Minimum level of all logs. \n" +
				"Supported values: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC, SILENT",
			Validate: auconfigapi.ConfigNeedsNoValidation, // validated by logging initialize
		}, {
			Key:         ConfLogTimeTransformer,
			Default:     "UTC",
			Description: "Type of transformation applied to each record's timestamp. Useful for testing purposes. Supported values: UTC, ZERO",
			Validate:    auconfigapi.ConfigNeedsNoValidation, // validated by logging initialize
		}, {
			Key:     ConfLogAttributeKeyMappings,
			Default: ECSMapping,
			Description: "Mappings for attribute keys of all logs. " +
				"Example: The entry [error: error.message] maps every attribute with key \"error\" to use the key \"error.message\" instead.",
			Validate: auconfigapi.ConfigNeedsNoValidation, // validated by logging initialize
		},
	}
}

func obtainDefaultValue(key string) string {
	for _, e := range ConfigItems() {
		if e.Key == key {
			return fmt.Sprintf("%v", e.Default)
		}
	}
	return ""
}

func setupJSONWithAllDefaults() {
	config := auslog.NewConfig()
	if err := config.ObtainValues(obtainDefaultValue); err != nil {
		// too bad - can't do anything here, will have broken logging
	}
}

func setupPlain(lvl slog.Level) {
	slog.SetLogLoggerLevel(lvl)
	plainLogger := slog.Default()
	aulogging.Logger = auslog.New().WithLogger(plainLogger)
}

func setupJSON() error {
	config := auslog.NewConfig()
	if err := config.ObtainValues(auconfigenv.Get); err != nil {
		return err
	}

	structuredLogger := slog.New(slog.NewJSONHandler(os.Stdout, config.HandlerOptions()))
	aulogging.Logger = auslog.New().WithLogger(structuredLogger)
	slog.SetDefault(structuredLogger)
	return nil
}
