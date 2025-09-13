package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"powermetrics-tui/internal/models"
)

func TestParsePowerMetricsOutput(t *testing.T) {
	// Read the sample file from project root
	samplePath := filepath.Join("..", "..", "sample_output.txt")
	content, err := os.ReadFile(samplePath)
	if err != nil {
		t.Skipf("Sample file not found at %s: %v", samplePath, err)
	}

	// Create a new state
	state := models.NewMetricsState()

	// Parse the output
	ParsePowerMetricsOutput(string(content), state)

	// Test that we got processes
	t.Run("Processes", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		if len(state.Processes) == 0 {
			t.Error("Expected processes to be parsed, but got 0")
		}

		// Check specific process data
		if len(state.Processes) > 0 {
			proc := state.Processes[0]
			if proc.Name == "" {
				t.Error("Process name should not be empty")
			}
			if proc.PID == 0 {
				t.Error("Process PID should not be 0")
			}
			if proc.CPUPercent < 0 {
				t.Error("Process CPU percent should not be negative")
			}
			t.Logf("First process: %s (PID: %d) CPU: %.2f%%", proc.Name, proc.PID, proc.CPUPercent)
		}

		t.Logf("Parsed %d processes", len(state.Processes))
	})

	// Test interrupts parsing
	t.Run("Interrupts", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		if state.IPICount == 0 && state.TimerCount == 0 {
			t.Error("Expected interrupts to be parsed, but both IPI and Timer are 0")
		}
		t.Logf("IPI: %d, Timer: %d", state.IPICount, state.TimerCount)
	})

	// Test CPU power parsing
	t.Run("CPU Power", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		// CPU power might be 0 if not in sample, that's ok
		t.Logf("CPU Power: %.2f mW", state.CPUPower)
	})

	// Test battery parsing
	t.Run("Battery", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		// Battery info might not be present in all samples
		if state.BatteryCharge > 100 {
			t.Errorf("Battery charge should not exceed 100: %.2f", state.BatteryCharge)
		}
		t.Logf("Battery: %.2f%% (State: %s)", state.BatteryCharge, state.BatteryState)
	})

	// Test thermal parsing
	t.Run("Thermal", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		t.Logf("Thermal Pressure: %s", state.ThermalPressure)
	})

	// Test network parsing
	t.Run("Network", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		t.Logf("Network In: %.2f, Out: %.2f", state.NetworkIn, state.NetworkOut)
	})

	// Test disk parsing
	t.Run("Disk", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		t.Logf("Disk Read: %.2f, Write: %.2f", state.DiskRead, state.DiskWrite)
	})

	// Test CPU frequency parsing
	t.Run("CPU Frequencies", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		t.Logf("E-Core frequencies: %v", state.ECoreFreq)
		t.Logf("P-Core frequencies: %v", state.PCoreFreq)
		t.Logf("All CPU frequencies: %v", state.AllCpuFreq)

		if len(state.AllCpuFreq) == 0 {
			t.Log("No CPU frequency data parsed (might be expected depending on sample)")
		}
	})
}

// Test specific regex patterns
func TestProcessRegex(t *testing.T) {
	// Test line from actual sample
	testLine := "node                               63766  142.39    95.99  243.06  28.42              337.15  9.80"

	// Recreate the regex pattern used in parser
	processTestRegex := regexp.MustCompile(`^(.+?)\s+(\d+)\s+([0-9.]+)\s+([0-9.]+)`)

	matches := processTestRegex.FindStringSubmatch(testLine)
	if matches == nil {
		t.Error("Process regex failed to match sample line")
		return
	}

	if len(matches) < 5 {
		t.Errorf("Expected at least 5 matches, got %d", len(matches))
		return
	}

	name := matches[1]
	pid := matches[2]
	cpuMs := matches[3]
	userPercent := matches[4]

	// The regex captures just "node" without the padding - that's OK
	expectedName := "node"
	if strings.TrimSpace(name) != expectedName {
		t.Errorf("Expected name %q, got %q", expectedName, name)
	}
	if pid != "63766" {
		t.Errorf("Expected PID 63766, got %s", pid)
	}
	if cpuMs != "142.39" {
		t.Errorf("Expected CPU ms/s 142.39, got %s", cpuMs)
	}
	if userPercent != "95.99" {
		t.Errorf("Expected User%% 95.99, got %s", userPercent)
	}
}

func TestInterruptRegex(t *testing.T) {
	// Recreate the regex patterns
	ipiTestRegex := regexp.MustCompile(`\|-> IPI:\s+([0-9.]+)\s+interrupts/sec`)
	timerTestRegex := regexp.MustCompile(`\|-> TIMER:\s+([0-9.]+)\s+interrupts/sec`)

	testCases := []struct {
		line     string
		expected string
		regex    *regexp.Regexp
		name     string
	}{
		{
			line:     "	|-> IPI: 1659.22 interrupts/sec",
			expected: "1659.22",
			regex:    ipiTestRegex,
			name:     "IPI rate",
		},
		{
			line:     "	|-> TIMER: 863.59 interrupts/sec",
			expected: "863.59",
			regex:    timerTestRegex,
			name:     "Timer rate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := tc.regex.FindStringSubmatch(tc.line)
			if matches == nil {
				t.Errorf("%s regex failed to match line", tc.name)
				return
			}
			if matches[1] != tc.expected {
				t.Errorf("%s: expected %s, got %s", tc.name, tc.expected, matches[1])
			}
		})
	}
}