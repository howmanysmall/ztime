package main

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	t.Parallel()

	metrics := Metrics{
		Command:      "sleep 1",
		UserTime:     500 * time.Millisecond,
		SystemTime:   250 * time.Millisecond,
		ElapsedTime:  2500 * time.Millisecond,
		CPUPercent:   30,
		MaxRSS:       1024,
		SharedRSS:    512,
		UnsharedData: 256,
		UnsharedStk:  128,
		PageFaults:   10,
		PageReclaims: 20,
		Swaps:        5,
		BlockInput:   100,
		BlockOutput:  200,
		MsgsRecv:     50,
		MsgsSent:     60,
		Signals:      2,
		VCtxSwitches: 15,
		ICtxSwitches: 25,
	}

	tests := []struct {
		name     string
		fmt      string
		expected string
	}{
		{
			name:     "Default",
			fmt:      "%J  %U user %S system %P cpu %*E total",
			expected: "sleep 1  0.50s user 0.25s system 30% cpu 0:02.50 total",
		},
		{
			name:     "Elapsed Simple",
			fmt:      "%E",
			expected: "2.50s",
		},
		{
			name:     "Elapsed Star",
			fmt:      "%*E",
			expected: "0:02.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := format(tt.fmt, metrics)
			if got != tt.expected {
				t.Errorf("format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatMetrics(t *testing.T) {
	t.Parallel()

	metrics := Metrics{
		MaxRSS:       1024,
		SharedRSS:    512,
		UnsharedData: 256,
		UnsharedStk:  128,
		PageFaults:   10,
		PageReclaims: 20,
		Swaps:        5,
		BlockInput:   100,
		BlockOutput:  200,
		MsgsRecv:     50,
		MsgsSent:     60,
		Signals:      2,
		VCtxSwitches: 15,
		ICtxSwitches: 25,
	}

	tests := []struct {
		name     string
		fmt      string
		expected string
	}{
		{
			name:     "RSS",
			fmt:      "%M",
			expected: "1024",
		},
		{
			name:     "Literal Percent",
			fmt:      "100%%",
			expected: "100%",
		},
		{
			name:     "All Int Metrics",
			fmt:      "%W %X %D %K %F %R %I %O %r %s %k %w %c",
			expected: "5 512 384 896 10 20 100 200 50 60 2 15 25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := format(tt.fmt, metrics)
			if got != tt.expected {
				t.Errorf("format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatElapsedHours(t *testing.T) {
	t.Parallel()

	m := Metrics{
		ElapsedTime: 3661 * time.Second, // 1h 1m 1s
	}
	got := format("%*E", m)
	// 1h = 3600, 61s left -> 1m 1s.
	// Expected: 1:01:01.00
	expected := "1:01:01.00"
	if got != expected {
		t.Errorf("format(%%*E) for >1h = %q, want %q", got, expected)
	}
}

func TestCalculateCPUPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     time.Duration
		sys      time.Duration
		elapsed  time.Duration
		expected int
	}{
		{
			name:     "Normal",
			user:     500 * time.Millisecond,
			sys:      500 * time.Millisecond,
			elapsed:  1000 * time.Millisecond,
			expected: 100,
		},
		{
			name:     "Zero Elapsed",
			user:     500 * time.Millisecond,
			sys:      500 * time.Millisecond,
			elapsed:  0,
			expected: 0,
		},
		{
			name:     "Low CPU",
			user:     10 * time.Millisecond,
			sys:      10 * time.Millisecond,
			elapsed:  1000 * time.Millisecond,
			expected: 2,
		},
		{
			name:     "Multi Core",
			user:     2000 * time.Millisecond,
			sys:      500 * time.Millisecond,
			elapsed:  1000 * time.Millisecond,
			expected: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := calculateCPUPercent(tt.user, tt.sys, tt.elapsed)
			if got != tt.expected {
				t.Errorf("calculateCPUPercent() = %d, want %d", got, tt.expected)
			}
		})
	}
}
