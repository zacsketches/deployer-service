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
var debugMode bool

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
	debugModeEnv := os.Getenv("DEBUG_MODE")
	if debugModeEnv != "true" && debugModeEnv != "false" {
		log.Fatal("DEBUG_MODE environment variable must be set to 'true' or 'false'; aborting startup")
	}
	// Set debugMode to true if DEBUG_MODE environment variable is "true"
	debugMode = debugModeEnv == "true"
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
	http.HandleFunc("/logout", LogoutHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	addr := ":" + port
	log.WithFields(log.Fields{
		"version": version,
		"addr":    addr,
	}).Infof("Starting deployer daemon %s on port %s", version, port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.WithError(err).Fatal("deploy service failed")
	}
	if debugMode {
		log.Info("Running in DEBUG mode")
	} else {
		log.Info("Running in PRODUCTION mode")
	}
}
