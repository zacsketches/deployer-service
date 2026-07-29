package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestComposeUpArgs(t *testing.T) {
	const composeFile = "/opt/deployer-service/compose/docker-compose.yml"

	tests := []struct {
		name    string
		service string
		want    []string
	}{
		{
			name:    "service",
			service: "frontend",
			want:    []string{"compose", "-f", composeFile, "up", "-d", "frontend"},
		},
		{
			name: "all services",
			want: []string{"compose", "-f", composeFile, "up", "-d"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := composeUpArgs(composeFile, test.service)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("composeUpArgs() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCommandWithTimeoutTerminatesBlockedCommand(t *testing.T) {
	originalCommandTimeout := commandTimeout
	commandTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		commandTimeout = originalCommandTimeout
	})

	cmd, commandCtx, cancel := commandWithTimeout(context.Background(), "sh", "-c", "exec sleep 10")
	defer cancel()

	started := time.Now()
	err := cmd.Run()
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("blocked command error = nil, want timeout")
	}
	if !errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		t.Errorf("command context error = %v, want %v", commandCtx.Err(), context.DeadlineExceeded)
	}
	if !errors.Is(commandError(commandCtx, err), errCommandTimedOut) {
		t.Errorf("command error = %v, want timeout error", commandError(commandCtx, err))
	}
	if elapsed > time.Second {
		t.Errorf("blocked command ran for %s, want no more than %s", elapsed, time.Second)
	}
}
