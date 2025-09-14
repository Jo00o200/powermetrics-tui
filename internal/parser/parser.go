package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"powermetrics-tui/internal/models"
)

var (
	// New format: interrupts/sec
	ipiRateRegex    = regexp.MustCompile(`\|-> IPI:\s+([0-9.]+)\s+interrupts/sec`)
	timerRateRegex  = regexp.MustCompile(`\|-> TIMER:\s+([0-9.]+)\s+interrupts/sec`)
	totalRateRegex  = regexp.MustCompile(`Total IRQ:\s+([0-9.]+)\s+interrupts/sec`)

	// Per-CPU interrupt patterns
	cpuInterruptRegex = regexp.MustCompile(`^CPU (\d+):$`)

	// Old format: absolute counts (keeping for compatibility)
	ipiRegex         = regexp.MustCompile(`CPU (\d+) IPI:\s+(\d+)`)
	timerRegex       = regexp.MustCompile(`CPU (\d+) Timer:\s+(\d+)`)
	totalRegex       = regexp.MustCompile(`CPU (\d+) Total:\s+(\d+)`)
	cpuPowerRegex    = regexp.MustCompile(`(?:CPU Power|CPU Energy|Combined Power \(CPU\)):\s+([0-9.]+)\s*mW`)
	gpuPowerRegex    = regexp.MustCompile(`(?:GPU Power|GPU Energy|Combined Power \(GPU\)):\s+([0-9.]+)\s*mW`)
	anePowerRegex    = regexp.MustCompile(`(?:ANE Power|ANE Energy|Combined Power \(ANE\)):\s+([0-9.]+)\s*mW`)
	dramPowerRegex   = regexp.MustCompile(`(?:DRAM Power|DRAM Energy|Combined Power \(DRAM\)):\s+([0-9.]+)\s*mW`)
	systemPowerRegex = regexp.MustCompile(`(?:Combined Power|System Power|System Average).*?:\s+([0-9.]+)\s*(?:mW|Watts)`)
	// Thermal pattern - Updated for actual format: "Current pressure level: Nominal"
	thermalRegex     = regexp.MustCompile(`Current pressure level:\s+(\w+)`)
	tempRegex        = regexp.MustCompile(`([^:]+):\s+([0-9.]+)\s*(?:C|°C)`)
	batteryRegex     = regexp.MustCompile(`(?:Battery charge|State of Charge|percent_charge):\s+([0-9.]+)(?:%)?`)
	batteryStateRegex = regexp.MustCompile(`Battery state:\s+(\w+)`)

	// CPU frequency patterns (various formats)
	ecoreFreqRegex = regexp.MustCompile(`E-Cluster HW active frequency:\s+([0-9]+)\s*MHz`)
	pcoreFreqRegex = regexp.MustCompile(`P\d*-Cluster HW active frequency:\s+([0-9]+)\s*MHz`)  // Matches P0-Cluster, P1-Cluster, P-Cluster
	gpuFreqRegex   = regexp.MustCompile(`(?:GPU active frequency|GPU frequency):\s+([0-9]+)\s*MHz`)

	// Per-CPU frequency
	cpuFreqRegex   = regexp.MustCompile(`CPU (\d+) frequency:\s+([0-9]+)\s*MHz`)

	// Network patterns
	// Network patterns - Updated for actual format: "in: 70.77 packets/s, 69338.38 bytes/s"
	networkInRegex  = regexp.MustCompile(`in:\s+[0-9.]+\s+packets/s,\s+([0-9.]+)\s+bytes/s`)
	networkOutRegex = regexp.MustCompile(`out:\s+[0-9.]+\s+packets/s,\s+([0-9.]+)\s+bytes/s`)

	// Disk I/O patterns - Updated for actual format: "read: 0.00 ops/s 0.00 KBytes/s"
	diskReadRegex  = regexp.MustCompile(`read:\s+[0-9.]+\s+ops/s\s+([0-9.]+)\s+KBytes/s`)
	diskWriteRegex = regexp.MustCompile(`write:\s+[0-9.]+\s+ops/s\s+([0-9.]+)\s+KBytes/s`)

	// Memory patterns
	memoryUsedRegex  = regexp.MustCompile(`(?:Memory Used|Physical Memory Used):\s+([0-9.]+)\s*(?:MB|GB)`)
	memoryAvailRegex = regexp.MustCompile(`(?:Memory Available|Physical Memory Available):\s+([0-9.]+)\s*(?:MB|GB)`)
	swapUsedRegex    = regexp.MustCompile(`(?:Swap Used|VM Swap Used):\s+([0-9.]+)\s*(?:MB|GB)`)

	// Process patterns
	// Updated regex for "Running tasks" format
	// Format: Name (padded to ~35 chars) ID CPU_ms/s User% ...
	// Using a simpler approach: capture everything before the first number as name
	processRegex = regexp.MustCompile(`^(.+?)\s+(\d+)\s+([0-9.]+)\s+([0-9.]+)`)
)

// ParsePowerMetricsOutput parses the output from powermetrics command using a state machine
func ParsePowerMetricsOutput(output string, state *models.MetricsState) {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	// Initialize maps if needed
	if state.AllSeenCPUs == nil {
		state.AllSeenCPUs = make(map[string]bool)
	}
	if state.PerCPUInterrupts == nil {
		state.PerCPUInterrupts = make(map[string]float64)
	}
	if state.PerCPUIPIs == nil {
		state.PerCPUIPIs = make(map[string]float64)
	}
	if state.PerCPUTimers == nil {
		state.PerCPUTimers = make(map[string]float64)
	}

	// Create state machine
	stateMachine := NewStateMachine(state)

	// Process each line through the state machine
	lines := strings.Split(output, "\n")
	for _, rawLine := range lines {
		if err := stateMachine.ProcessLine(rawLine); err != nil {
			// Log error but continue processing
			fmt.Printf("Parser error: %v\n", err)
		}
	}

	// Get the accumulated values from the state machine context
	ctx := stateMachine.GetContext()

	// Update interrupt totals
	if ctx.IPITotal > 0 {
		state.IPICount = int(ctx.IPITotal)
	}
	if ctx.TimerTotal > 0 {
		state.TimerCount = int(ctx.TimerTotal)
	}
	if ctx.InterruptsTotal > 0 {
		state.TotalInterrupts = int(ctx.InterruptsTotal)
	}

	// Note: Process and coalition tracking is now handled in the TasksHandler's Exit method

	// Organize CPU frequencies based on what we detected
	organizeCPUFrequencies(state)

	// Update per-CPU interrupt history for all known CPUs
	// Since we reset all CPUs to 0 at the start, this will include zeros for missing CPUs
	for cpu := range state.AllSeenCPUs {
		total := state.PerCPUInterrupts[cpu] // Will be 0 if CPU wasn't in this sample
		if state.PerCPUInterruptHistory[cpu] == nil {
			state.PerCPUInterruptHistory[cpu] = make([]float64, 0, 30)
		}
		state.PerCPUInterruptHistory[cpu] = models.AddToHistory(state.PerCPUInterruptHistory[cpu], total, 30)
	}

	// Update history
	state.History.IPIHistory = models.AddToIntHistory(state.History.IPIHistory, state.IPICount, state.History.MaxHistory)
	state.History.TimerHistory = models.AddToIntHistory(state.History.TimerHistory, state.TimerCount, state.History.MaxHistory)
	state.History.TotalHistory = models.AddToIntHistory(state.History.TotalHistory, state.TotalInterrupts, state.History.MaxHistory)
	state.History.CPUPowerHistory = models.AddToHistory(state.History.CPUPowerHistory, state.CPUPower, state.History.MaxHistory)
	state.History.GPUPowerHistory = models.AddToHistory(state.History.GPUPowerHistory, state.GPUPower, state.History.MaxHistory)
	state.History.SystemHistory = models.AddToHistory(state.History.SystemHistory, state.SystemPower, state.History.MaxHistory)
	state.History.NetworkInHistory = models.AddToHistory(state.History.NetworkInHistory, state.NetworkIn, state.History.MaxHistory)
	state.History.NetworkOutHistory = models.AddToHistory(state.History.NetworkOutHistory, state.NetworkOut, state.History.MaxHistory)
	state.History.DiskReadHistory = models.AddToHistory(state.History.DiskReadHistory, state.DiskRead, state.History.MaxHistory)
	state.History.DiskWriteHistory = models.AddToHistory(state.History.DiskWriteHistory, state.DiskWrite, state.History.MaxHistory)
	state.History.BatteryHistory = models.AddToHistory(state.History.BatteryHistory, state.BatteryCharge, state.History.MaxHistory)
	state.History.MemoryHistory = models.AddToHistory(state.History.MemoryHistory, state.MemoryUsed, state.History.MaxHistory)

	// Update average temperature
	if len(state.Temperature) > 0 {
		var avgTemp float64
		for _, temp := range state.Temperature {
			avgTemp += temp
		}
		avgTemp /= float64(len(state.Temperature))
		state.History.TempHistory = models.AddToHistory(state.History.TempHistory, avgTemp, state.History.MaxHistory)
	}
}

func convertToMB(value float64, line string) float64 {
	if strings.Contains(line, "KB") {
		return value / 1024
	} else if strings.Contains(line, "GB") {
		return value * 1024
	} else if strings.Contains(line, "bytes") {
		return value / (1024 * 1024)
	}
	return value
}

// organizeCPUFrequencies categorizes CPUs based on cluster information parsed from powermetrics
func organizeCPUFrequencies(state *models.MetricsState) {
	if state.AllCpuFreq == nil || len(state.AllCpuFreq) == 0 {
		return
	}

	// Separate CPUs based on cluster membership information
	ecoreCPUs := make([]int, 0)
	pcoreCPUs := make([]int, 0)

	// Check if we have cluster membership info
	hasClusterInfo := false
	if state.AllSeenCPUs != nil {
		for key := range state.AllSeenCPUs {
			if strings.HasPrefix(key, "E-CPU") || strings.HasPrefix(key, "P-CPU") {
				hasClusterInfo = true
				break
			}
		}
	}

	if hasClusterInfo {
		// Use cluster membership information from parsing
		for key := range state.AllSeenCPUs {
			if strings.HasPrefix(key, "E-CPU") {
				// Extract CPU number from "E-CPU0", "E-CPU1", etc.
				cpuNumStr := strings.TrimPrefix(key, "E-CPU")
				if cpuNum, err := strconv.Atoi(cpuNumStr); err == nil {
					ecoreCPUs = append(ecoreCPUs, cpuNum)
				}
			} else if strings.HasPrefix(key, "P-CPU") {
				// Extract CPU number from "P-CPU2", "P-CPU3", etc.
				cpuNumStr := strings.TrimPrefix(key, "P-CPU")
				if cpuNum, err := strconv.Atoi(cpuNumStr); err == nil {
					pcoreCPUs = append(pcoreCPUs, cpuNum)
				}
			}
		}

		// Sort CPU lists for consistent display
		for i := 0; i < len(ecoreCPUs); i++ {
			for j := i + 1; j < len(ecoreCPUs); j++ {
				if ecoreCPUs[i] > ecoreCPUs[j] {
					ecoreCPUs[i], ecoreCPUs[j] = ecoreCPUs[j], ecoreCPUs[i]
				}
			}
		}
		for i := 0; i < len(pcoreCPUs); i++ {
			for j := i + 1; j < len(pcoreCPUs); j++ {
				if pcoreCPUs[i] > pcoreCPUs[j] {
					pcoreCPUs[i], pcoreCPUs[j] = pcoreCPUs[j], pcoreCPUs[i]
				}
			}
		}
	} else {
		// Fallback: No cluster info, categorize all CPUs as P-cores (Intel Mac)
		for cpuID := range state.AllCpuFreq {
			pcoreCPUs = append(pcoreCPUs, cpuID)
		}
		// Sort for consistency
		for i := 0; i < len(pcoreCPUs); i++ {
			for j := i + 1; j < len(pcoreCPUs); j++ {
				if pcoreCPUs[i] > pcoreCPUs[j] {
					pcoreCPUs[i], pcoreCPUs[j] = pcoreCPUs[j], pcoreCPUs[i]
				}
			}
		}
	}

	// Build E-core frequencies
	newECores := make([]int, 0, len(ecoreCPUs))
	for i, cpuID := range ecoreCPUs {
		freq := 0
		if f, exists := state.AllCpuFreq[cpuID]; exists {
			freq = f
		}
		newECores = append(newECores, freq)

		// Update history
		if state.ECoreFreqHistory[i] == nil {
			state.ECoreFreqHistory[i] = make([]float64, 0, 30)
		}
		state.ECoreFreqHistory[i] = models.AddToHistory(
			state.ECoreFreqHistory[i], float64(freq), 30)
	}

	// Build P-core frequencies
	newPCores := make([]int, 0, len(pcoreCPUs))
	for i, cpuID := range pcoreCPUs {
		freq := 0
		if f, exists := state.AllCpuFreq[cpuID]; exists {
			freq = f
		}
		newPCores = append(newPCores, freq)

		// Update history
		if state.PCoreFreqHistory[i] == nil {
			state.PCoreFreqHistory[i] = make([]float64, 0, 30)
		}
		state.PCoreFreqHistory[i] = models.AddToHistory(
			state.PCoreFreqHistory[i], float64(freq), 30)
	}

	// Update state
	state.ECoreFreq = newECores
	state.PCoreFreq = newPCores

	// Update max cores seen
	if len(state.ECoreFreq) > state.MaxECores {
		state.MaxECores = len(state.ECoreFreq)
	}
	if len(state.PCoreFreq) > state.MaxPCores {
		state.MaxPCores = len(state.PCoreFreq)
	}

	// Keep the AllCpuFreq for reference/debugging
}