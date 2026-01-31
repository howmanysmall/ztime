// github.com/howmanysmall/ztime

// Package main is the entry point.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Metrics holds the timing and resource usage data.
type Metrics struct {
	Command      string
	UserTime     time.Duration
	SystemTime   time.Duration
	ElapsedTime  time.Duration
	CPUPercent   int
	MaxRSS       int64 // in KB
	SharedRSS    int64 // in KB
	UnsharedRSS  int64 // in KB
	UnsharedData int64 // in KB
	UnsharedStk  int64 // in KB
	PageFaults   int64 // Major
	PageReclaims int64 // Minor
	Swaps        int64
	BlockInput   int64
	BlockOutput  int64
	MsgsSent     int64
	MsgsRecv     int64
	Signals      int64
	VCtxSwitches int64
	ICtxSwitches int64
}

func main() {
	if len(os.Args) < 2 {
		// Mimic zsh time with no args (which usually prints shell stats, but here just usage/zero)
		// But strictly, we should probably just print usage.
		fmt.Fprintln(os.Stderr, "usage: ztime <command> [arguments...]")
		os.Exit(0)
	}

	// 1. Setup Command
	cmdArgs := os.Args[1:]
	//nolint:gosec // Intended behavior: ztime runs arbitrary commands.
	cmd := exec.CommandContext(context.Background(), cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 2. Signal Handling
	// Forward signals to the child process.
	sigChan := make(chan os.Signal, 1)
	// Notify for common signals we want to forward.
	// We avoid catching everything (like SIGCHLD, SIGURG, etc.) to avoid noise.
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	// 3. Execution & Measurement
	start := time.Now()
	err := cmd.Run()
	end := time.Now()

	signal.Stop(sigChan)
	close(sigChan)

	// 4. Metrics Extraction
	metrics := extractMetrics(cmd, start, end, cmdArgs)

	// 5. Formatting
	timeFmt := os.Getenv("TIMEFMT")
	if timeFmt == "" {
		// Default zsh format
		timeFmt = "%J  %U user %S system %P cpu %*E total"
	}

	fmt.Fprintln(os.Stderr, format(timeFmt, metrics))

	// 6. Exit Code
	if err == nil {
		os.Exit(0)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			os.Exit(128 + int(status.Signal()))
		}

		os.Exit(exitErr.ExitCode())
	}

	// Some other error starting the command
	fmt.Fprintf(os.Stderr, "ztime: %v\n", err)
	os.Exit(127) // Command not found or similar
}

func extractMetrics(cmd *exec.Cmd, start, end time.Time, args []string) Metrics {
	elapsed := end.Sub(start)

	m := Metrics{
		Command:     strings.Join(args, " "),
		ElapsedTime: elapsed,
	}

	if cmd.ProcessState != nil {
		m.UserTime = cmd.ProcessState.UserTime()
		m.SystemTime = cmd.ProcessState.SystemTime()

		// CPU Percentage
		m.CPUPercent = calculateCPUPercent(m.UserTime, m.SystemTime, elapsed)

		if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			// Darwin/Linux Rusage fields often use int64 or int32 depending on arch.
			m.MaxRSS = usage.Maxrss
			m.SharedRSS = usage.Ixrss
			m.UnsharedData = usage.Idrss
			m.UnsharedStk = usage.Isrss
			m.PageFaults = usage.Majflt
			m.PageReclaims = usage.Minflt
			m.Swaps = usage.Nswap
			m.BlockInput = usage.Inblock
			m.BlockOutput = usage.Oublock
			m.MsgsSent = usage.Msgsnd
			m.MsgsRecv = usage.Msgrcv
			m.Signals = usage.Nsignals
			m.VCtxSwitches = usage.Nvcsw
			m.ICtxSwitches = usage.Nivcsw
		}
	}

	return m
}

func format(tmpl string, m Metrics) string {
	var out bytes.Buffer

	inPercent := false

	//nolint:intrange // We need C-style loop to modify i for %*E skipping.
	for i := 0; i < len(tmpl); i++ {
		char := tmpl[i]

		if inPercent {
			handled := handleSpecifier(&out, char, m, &i, tmpl)
			if !handled {
				out.WriteByte('%')
				out.WriteByte(char)
			}

			inPercent = false
		} else {
			if char == '%' {
				inPercent = true
			} else {
				out.WriteByte(char)
			}
		}
	}

	// Dangle %
	if inPercent {
		out.WriteByte('%')
	}

	return out.String()
}

func handleSpecifier(out *bytes.Buffer, char byte, m Metrics, idx *int, tmpl string) bool {
	switch char {
	case '%':
		out.WriteByte('%')
	case 'J':
		out.WriteString(m.Command)
	case 'U':
		fmt.Fprintf(out, "%.2fs", m.UserTime.Seconds())
	case 'S':
		fmt.Fprintf(out, "%.2fs", m.SystemTime.Seconds())
	case 'E':
		fmt.Fprintf(out, "%.2fs", m.ElapsedTime.Seconds())
	case '*':
		handleStar(out, m, idx, tmpl)
	case 'P':
		out.WriteString(strconv.Itoa(m.CPUPercent) + "%")
	default:
		return handleIntSpecifier(out, char, m)
	}

	return true
}

func handleIntSpecifier(out *bytes.Buffer, char byte, m Metrics) bool {
	switch char {
	case 'M':
		out.WriteString(strconv.FormatInt(m.MaxRSS, 10))
	case 'W':
		out.WriteString(strconv.FormatInt(m.Swaps, 10))
	case 'X':
		out.WriteString(strconv.FormatInt(m.SharedRSS, 10))
	case 'D':
		out.WriteString(strconv.FormatInt(m.UnsharedData+m.UnsharedStk, 10))
	case 'K':
		out.WriteString(strconv.FormatInt(m.SharedRSS+m.UnsharedData+m.UnsharedStk, 10))
	case 'F':
		out.WriteString(strconv.FormatInt(m.PageFaults, 10))
	case 'R':
		out.WriteString(strconv.FormatInt(m.PageReclaims, 10))
	case 'I':
		out.WriteString(strconv.FormatInt(m.BlockInput, 10))
	case 'O':
		out.WriteString(strconv.FormatInt(m.BlockOutput, 10))
	case 'r':
		out.WriteString(strconv.FormatInt(m.MsgsRecv, 10))
	case 's':
		out.WriteString(strconv.FormatInt(m.MsgsSent, 10))
	case 'k':
		out.WriteString(strconv.FormatInt(m.Signals, 10))
	case 'w':
		out.WriteString(strconv.FormatInt(m.VCtxSwitches, 10))
	case 'c':
		out.WriteString(strconv.FormatInt(m.ICtxSwitches, 10))
	default:
		return false
	}

	return true
}

func handleStar(out *bytes.Buffer, m Metrics, idx *int, tmpl string) {
	// Handle %*E
	if *idx+1 < len(tmpl) && tmpl[*idx+1] == 'E' {
		*idx++
		// Format as mm:ss.SS or h:mm:ss
		d := m.ElapsedTime
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		secs := d.Seconds() - float64(int(d.Minutes())*60)

		if hours > 0 {
			fmt.Fprintf(out, "%d:%02d:%05.2f", hours, mins, secs)
		} else {
			fmt.Fprintf(out, "%d:%05.2f", mins, secs)
		}
	} else {
		// Unknown, just print *
		out.WriteByte('*')
	}
}

func calculateCPUPercent(user, sys, elapsed time.Duration) int {
	totalCPU := user.Seconds() + sys.Seconds()
	realSec := elapsed.Seconds()

	if realSec > 0 {
		return int((totalCPU / realSec) * 100)
	}

	return 0
}
