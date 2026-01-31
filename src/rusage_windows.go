//go:build windows

package main

import "os"

func populateUsage(m *Metrics, state *os.ProcessState) {
	_ = m
	_ = state
}
