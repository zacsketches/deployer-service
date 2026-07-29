package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestDeploymentCoordinatorPreventsOverlappingCommands(t *testing.T) {
	pullStarted := make(chan struct{})
	releasePull := make(chan struct{})

	var commandMutex sync.Mutex
	activeCommands := 0
	maxActiveCommands := 0
	pullCalls := 0
	loginCalls := 0
	composeUpCalls := 0

	runCommand := func(block bool) {
		commandMutex.Lock()
		activeCommands++
		if activeCommands > maxActiveCommands {
			maxActiveCommands = activeCommands
		}
		commandMutex.Unlock()

		if block {
			close(pullStarted)
			<-releasePull
		}

		commandMutex.Lock()
		activeCommands--
		commandMutex.Unlock()
	}

	coordinator := newDeploymentCoordinator(deploymentCommands{
		pull: func(context.Context, string, string) error {
			commandMutex.Lock()
			pullCalls++
			commandMutex.Unlock()
			runCommand(true)
			return nil
		},
		login: func(context.Context) error {
			commandMutex.Lock()
			loginCalls++
			commandMutex.Unlock()
			return nil
		},
		composeUp: func(context.Context, string, string) error {
			commandMutex.Lock()
			composeUpCalls++
			commandMutex.Unlock()
			runCommand(false)
			return nil
		},
	})

	firstResult := make(chan error, 1)
	go func() {
		firstResult <- coordinator.deploy(context.Background(), "/test/docker-compose.yml", "frontend")
	}()
	<-pullStarted

	const concurrentRequests = 5
	results := make(chan error, concurrentRequests)
	var waitGroup sync.WaitGroup
	for range concurrentRequests {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- coordinator.deploy(context.Background(), "/test/docker-compose.yml", "frontend")
		}()
	}
	waitGroup.Wait()
	close(results)

	for err := range results {
		if !errors.Is(err, errDeploymentInProgress) {
			t.Errorf("concurrent deploy error = %v, want %v", err, errDeploymentInProgress)
		}
	}

	close(releasePull)
	if err := <-firstResult; err != nil {
		t.Fatalf("first deploy error = %v, want nil", err)
	}

	commandMutex.Lock()
	defer commandMutex.Unlock()
	if maxActiveCommands != 1 {
		t.Errorf("maximum active Docker commands = %d, want 1", maxActiveCommands)
	}
	if pullCalls != 1 {
		t.Errorf("pull calls = %d, want 1", pullCalls)
	}
	if loginCalls != 0 {
		t.Errorf("login calls = %d, want 0", loginCalls)
	}
	if composeUpCalls != 1 {
		t.Errorf("compose up calls = %d, want 1", composeUpCalls)
	}
}

func TestWriteDeployErrorWhenDeploymentInProgress(t *testing.T) {
	response := httptest.NewRecorder()

	writeDeployError(response, "frontend", errDeploymentInProgress)

	if response.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if retryAfter := response.Header().Get("Retry-After"); retryAfter != deployRetryAfterSeconds {
		t.Errorf("Retry-After = %q, want %q", retryAfter, deployRetryAfterSeconds)
	}
	if body := strings.TrimSpace(response.Body.String()); body != "deployment already in progress; retry later" {
		t.Errorf("response body = %q, want deployment retry message", body)
	}
}

func TestWriteDeployErrorWhenCommandTimesOut(t *testing.T) {
	response := httptest.NewRecorder()

	writeDeployError(response, "frontend", errCommandTimedOut)

	if response.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", response.Code, http.StatusGatewayTimeout)
	}
	if body := strings.TrimSpace(response.Body.String()); body != "deployment command timed out; retry later" {
		t.Errorf("response body = %q, want timeout retry message", body)
	}
}
