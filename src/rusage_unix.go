//go:build !windows

package main

import (
	"os"
	"syscall"
)

func populateUsage(m *Metrics, state *os.ProcessState) {
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok {
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
