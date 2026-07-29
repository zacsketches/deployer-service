package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitializeService(t *testing.T) {
	loginErr := errors.New("login failed")
	composeErr := errors.New("compose failed")

	tests := []struct {
		name             string
		loginErr         error
		composeErr       error
		wantError        string
		wantComposeCalls int
		wantReady        bool
	}{
		{
			name:      "login failure",
			loginErr:  loginErr,
			wantError: "ECR login during startup",
		},
		{
			name:             "compose failure",
			composeErr:       composeErr,
			wantError:        "initial Docker Compose up during startup",
			wantComposeCalls: 1,
		},
		{
			name:             "success",
			wantComposeCalls: 1,
			wantReady:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			composeCalls := 0
			composeFile := ""

			err := initializeService(context.Background(),
				func(context.Context) error {
					return test.loginErr
				},
				func(_ context.Context, gotComposeFile string) error {
					composeCalls++
					composeFile = gotComposeFile
					return test.composeErr
				},
			)

			if test.wantError == "" {
				if err != nil {
					t.Fatalf("initializeService() error = %v, want nil", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("initializeService() error = %v, want error containing %q", err, test.wantError)
			}

			if composeCalls != test.wantComposeCalls {
				t.Errorf("compose calls = %d, want %d", composeCalls, test.wantComposeCalls)
			}
			if composeCalls > 0 && composeFile != dockerComposePath {
				t.Errorf("compose file = %q, want %q", composeFile, dockerComposePath)
			}
			if gotReady := initializationComplete.Load(); gotReady != test.wantReady {
				t.Errorf("initialization readiness = %t, want %t", gotReady, test.wantReady)
			}
		})
	}
}

func TestHealthHandlerInitializationReadiness(t *testing.T) {
	originalReadiness := initializationComplete.Load()
	t.Cleanup(func() {
		initializationComplete.Store(originalReadiness)
	})

	tests := []struct {
		name       string
		ready      bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "initialization incomplete",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "initialization incomplete",
		},
		{
			name:       "initialization complete",
			ready:      true,
			wantStatus: http.StatusOK,
			wantBody:   "deployer healthy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initializationComplete.Store(test.ready)

			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			response := httptest.NewRecorder()
			HealthHandler(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("HealthHandler() status = %d, want %d", response.Code, test.wantStatus)
			}
			if body := strings.TrimSpace(response.Body.String()); body != test.wantBody {
				t.Errorf("HealthHandler() body = %q, want %q", body, test.wantBody)
			}
		})
	}
}
