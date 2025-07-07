package main

import (
	"net/http"
	"os"

	"github.com/apex/log"
)

// Constants
const DefaultPort = "8686"

// Set by environment variables and the program will fatally fail without these
var dockerComposePath string
var jwtKeyPath string
var awsRegion string
var ecrDomain string

// Injected at build time via build flag -ldflags "-X=main.version=$(git rev-parse --short HEAD)"
var version string

func init() {
	jwtKeyPath = os.Getenv("JWT_PUBLIC_KEY_PATH")
	if jwtKeyPath == "" {
		log.Fatal("JWT_PUBLIC_KEY_PATH environment variable not set; aborting startup")
	}
	dockerComposePath = os.Getenv("DOCKER_COMPOSE_FILE")
	if dockerComposePath == "" {
		log.Fatal("DOCKER_COMPOSE_FILE environment variable not set; aborting startup")
	}
	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Fatal("AWS_REGION environment variable not set; aborting startup")
	}
	ecrDomain = os.Getenv("ECR_REPOSITORY")
	if ecrDomain == "" {
		log.Fatal("ECR_REPOSITORY environment variable not set; aborting startup")
	}
}

func main() {
	loggingSetup()

	// Launch the cluster
	runLogin()
	runComposeUp()

	// Configure the daemon
	http.HandleFunc("/version", VersionHandler)
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/deploy", DeployHandler)
	var port string
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = DefaultPort
	}

	addr := ":" + port
	log.WithField("addr", addr).Info("starting deployer daemon on port" + port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.WithError(err).Fatal("deploy service failed")
	}
}
