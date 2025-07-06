package main

import (
	"net/http"
	"os"

	"github.com/apex/log"

	"github.com/zacsketches/deployer-service/handlers"
	"github.com/zacsketches/deployer-service/logging"
)

func main() {
	logging.Setup()

	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/deploy", handlers.DeployHandler)

	port := "8686"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	addr := ":" + port
	log.WithField("addr", addr).Info("starting deployer service")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.WithError(err).Fatal("deploy service failed")
	}
}
