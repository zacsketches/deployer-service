package main

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/apex/log"
)

func runLogin() error {
	log.Info("login handler triggered")

	var stdoutBuf, stderrBuf bytes.Buffer

	loginCmd := exec.Command("sh", "-c", fmt.Sprintf(`aws ecr get-login-password --region %s | docker login --username AWS --password-stdin %s`, awsRegion, ecrDomain))
	loginCmd.Stdout = &stdoutBuf
	loginCmd.Stderr = &stderrBuf

	err := loginCmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"action": "login",
			"stdout": stdoutBuf.String(),
			"stderr": stderrBuf.String(),
			"error":  err,
		}).Error("ecr login failed")
		return err
	}

	log.WithFields(log.Fields{
		"action": "login",
		"stdout": stdoutBuf.String(),
	}).Info("ecr login was successful")
	return nil
}

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

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "pull", service)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	if err != nil {
		log.WithFields(log.Fields{
			"action":       "pull",
			"service":      service,
			"compose_file": composeFilePath,
			"stdout":       stdoutBuf.String(),
			"stderr":       stderrBuf.String(),
			"error":        err,
		}).Error("docker compose pull failed")
		return err
	}

	log.WithFields(log.Fields{
		"action":       "pull",
		"service":      service,
		"compose_file": composeFilePath,
		"stdout":       stdoutBuf.String(),
	}).Info("docker compose pull completed")

	return nil
}
