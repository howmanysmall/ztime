// github.com/howmanysmall/ztime

// Package main is the entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ztime <command> [arguments...]")
		os.Exit(2)
	}

	//nolint:gosec // Intended behavior: ztime runs arbitrary commands.
	command := exec.CommandContext(context.Background(), os.Args[1], os.Args[2:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	startTime := time.Now()
	err := command.Run()
	elapsedTime := time.Since(startTime)

	var userTime, systemTime time.Duration
	if processState := command.ProcessState; processState != nil {
		userTime = processState.UserTime()
		systemTime = processState.SystemTime()
	}

	realSeconds := elapsedTime.Seconds()
	userSeconds := userTime.Seconds()
	systemSeconds := systemTime.Seconds()

	cpuUsage := 0.0
	if realSeconds > 0 {
		cpuUsage = 100.0 * (userSeconds + systemSeconds) / realSeconds
	}

	fmt.Fprintf(os.Stderr, "%.2fs user %.2fs system %.0f%% cpu %.3fs total\n", userSeconds, systemSeconds, cpuUsage, realSeconds)

	if err == nil {
		os.Exit(0)
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		os.Exit(exitError.ExitCode())
	}

	os.Exit(1)
}
