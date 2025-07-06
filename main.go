package main

import (
	"net/http"
	"os"

	"github.com/apex/log"

	"github.com/zacsketches/deployer-service/handlers"
	"github.com/zacsketches/deployer-service/logging"
)

func init() {
	JWTKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if JWTKeyPath == "" {
		log.Fatal("JWT_PUBLIC_KEY_PATH environment variable not set; aborting startup")
	}
}

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
