package main

import (
	"os"
	"os/exec"

	"github.com/apex/log"
)

// func runComposeUp(composeFilePath string, service string) error {
// 	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "up", "--pull", "always", "-d", service)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	return cmd.Run()
// }

func runComposePull(composeFilePath, service string) error {
	log.WithFields(log.Fields{
		"action":       "pull",
		"service":      service,
		"compose_file": composeFilePath,
	}).Info("Starting docker compose pull")

	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "pull", service)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
