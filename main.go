package main

import (
	"net/http"
	"os"

	"github.com/apex/log"
)

// Set by environment variables and the program will fatally fail without these
var DockerComposePath string
var JWTKeyPath string

// Injected at build time via build flag -ldflags "-X=main.version=$(git rev-parse --short HEAD)"
var version string

func init() {
	JWTKeyPath = os.Getenv("JWT_PUBLIC_KEY_PATH")
	if JWTKeyPath == "" {
		log.Fatal("JWT_PUBLIC_KEY_PATH environment variable not set; aborting startup")
	}
	DockerComposePath = os.Getenv("DOCKER_COMPOSE_FILE")
	if DockerComposePath == "" {
		log.Fatal("DOCKER_COMPOSE_FILE environment variable not set; aborting startup")
	}
}

func main() {
	loggingSetup()

	http.HandleFunc("/version", VersionHandler)
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/deploy", DeployHandler)

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
