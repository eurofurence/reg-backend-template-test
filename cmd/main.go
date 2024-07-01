package main

import (
	"github.com/eurofurence/reg-backend-template-test/internal/application/app"
	"os"
)

func main() {
	os.Exit(app.New().Run())
}
