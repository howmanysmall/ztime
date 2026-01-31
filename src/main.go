// github.com/howmanysmall/ztime

// Package main is the entry point.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
)

// Metrics holds the timing and resource usage data.
type Metrics struct {
	Command      string        `json:"command"`
	UserTime     time.Duration `json:"user_time"`
	SystemTime   time.Duration `json:"system_time"`
	ElapsedTime  time.Duration `json:"elapsed_time"`
	CPUPercent   int           `json:"cpu_percent"`
	MaxRSS       int64         `json:"max_rss"`       // in KB
	SharedRSS    int64         `json:"shared_rss"`    // in KB
	UnsharedRSS  int64         `json:"unshared_rss"`  // in KB
	UnsharedData int64         `json:"unshared_data"` // in KB
	UnsharedStk  int64         `json:"unshared_stk"`  // in KB
	PageFaults   int64         `json:"page_faults"`   // Major
	PageReclaims int64         `json:"page_reclaims"` // Minor
	Swaps        int64         `json:"swaps"`
	BlockInput   int64         `json:"block_input"`
	BlockOutput  int64         `json:"block_output"`
	MsgsSent     int64         `json:"msgs_sent"`
	MsgsRecv     int64         `json:"msgs_recv"`
	Signals      int64         `json:"signals"`
	VCtxSwitches int64         `json:"v_ctx_switches"`
	ICtxSwitches int64         `json:"i_ctx_switches"`
}

func main() {
	var cli struct {
		JSON    bool     `help:"Output metrics in JSON format."`
		Quiet   bool     `short:"q" help:"Suppress the summary output."`
		Command []string `arg:"" help:"Command to execute." passthrough:""`
	}

	kctx := kong.Parse(&cli,
		kong.Name("ztime"),
		kong.Description("A shell-independent command timer replacement for 'zsh time'."),
		kong.UsageOnError(),
	)

	if len(cli.Command) == 0 {
		_ = kctx.PrintUsage(false)

		os.Exit(0)
	}

	metrics, err := runCommand(cli.Command)

	// 5. Output
	if !cli.Quiet {
		if cli.JSON {
			data, _ := json.MarshalIndent(metrics, "", "  ")

			fmt.Fprintln(os.Stderr, string(data))
		} else {
			printSummary(metrics)
		}
	}

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

	fmt.Fprintf(os.Stderr, "ztime: %v\n", err)

	os.Exit(127)
}

func runCommand(args []string) (Metrics, error) {
	// 1. Setup Command
	//nolint:gosec // Intended behavior: ztime runs arbitrary commands.
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 2. Signal Handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, signalList()...)

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
	return extractMetrics(cmd, start, end, args), err
}

func printSummary(m Metrics) {
	timeFmt := os.Getenv("TIMEFMT")
	if timeFmt != "" {
		fmt.Fprintln(os.Stderr, format(timeFmt, m))
		return
	}

	// Default styled output using lipgloss
	bold := lipgloss.NewStyle().Bold(true)
	faint := lipgloss.NewStyle().Faint(true)
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("%s  %s user %s system %s cpu %s total\n",
		faint.Render(m.Command),
		blue.Render(fmt.Sprintf("%.2fs", m.UserTime.Seconds())),
		blue.Render(fmt.Sprintf("%.2fs", m.SystemTime.Seconds())),
		green.Render(fmt.Sprintf("%d%%", m.CPUPercent)),
		bold.Render(fmt.Sprintf("%.3fs", m.ElapsedTime.Seconds())),
	))

	fmt.Fprint(os.Stderr, summary.String())
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
		m.CPUPercent = calculateCPUPercent(m.UserTime, m.SystemTime, elapsed)

		populateUsage(&m, cmd.ProcessState)
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
	if *idx+1 < len(tmpl) && tmpl[*idx+1] == 'E' {
		*idx++
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
