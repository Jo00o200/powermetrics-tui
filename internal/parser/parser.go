package parser

import (
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

	// Old format: absolute counts (keeping for compatibility)
	ipiRegex         = regexp.MustCompile(`CPU (\d+) IPI:\s+(\d+)`)
	timerRegex       = regexp.MustCompile(`CPU (\d+) Timer:\s+(\d+)`)
	totalRegex       = regexp.MustCompile(`CPU (\d+) Total:\s+(\d+)`)
	cpuPowerRegex    = regexp.MustCompile(`(?:CPU Power|CPU Energy|Combined Power \(CPU\)):\s+([0-9.]+)\s*mW`)
	gpuPowerRegex    = regexp.MustCompile(`(?:GPU Power|GPU Energy|Combined Power \(GPU\)):\s+([0-9.]+)\s*mW`)
	anePowerRegex    = regexp.MustCompile(`(?:ANE Power|ANE Energy|Combined Power \(ANE\)):\s+([0-9.]+)\s*mW`)
	dramPowerRegex   = regexp.MustCompile(`(?:DRAM Power|DRAM Energy|Combined Power \(DRAM\)):\s+([0-9.]+)\s*mW`)
	systemPowerRegex = regexp.MustCompile(`(?:Combined Power|System Power|System Average).*?:\s+([0-9.]+)\s*(?:mW|Watts)`)
	thermalRegex     = regexp.MustCompile(`Thermal Pressure:\s+(\w+)`)
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
	networkInRegex  = regexp.MustCompile(`(?:Bytes In|RX Bytes|Network In):\s+([0-9.]+)\s*(?:MB|KB|GB|bytes)`)
	networkOutRegex = regexp.MustCompile(`(?:Bytes Out|TX Bytes|Network Out):\s+([0-9.]+)\s*(?:MB|KB|GB|bytes)`)

	// Disk I/O patterns
	diskReadRegex  = regexp.MustCompile(`(?:Disk Read|Read Bytes|Bytes Read):\s+([0-9.]+)\s*(?:MB|KB|GB|bytes)`)
	diskWriteRegex = regexp.MustCompile(`(?:Disk Write|Write Bytes|Bytes Written):\s+([0-9.]+)\s*(?:MB|KB|GB|bytes)`)

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

// ParsePowerMetricsOutput parses the output from powermetrics command
func ParsePowerMetricsOutput(output string, state *models.MetricsState) {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	lines := strings.Split(output, "\n")

	// Reset accumulators
	ipiTotal := 0.0
	timerTotal := 0.0
	interrupts := 0.0
	state.ECoreFreq = []int{}
	state.PCoreFreq = []int{}
	state.Processes = []models.ProcessInfo{}

	inProcessSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for process section - new format "*** Running tasks ***"
		if strings.Contains(line, "*** Running tasks ***") {
			inProcessSection = true
			continue
		}

		// Skip the header line
		if inProcessSection && strings.Contains(line, "Name") && strings.Contains(line, "ID") {
			continue
		}

		// Exit process section when we hit another *** section
		if inProcessSection && strings.HasPrefix(line, "***") {
			inProcessSection = false
		}

		// Parse interrupts - new format (interrupts/sec)
		if matches := ipiRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				ipiTotal += val
			}
		}

		if matches := timerRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				timerTotal += val
			}
		}

		if matches := totalRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				interrupts += val
			}
		}

		// Parse interrupts - old format (absolute counts)
		if matches := ipiRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				ipiTotal += float64(val)
			}
		}

		if matches := timerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				timerTotal += float64(val)
			}
		}

		if matches := totalRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				interrupts += float64(val)
			}
		}

		// Parse power metrics
		if matches := cpuPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.CPUPower = val
			}
		}

		if matches := gpuPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.GPUPower = val
			}
		}

		if matches := anePowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.ANEPower = val
			}
		}

		if matches := dramPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DRAMPower = val
			}
		}

		if matches := systemPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// Convert watts to milliwatts if needed
				if strings.Contains(line, "Watts") {
					val *= 1000
				}
				state.SystemPower = val
			}
		}

		// Parse cluster frequencies - these tell us the architecture
		if matches := ecoreFreqRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				// This is an Apple Silicon Mac with E-cores
				if len(state.ECoreFreq) == 0 {
					state.ECoreFreq = []int{val}
				}
			}
		}

		if matches := pcoreFreqRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				// This is an Apple Silicon Mac with P-cores
				state.PCoreFreq = append(state.PCoreFreq, val)
			}
		}

		// Parse individual CPU frequencies
		if matches := cpuFreqRegex.FindStringSubmatch(line); matches != nil {
			if cpuNum, err := strconv.Atoi(matches[1]); err == nil {
				if freq, err := strconv.Atoi(matches[2]); err == nil && freq > 0 {
					// Store all CPU frequencies in a temporary map first
					// We'll organize them based on what clusters we detected
					if state.AllCpuFreq == nil {
						state.AllCpuFreq = make(map[int]int)
					}
					state.AllCpuFreq[cpuNum] = freq
				}
			}
		}

		if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				state.GPUFreq = val
			}
		}

		// Parse network
		if matches := networkInRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.NetworkIn = convertToMB(val, line)
			}
		}

		if matches := networkOutRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.NetworkOut = convertToMB(val, line)
			}
		}

		// Parse disk I/O
		if matches := diskReadRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DiskRead = convertToMB(val, line)
			}
		}

		if matches := diskWriteRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DiskWrite = convertToMB(val, line)
			}
		}

		// Parse battery
		if matches := batteryRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.BatteryCharge = val
			}
		}

		if matches := batteryStateRegex.FindStringSubmatch(line); matches != nil {
			state.BatteryState = matches[1]
		}

		// Parse thermal
		if matches := thermalRegex.FindStringSubmatch(line); matches != nil {
			state.ThermalPressure = matches[1]
		}

		// Parse temperatures
		if matches := tempRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[2], 64); err == nil {
				state.Temperature[matches[1]] = val
			}
		}

		// Parse memory
		if matches := memoryUsedRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.MemoryUsed = convertToMB(val, line)
			}
		}

		if matches := memoryAvailRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.MemoryAvailable = convertToMB(val, line)
			}
		}

		if matches := swapUsedRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.SwapUsed = convertToMB(val, line)
			}
		}

		// Parse processes from "Running tasks" section
		// Format: Name(padded) ID CPU_ms/s User% ...
		if inProcessSection {
			if matches := processRegex.FindStringSubmatch(line); matches != nil {
				// matches[1] = Name (with spaces)
				// matches[2] = ID (PID)
				// matches[3] = CPU ms/s
				// matches[4] = User%

				name := strings.TrimSpace(matches[1])
				pid, _ := strconv.Atoi(matches[2])
				cpuMs, _ := strconv.ParseFloat(matches[3], 64)
				userPercent, _ := strconv.ParseFloat(matches[4], 64)

				// Convert CPU ms/s to percentage (approximate)
				// 1000 ms/s = 100% of one core
				cpuPercent := cpuMs / 10.0

				state.Processes = append(state.Processes, models.ProcessInfo{
					PID:        pid,
					Name:       name,
					CPUPercent: cpuPercent,
					MemoryMB:   userPercent, // Using User% as a proxy for now
					DiskMB:     0,
					NetworkMB:  0,
				})
			}
		}
	}

	// Organize CPU frequencies based on what we detected
	organizeCPUFrequencies(state)

	// Update interrupt totals (convert float rates to int for display)
	if ipiTotal > 0 {
		state.IPICount = int(ipiTotal)
	}
	if timerTotal > 0 {
		state.TimerCount = int(timerTotal)
	}
	if interrupts > 0 {
		state.TotalInterrupts = int(interrupts)
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

// organizeCPUFrequencies intelligently organizes CPU frequencies based on detected architecture
func organizeCPUFrequencies(state *models.MetricsState) {
	if state.AllCpuFreq == nil || len(state.AllCpuFreq) == 0 {
		return
	}

	// Find the range of CPU numbers
	minCPU, maxCPU := 999, -1
	for cpu := range state.AllCpuFreq {
		if cpu < minCPU {
			minCPU = cpu
		}
		if cpu > maxCPU {
			maxCPU = cpu
		}
	}

	hasECores := len(state.ECoreFreq) > 0
	hasPCores := len(state.PCoreFreq) > 0

	if hasECores || hasPCores {
		// Apple Silicon with explicit E/P core clusters
		// We need to figure out which CPUs belong to which cluster

		// Strategy: Look for patterns in the CPU numbering and frequencies
		// Usually E-cores are lower numbered and have lower max frequencies

		// First, find the natural break between E-cores and P-cores
		// E-cores typically have lower frequencies
		frequencies := make([]int, 0, len(state.AllCpuFreq))
		for _, freq := range state.AllCpuFreq {
			frequencies = append(frequencies, freq)
		}

		// Simple heuristic: CPUs with consistently lower frequencies are likely E-cores
		// This works because E-cores typically max out around 2.5 GHz while P-cores go up to 3.5+ GHz

		// Clear and rebuild the frequency arrays with individual CPU data
		if hasECores {
			state.ECoreFreq = []int{}
		}
		if hasPCores {
			state.PCoreFreq = []int{}
		}

		// Look for a gap in CPU numbering or frequency patterns
		for i := minCPU; i <= maxCPU; i++ {
			if freq, exists := state.AllCpuFreq[i]; exists {
				// Simple heuristic: first few CPUs are usually E-cores on Apple Silicon
				// More sophisticated detection could look at frequency ranges
				if hasECores && i < 2 { // Typical M1/M2 has 2-4 E-cores numbered first
					state.ECoreFreq = append(state.ECoreFreq, freq)
				} else if hasPCores {
					state.PCoreFreq = append(state.PCoreFreq, freq)
				}
			}
		}
	} else {
		// Intel Mac or unknown architecture - just show all cores as regular cores
		// Convert map to sorted array
		state.PCoreFreq = []int{}
		for i := minCPU; i <= maxCPU; i++ {
			if freq, exists := state.AllCpuFreq[i]; exists {
				state.PCoreFreq = append(state.PCoreFreq, freq)
			}
		}
	}

	// Clear the temporary map
	state.AllCpuFreq = nil
}