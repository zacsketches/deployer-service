package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/apex/log"
)

var errCommandTimedOut = errors.New("command timed out")

func commandWithTimeout(parent context.Context, name string, args ...string) (*exec.Cmd, context.Context, context.CancelFunc) {
	commandCtx, cancel := context.WithTimeout(parent, commandTimeout)
	return exec.CommandContext(commandCtx, name, args...), commandCtx, cancel
}

func commandError(commandCtx context.Context, err error) error {
	if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w after %s", errCommandTimedOut, commandTimeout)
	}
	return err
}

func logCommandFailure(commandCtx context.Context, fields log.Fields, message string, err error) error {
	commandErr := commandError(commandCtx, err)
	if errors.Is(commandErr, errCommandTimedOut) {
		fields["error"] = commandErr
		fields["timeout"] = commandTimeout.String()
		fields["timeout_seconds"] = commandTimeout.Seconds()
		log.WithFields(fields).Error(message + " timed out")
		return commandErr
	}

	fields["error"] = commandErr
	log.WithFields(fields).Error(message)
	return commandErr
}

func runLogin(parent context.Context) error {
	log.Info("login handler triggered")

	var stdoutBuf, stderrBuf bytes.Buffer

	loginCmd, commandCtx, cancel := commandWithTimeout(parent, "sh", "-c", fmt.Sprintf(`aws ecr get-login-password --region %s | docker login --username AWS --password-stdin %s`, awsRegion, ecrDomain))
	defer cancel()
	loginCmd.Stdout = &stdoutBuf
	loginCmd.Stderr = &stderrBuf

	err := loginCmd.Run()
	if err != nil {
		return logCommandFailure(commandCtx, log.Fields{
			"action": "login",
			"stdout": stdoutBuf.String(),
			"stderr": stderrBuf.String(),
		}, "ecr login failed", err)
	}

	log.WithFields(log.Fields{
		"action": "login",
		"stdout": stdoutBuf.String(),
	}).Info("ecr login was successful")
	return nil
}

func runLogout(parent context.Context) error {
	log.WithFields(log.Fields{
		"action": "logout",
		"target": ecrDomain,
	}).Info("starting docker logout")

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd, commandCtx, cancel := commandWithTimeout(parent, "docker", "logout", ecrDomain)
	defer cancel()
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return logCommandFailure(commandCtx, log.Fields{
			"action": "logout",
			"target": ecrDomain,
			"stdout": stdoutBuf.String(),
			"stderr": stderrBuf.String(),
		}, "docker logout failed", err)
	}

	log.WithFields(log.Fields{
		"action": "logout",
		"target": ecrDomain,
		"stdout": stdoutBuf.String(),
	}).Info("docker logout successful")

	return nil
}

func composeUpArgs(composeFilePath, service string) []string {
	args := []string{"compose", "-f", composeFilePath, "up", "-d"}
	if service != "" {
		args = append(args, service)
	}
	return args
}

func runComposeUpAll(parent context.Context, composeFilePath string) error {
	return runComposeUp(parent, composeFilePath, "")
}

func runComposeUpService(parent context.Context, composeFilePath, service string) error {
	return runComposeUp(parent, composeFilePath, service)
}

func runComposeUp(parent context.Context, composeFilePath, service string) error {
	logFields := log.Fields{
		"action":       "compose up",
		"compose_file": composeFilePath,
	}
	if service == "" {
		logFields["scope"] = "all-services"
	} else {
		logFields["scope"] = "service"
		logFields["service"] = service
	}

	log.WithFields(logFields).Info("starting docker compose up -d")

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd, commandCtx, cancel := commandWithTimeout(parent, "docker", composeUpArgs(composeFilePath, service)...)
	defer cancel()
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		logFields["stdout"] = stdoutBuf.String()
		logFields["stderr"] = stderrBuf.String()
		return logCommandFailure(commandCtx, logFields, "docker compose up failed", err)
	}

	logFields["stdout"] = stdoutBuf.String()
	log.WithFields(logFields).Info("docker compose up successful")

	return nil
}

func runComposePull(parent context.Context, composeFilePath, service string) error {
	log.WithFields(log.Fields{
		"action":       "pull",
		"service":      service,
		"compose_file": composeFilePath,
	}).Info("starting docker compose pull")

	var stdoutBuf, stderrBuf bytes.Buffer

	cmd, commandCtx, cancel := commandWithTimeout(parent, "docker", "compose", "-f", composeFilePath, "pull", service)
	defer cancel()
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return logCommandFailure(commandCtx, log.Fields{
			"action":       "pull",
			"service":      service,
			"compose_file": composeFilePath,
			"stdout":       stdoutBuf.String(),
			"stderr":       stderrBuf.String(),
		}, "docker compose pull failed", err)
	}

	log.WithFields(log.Fields{
		"action":       "pull",
		"service":      service,
		"compose_file": composeFilePath,
		"stdout":       stdoutBuf.String(),
	}).Info("docker compose pull completed")

	return nil
}
