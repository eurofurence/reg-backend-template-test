package main

import (
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/config"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/logging/consolelogging/logformat"
	"github.com/eurofurence/reg-backend-template-test/web"
	"log"
)

func main() {
	err := config.LoadConfiguration("config.yaml")
	if err != nil {
		log.Fatal(logformat.Logformat("ERROR", "00000000", fmt.Sprintf("Error while loading configuration: %v", err)))
	}
	log.Println(logformat.Logformat("INFO", "00000000", "Initializing..."))
	server := web.Create()
	web.Serve(server)
}
