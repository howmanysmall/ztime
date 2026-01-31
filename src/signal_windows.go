//go:build windows

package main

import "os"

func signalList() []os.Signal {
	return []os.Signal{os.Interrupt}
}
