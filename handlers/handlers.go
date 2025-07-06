package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/apex/log"
)

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
