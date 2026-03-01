package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/apex/log"
)

type DeployRequest struct {
	Service string `json:"service"`
}

type ConfigResponse struct {
	Version           string `json:"version"`
	Port              string `json:"port"`
	DebugMode         bool   `json:"debug_mode"`
	AWSRegion         string `json:"aws_region"`
	ECRDomain         string `json:"ecr_domain"`
	DockerComposeFile string `json:"docker_compose_file"`
	JWTPublicKeyPath  string `json:"jwt_public_key_path"`
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	issuer, err := VerifyJWTFromHeader(r.Header.Get("Authorization"), jwtKeyPath)
	if err != nil {
		log.WithError(err).WithField("ip", ip).Warn("unauthorized deploy attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Read and decode base64-encoded JSON payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Error("failed to read request body")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		log.WithError(err).Error("failed to decode base64 payload")
		http.Error(w, "invalid base64", http.StatusBadRequest)
		return
	}

	var req DeployRequest
	if err := json.Unmarshal(decoded, &req); err != nil {
		log.WithError(err).Error("invalid json in base64 string")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Service) == "" {
		http.Error(w, "service is required", http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"ip":      ip,
		"iss":     issuer,
		"service": req.Service,
	}).Info("received authenticated deploy request")

	// make an initial pull attempt, followed by a login-->try again before failing.
	if err := runComposePull(dockerComposePath, req.Service); err != nil {
		// Try to login and pull again if the first pull fails
		if loginErr := runLogin(); loginErr != nil {
			http.Error(w, "unable to update service", http.StatusInternalServerError)
			return
		}
		if err := runComposePull(dockerComposePath, req.Service); err != nil {
			http.Error(w, "unable to update service", http.StatusInternalServerError)
			return
		}
	}

	if err := runComposeUp(); err != nil {
		//Service error logging completed in the exec function
		http.Error(w, "unable to update service", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "deploy request successful for service %s\n", req.Service)
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	if r.Method != http.MethodGet {
		log.WithFields(log.Fields{
			"method": r.Method,
			"ip":     ip,
			"path":   r.URL.Path,
		}).Warn("invalid method on /version")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.WithFields(log.Fields{
		"method": r.Method,
		"ip":     ip,
		"path":   r.URL.Path,
	}).Info("version query returned ok")

	fmt.Fprintln(w, version)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	if r.Method != http.MethodGet {
		log.WithFields(log.Fields{
			"method": r.Method,
			"ip":     ip,
			"path":   r.URL.Path,
		}).Warn("invalid method on /health")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.WithFields(log.Fields{
		"method": r.Method,
		"ip":     ip,
		"path":   r.URL.Path,
	}).Info("health check query returned ok")

	fmt.Fprintln(w, "deployer healthy")
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	if r.Method != http.MethodGet {
		log.WithFields(log.Fields{
			"method": r.Method,
			"ip":     ip,
			"path":   r.URL.Path,
		}).Warn("invalid method on /config")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	cfg := ConfigResponse{
		Version:           version,
		Port:              port,
		DebugMode:         debugMode,
		AWSRegion:         awsRegion,
		ECRDomain:         ecrDomain,
		DockerComposeFile: dockerComposePath,
		JWTPublicKeyPath:  jwtKeyPath,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		log.WithError(err).WithField("ip", ip).Error("failed to write config response")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"method": r.Method,
		"ip":     ip,
		"path":   r.URL.Path,
	}).Info("config query returned ok")
}

func getRemoteIP(r *http.Request) string {
	// try to get the real IP from headers if behind proxy
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	if r.Method != http.MethodPost {
		log.WithFields(log.Fields{
			"method": r.Method,
			"ip":     ip,
			"path":   r.URL.Path,
		}).Warn("invalid method on /logout")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !debugMode {
		log.WithFields(log.Fields{
			"ip":   ip,
			"path": r.URL.Path,
		}).Warn("logout attempted in non-debug mode")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	log.WithField("ip", ip).Info("received logout request")

	if err := runLogout(); err != nil {
		log.WithError(err).WithField("ip", ip).Error("logout failed")
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "docker logout successful")
}
