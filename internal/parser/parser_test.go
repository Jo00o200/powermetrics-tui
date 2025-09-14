package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"powermetrics-tui/internal/models"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestParsePowerMetricsOutput(t *testing.T) {
	// Read the sample file with --show-all format
	samplePath := filepath.Join("..", "..", "sample_output_all.txt")
	content, err := os.ReadFile(samplePath)
	if err != nil {
		t.Skipf("Sample file not found at %s: %v", samplePath, err)
	}

	// Create a new state
	state := models.NewMetricsState()

	// Parse the output
	ParsePowerMetricsOutput(string(content), state)

	// Test that we got processes and coalitions
	t.Run("Processes and Coalitions", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		if len(state.Processes) == 0 {
			t.Error("Expected processes to be parsed, but got 0")
		}

		if len(state.Coalitions) == 0 {
			t.Error("Expected coalitions to be parsed, but got 0")
		}

		// With --show-all we should have many more processes (subprocesses)
		if len(state.Processes) < 50 {
			t.Errorf("Expected at least 50 subprocesses with --show-all, got %d", len(state.Processes))
		}

		// Should have fewer coalitions than processes
		if len(state.Coalitions) >= len(state.Processes) {
			t.Errorf("Expected fewer coalitions (%d) than processes (%d)", len(state.Coalitions), len(state.Processes))
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
			if proc.CoalitionName == "" {
				t.Error("Process coalition name should not be empty")
			}
			t.Logf("First subprocess: %s (PID: %d) Coalition: %s CPU: %.2f%%", proc.Name, proc.PID, proc.CoalitionName, proc.CPUPercent)
		}

		// Check specific coalition data
		if len(state.Coalitions) > 0 {
			coalition := state.Coalitions[0]
			if coalition.Name == "" {
				t.Error("Coalition name should not be empty")
			}
			if coalition.CoalitionID == 0 {
				t.Error("Coalition ID should not be 0")
			}
			if coalition.CPUPercent < 0 {
				t.Error("Coalition CPU percent should not be negative")
			}
			if len(coalition.Subprocesses) == 0 {
				t.Error("Coalition should have subprocesses")
			}
			t.Logf("First coalition: %s (ID: %d) CPU: %.2f%% Subprocesses: %d", coalition.Name, coalition.CoalitionID, coalition.CPUPercent, len(coalition.Subprocesses))
		}

		// Verify hierarchy integrity
		for _, proc := range state.Processes {
			found := false
			for _, coalition := range state.Coalitions {
				if coalition.Name == proc.CoalitionName {
					// Check if this process is in the coalition's subprocess list
					for _, subprocess := range coalition.Subprocesses {
						if subprocess.PID == proc.PID {
							found = true
							break
						}
					}
					break
				}
			}
			if !found {
				t.Errorf("Process %s (PID: %d) claims to belong to coalition %s but is not in any coalition's subprocess list", proc.Name, proc.PID, proc.CoalitionName)
			}
		}

		t.Logf("Parsed %d processes and %d coalitions", len(state.Processes), len(state.Coalitions))
	})

	// Test interrupts parsing
	t.Run("Interrupts", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		if state.IPICount == 0 && state.TimerCount == 0 {
			t.Error("Expected interrupts to be parsed, but both IPI and Timer are 0")
		}
		t.Logf("Total IPI: %d, Timer: %d", state.IPICount, state.TimerCount)

		// Test per-CPU interrupts
		if len(state.PerCPUIPIs) > 0 {
			t.Logf("Per-CPU IPI breakdown:")
			for cpu, ipi := range state.PerCPUIPIs {
				t.Logf("  %s: IPI=%.1f/s", cpu, ipi)
			}
		} else {
			t.Log("No per-CPU IPI data parsed")
		}

		if len(state.PerCPUTimers) > 0 {
			t.Logf("Per-CPU Timer breakdown:")
			for cpu, timer := range state.PerCPUTimers {
				t.Logf("  %s: Timer=%.1f/s", cpu, timer)
			}
		}

		if len(state.PerCPUInterrupts) > 0 {
			t.Logf("Per-CPU Total interrupts:")
			for cpu, total := range state.PerCPUInterrupts {
				t.Logf("  %s: Total=%.1f/s", cpu, total)
			}
		}
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

		t.Logf("E-Core frequencies (CPU 0-1): %v", state.ECoreFreq)
		t.Logf("P-Core frequencies (CPU 2-9): %v", state.PCoreFreq)
		t.Logf("All CPU frequencies: %v", state.AllCpuFreq)

		// Check that we have reasonable number of cores (dynamic based on actual hardware)
		if len(state.ECoreFreq) == 0 && len(state.PCoreFreq) == 0 {
			t.Error("Expected to find some CPU cores, but got none")
		}

		// Apple Silicon typically has E-cores, Intel Macs typically don't
		// Note: CPU classification needs cluster headers in the sample to work correctly
		if len(state.ECoreFreq) > 0 {
			t.Logf("Found %d E-cores (Apple Silicon detected)", len(state.ECoreFreq))
			// This test is more lenient since cluster detection depends on sample data
			if len(state.ECoreFreq) > 16 {
				t.Errorf("Unexpected number of E-cores: %d (max expected: 16)", len(state.ECoreFreq))
			}
		}

		if len(state.PCoreFreq) > 0 {
			t.Logf("Found %d P-cores", len(state.PCoreFreq))
			if len(state.PCoreFreq) > 16 {
				t.Errorf("Unexpected number of P-cores: %d (max expected: 16)", len(state.PCoreFreq))
			}
		}

		// Check that we have individual CPU frequencies
		for i := 0; i < 10; i++ {
			if freq, exists := state.AllCpuFreq[i]; exists && freq > 0 {
				t.Logf("CPU %d: %d MHz", i, freq)
			}
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

func TestCPUFrequencyRegex(t *testing.T) {
	// Test CPU frequency regex with sample data
	testCases := []struct {
		line     string
		expected map[string]string
	}{
		{
			line:     "CPU 0 frequency: 1058 MHz",
			expected: map[string]string{"cpu": "0", "freq": "1058"},
		},
		{
			line:     "CPU 1 frequency: 1051 MHz",
			expected: map[string]string{"cpu": "1", "freq": "1051"},
		},
		{
			line:     "CPU 2 frequency: 960 MHz",
			expected: map[string]string{"cpu": "2", "freq": "960"},
		},
		{
			line:     "CPU 9 frequency: 1050 MHz",
			expected: map[string]string{"cpu": "9", "freq": "1050"},
		},
	}

	cpuFreqRegex := regexp.MustCompile(`CPU (\d+) frequency:\s+([0-9]+)\s*MHz`)

	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			matches := cpuFreqRegex.FindStringSubmatch(tc.line)
			if matches == nil {
				t.Errorf("CPU frequency regex failed to match line: %s", tc.line)
				return
			}
			if matches[1] != tc.expected["cpu"] {
				t.Errorf("Expected CPU %s, got %s", tc.expected["cpu"], matches[1])
			}
			if matches[2] != tc.expected["freq"] {
				t.Errorf("Expected frequency %s, got %s", tc.expected["freq"], matches[2])
			}
		})
	}
}

func TestParsePowerMetricsOutputAll(t *testing.T) {
	// Test with --show-all format
	sample, err := os.ReadFile("../../sample_output_all.txt")
	if err != nil {
		t.Skipf("sample_output_all.txt not found, skipping --show-all format test")
		return
	}

	state := models.NewMetricsState()
	ParsePowerMetricsOutput(string(sample), state)

	// Test that we parsed many more processes with --show-all
	if len(state.Processes) < 50 {
		t.Errorf("Expected at least 50 processes with --show-all, got %d", len(state.Processes))
	}

	// Test that we have coalitions
	if len(state.Coalitions) == 0 {
		t.Error("Expected to find coalitions, but got 0")
	}

	// Check some known processes from the sample
	foundGhostty := false
	foundNode := false
	foundChrome := false
	foundGhosttyCoalition := false
	foundChromeCoalition := false

	// Check subprocesses
	for _, proc := range state.Processes {
		if strings.Contains(proc.Name, "ghostty") {
			foundGhostty = true
		}
		if strings.Contains(proc.Name, "node") {
			foundNode = true
		}
		if strings.Contains(proc.Name, "Chrome") {
			foundChrome = true
		}
	}

	// Check coalitions
	for _, coalition := range state.Coalitions {
		if strings.Contains(coalition.Name, "ghostty") {
			foundGhosttyCoalition = true
		}
		if strings.Contains(coalition.Name, "Chrome") {
			foundChromeCoalition = true
		}
	}

	if !foundGhostty {
		t.Error("Expected to find ghostty subprocess")
	}
	if !foundNode {
		t.Error("Expected to find node subprocess")
	}
	if !foundChrome {
		t.Error("Expected to find Chrome subprocess")
	}
	if !foundGhosttyCoalition {
		t.Error("Expected to find ghostty coalition")
	}
	if !foundChromeCoalition {
		t.Error("Expected to find Chrome coalition")
	}

	// Test CPU frequency parsing (dynamic based on sample data)
	totalCores := len(state.ECoreFreq) + len(state.PCoreFreq)
	if totalCores == 0 {
		t.Error("Expected to find CPU cores, but got none")
	} else {
		t.Logf("Found %d E-cores and %d P-cores (total: %d)",
			len(state.ECoreFreq), len(state.PCoreFreq), totalCores)
	}

	t.Logf("Parsed %d processes and %d coalitions from --show-all format", len(state.Processes), len(state.Coalitions))
	t.Logf("E-Core frequencies: %v", state.ECoreFreq)
	t.Logf("P-Core frequencies: %v", state.PCoreFreq)
}

func TestInterruptRegex(t *testing.T) {
	// Recreate the regex patterns
	ipiTestRegex := regexp.MustCompile(`\|-> IPI:\s+([0-9.]+)\s+interrupts/sec`)
	timerTestRegex := regexp.MustCompile(`\|-> TIMER:\s+([0-9.]+)\s+interrupts/sec`)
	cpuTestRegex := regexp.MustCompile(`^CPU (\d+):$`)
	totalIRQTestRegex := regexp.MustCompile(`Total IRQ:\s+([0-9.]+)\s+interrupts/sec`)

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
		{
			line:     "CPU 0:",
			expected: "0",
			regex:    cpuTestRegex,
			name:     "CPU identifier",
		},
		{
			line:     "	Total IRQ: 2057.23 interrupts/sec",
			expected: "2057.23",
			regex:    totalIRQTestRegex,
			name:     "Total IRQ rate",
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

func TestDeadProcessesParsing(t *testing.T) {
	// Test parsing sample with dead processes (empty-name PIDs)
	samplePath := filepath.Join("..", "..", "sample_output_dead_processes.txt")
	content, err := os.ReadFile(samplePath)
	if err != nil {
		t.Skipf("Sample file not found at %s: %v", samplePath, err)
	}

	// Create a new state
	state := models.NewMetricsState()

	// Create persistent parser
	parser := NewParser(state)

	// Parse the output
	parser.ParseOutput(string(content))

	t.Run("Dead process detection", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		// The sample shows PID 53571 with empty name
		// It should NOT appear in active processes
		for _, proc := range state.Processes {
			if proc.PID == 53571 {
				t.Errorf("Dead process PID 53571 should not be in active processes list, but found: %s", proc.Name)
			}
		}

		// Check if it was tracked as an exited process
		foundInExited := false
		for _, exitInfo := range state.RecentlyExited {
			for _, pid := range exitInfo.PIDs {
				if pid == 53571 {
					foundInExited = true
					t.Logf("Dead process PID 53571 correctly tracked in RecentlyExited as: %s", exitInfo.Name)
					break
				}
			}
			if foundInExited {
				break
			}
		}

		if !foundInExited {
			t.Error("Dead process PID 53571 should be tracked in RecentlyExited")
		}

		t.Logf("RecentlyExited processes: %d", len(state.RecentlyExited))
		for _, exitInfo := range state.RecentlyExited {
			t.Logf("  - %s (PIDs: %v, occurrences: %d)", exitInfo.Name, exitInfo.PIDs, exitInfo.Occurrences)
		}
	})

	t.Run("DEAD_TASKS coalition handling", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		// DEAD_TASKS_COALITION should not be in active coalitions
		for _, coalition := range state.Coalitions {
			if coalition.Name == "DEAD_TASKS_COALITION" {
				t.Error("DEAD_TASKS_COALITION should not be in active coalitions list")
			}
		}

		// DEAD_TASKS subprocess should not be in active processes
		for _, proc := range state.Processes {
			if proc.Name == "DEAD_TASKS" {
				t.Error("DEAD_TASKS should not be in active processes list")
			}
		}
	})

	t.Run("Valid processes still parsed correctly", func(t *testing.T) {
		state.Mu.RLock()
		defer state.Mu.RUnlock()

		// Ensure we still parsed valid processes
		if len(state.Processes) == 0 {
			t.Error("Expected valid processes to be parsed, but got 0")
		}

		if len(state.Coalitions) == 0 {
			t.Error("Expected valid coalitions to be parsed, but got 0")
		}

		// Check for known valid processes from the sample
		foundGhostty := false
		foundPowermetrics := false
		foundNode := false

		for _, proc := range state.Processes {
			if strings.Contains(proc.Name, "ghostty") {
				foundGhostty = true
			}
			if strings.Contains(proc.Name, "powermetrics") {
				foundPowermetrics = true
			}
			if strings.Contains(proc.Name, "node") {
				foundNode = true
			}
		}

		if !foundGhostty {
			t.Error("Expected to find ghostty process")
		}
		if !foundPowermetrics {
			t.Error("Expected to find powermetrics process")
		}
		if !foundNode {
			t.Error("Expected to find node process")
		}

		t.Logf("Parsed %d valid processes and %d valid coalitions (excluding dead tasks)",
			len(state.Processes), len(state.Coalitions))
	})

	t.Run("Sample file metadata", func(t *testing.T) {
		// Verify the sample file contains expected dead process indicators
		lines := strings.Split(string(content), "\n")
		foundEmptyPID := false

		for _, line := range lines {
			// Look for the line with PID 53571 which should have empty name
			if strings.Contains(line, "53571") && strings.Contains(line, "0.00      111.57") {
				foundEmptyPID = true
				// The line starts with spaces (empty name) then PID
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, "53571") {
					t.Errorf("Expected PID 53571 line to start with PID (empty name), but got: %s", trimmed[:min(20, len(trimmed))])
				}
				break
			}
		}

		if !foundEmptyPID {
			t.Error("Sample file should contain PID 53571 with empty name")
		}
	})

	// Also test parsing with the original sample file that has comments
	t.Run("Original sample with comments", func(t *testing.T) {
		originalPath := filepath.Join("..", "..", "sample_output_dead_processes.txt")
		originalContent, err := os.ReadFile(originalPath)
		if err != nil {
			t.Skip("Original sample file not found")
		}

		// Remove comment lines (lines starting with #)
		lines := strings.Split(string(originalContent), "\n")
		var cleanLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, "#") {
				cleanLines = append(cleanLines, line)
			}
		}
		cleanContent := strings.Join(cleanLines, "\n")

		// Create new state and parser
		state2 := models.NewMetricsState()
		parser2 := NewParser(state2)
		parser2.ParseOutput(cleanContent)

		state2.Mu.RLock()
		foundInExited := false
		for _, exitInfo := range state2.RecentlyExited {
			for _, pid := range exitInfo.PIDs {
				if pid == 53571 {
					foundInExited = true
					break
				}
			}
		}
		state2.Mu.RUnlock()

		if !foundInExited {
			t.Error("Dead process PID 53571 should be tracked in RecentlyExited (original sample)")
		}
	})
}

func TestCompleteProcessClassification(t *testing.T) {
	// Test that EVERY process in the sample is correctly classified as either coalition or subprocess
	samplePath := filepath.Join("..", "..", "sample_output_all.txt")
	content, err := os.ReadFile(samplePath)
	if err != nil {
		t.Skipf("Sample file not found at %s: %v", samplePath, err)
	}

	state := models.NewMetricsState()
	ParsePowerMetricsOutput(string(content), state)

	// Expected processes from the sample file (based on actual data)
	expectedCoalitions := map[string]int{
		"com.mitchellh.ghostty": 653,
		"com.google.Chrome":     661, // Updated to actual ID from sample
		"kernel_coalition":      1,
	}

	expectedSubprocesses := map[int]string{
		24620: "powermetrics", // under ghostty
		13198: "ghostty",      // under ghostty
		5165:  "esbuild",      // under ghostty
	}

	t.Run("All expected coalitions found", func(t *testing.T) {
		foundCoalitions := make(map[string]int)
		for _, coalition := range state.Coalitions {
			foundCoalitions[coalition.Name] = coalition.CoalitionID
		}

		for expectedName, expectedID := range expectedCoalitions {
			if foundID, exists := foundCoalitions[expectedName]; !exists {
				t.Errorf("Expected coalition %s (ID: %d) not found", expectedName, expectedID)
			} else if foundID != expectedID {
				t.Errorf("Coalition %s has wrong ID: expected %d, got %d", expectedName, expectedID, foundID)
			}
		}

		t.Logf("Found %d coalitions total", len(state.Coalitions))
		for name, id := range foundCoalitions {
			t.Logf("Coalition: %s (ID: %d)", name, id)
		}
	})

	t.Run("All expected subprocesses found", func(t *testing.T) {
		foundSubprocesses := make(map[int]string)
		for _, proc := range state.Processes {
			foundSubprocesses[proc.PID] = proc.Name
		}

		for expectedPID, expectedName := range expectedSubprocesses {
			if foundName, exists := foundSubprocesses[expectedPID]; !exists {
				t.Errorf("Expected subprocess PID %d (%s) not found", expectedPID, expectedName)
			} else {
				t.Logf("Found subprocess PID %d: %s (Coalition: %s)", expectedPID, foundName, func() string {
					for _, proc := range state.Processes {
						if proc.PID == expectedPID {
							return proc.CoalitionName
						}
					}
					return "unknown"
				}())
			}
		}

		t.Logf("Found %d subprocesses total", len(state.Processes))
	})

	t.Run("Verify key PIDs are correctly classified", func(t *testing.T) {
		// Test specific PIDs that are present in the sample file
		testPIDs := []int{24620, 13198, 5165, 653}

		for _, pid := range testPIDs {
			// Check if it's a subprocess
			foundAsSubprocess := false
			for _, proc := range state.Processes {
				if proc.PID == pid {
					foundAsSubprocess = true
					t.Logf("PID %d found as subprocess: %s (Coalition: %s)", pid, proc.Name, proc.CoalitionName)
					break
				}
			}

			// Check if it's a coalition
			foundAsCoalition := false
			for _, coalition := range state.Coalitions {
				if coalition.CoalitionID == pid {
					foundAsCoalition = true
					t.Logf("PID %d found as coalition: %s", pid, coalition.Name)
					break
				}
			}

			if !foundAsSubprocess && !foundAsCoalition {
				t.Errorf("PID %d not found as either subprocess or coalition!", pid)
			} else if foundAsSubprocess && foundAsCoalition {
				t.Errorf("PID %d found as BOTH subprocess and coalition - this is wrong!", pid)
			}
		}
	})

	t.Run("Verify hierarchy integrity", func(t *testing.T) {
		// Every subprocess should belong to a coalition
		for _, proc := range state.Processes {
			found := false
			for _, coalition := range state.Coalitions {
				if coalition.Name == proc.CoalitionName {
					// Check if this process is in the coalition's subprocess list
					for _, subprocess := range coalition.Subprocesses {
						if subprocess.PID == proc.PID {
							found = true
							break
						}
					}
					break
				}
			}
			if !found {
				t.Errorf("Subprocess PID %d (%s) claims coalition %s but not found in any coalition's subprocess list",
					proc.PID, proc.Name, proc.CoalitionName)
			}
		}
	})
}
