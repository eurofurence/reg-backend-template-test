# reg-backend-template-test

<img src="https://github.com/eurofurence/reg-backend-template-test/actions/workflows/go.yml/badge.svg" alt="test status"/>

## Overview

Purpose of the service.

## Installation

This service uses go modules to provide dependency management, see `go.mod`.

If you place this repository outside your gopath, `go build cmd/main.go` and
`go test ./...` will download all required dependencies by default.

## Generate models

In a shell or git bash in the project root, run `./api/generator/generate.sh`.

Models are checked in for convenience and change tracking.

_Note: the generator needs a current Java runtime environment._

## Running on development system

Copy the configuration template from `docs/local-config.template.yaml` to `./local-config.yaml`
and edit as needed.

Then run `go run cmd/main.go`.

## Test Coverage

In order to collect full test coverage, set go tool arguments to `-coverpkg=./internal/...`,
or manually run
```
go test -coverpkg=./internal/... ./...
```

## Architecture

Components are grouped by stereotypes:
 * controller = anything that's being interacted with from the outside
 * service = business logic
 * repository = anything that interacts with the outside
