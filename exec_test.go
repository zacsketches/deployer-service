package main

import (
	"reflect"
	"testing"
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
