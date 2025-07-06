package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/zacsketches/deployer-service/auth.go"
)

type DeployRequest struct {
	Service string `json:"service"`
	Image   string `json:"image"`
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	ip := getRemoteIP(r)

	// Verify JWT using Authorization header
	pubKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if pubKeyPath == "" {
		log.Error("JWT_PUBLIC_KEY_PATH environment variable not set; shutting down")
		os.Exit(1)
	}

	if err := auth.VerifyJWTFromHeader(r.Header.Get("Authorization"), pubKeyPath); err != nil {
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

	log.WithFields(log.Fields{
		"ip":      ip,
		"service": req.Service,
		"image":   req.Image,
	}).Info("received deploy request")

	fmt.Fprintf(w, "deploy request accepted for service %s using image %s\n", req.Service, req.Image)
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

func getRemoteIP(r *http.Request) string {
	// try to get the real IP from headers if behind proxy
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
