package parser

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	gpuFreqRegex   = regexp.MustCompile(`(?:GPU HW active frequency|GPU active frequency|GPU frequency):\s+([0-9]+)\s*MHz`)

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

	// GPU usage patterns
	gpuActiveRegex = regexp.MustCompile(`GPU (?:HW )?active residency:\s+([0-9.]+)%`)

	// Backlight pattern
	backlightRegex = regexp.MustCompile(`Backlight level:\s+([0-9]+)`)
)

// Parser maintains a persistent state machine for parsing powermetrics output
type Parser struct {
	stateMachine *StateMachine
	state        *models.MetricsState
}

// NewParser creates a new parser with a persistent state machine
func NewParser(state *models.MetricsState) *Parser {
	return &Parser{
		stateMachine: NewStateMachine(state),
		state:        state,
	}
}

// ParseOutput parses powermetrics output using the persistent state machine
func (p *Parser) ParseOutput(output string) {
	p.state.Mu.Lock()
	defer p.state.Mu.Unlock()

	// Initialize history if needed
	if p.state.History == nil {
		p.state.History = &models.HistoricalData{
			MaxHistory: 30,
		}
	}

	// Initialize maps if needed
	if p.state.AllSeenCPUs == nil {
		p.state.AllSeenCPUs = make(map[string]bool)
	}
	if p.state.PerCPUInterrupts == nil {
		p.state.PerCPUInterrupts = make(map[string]float64)
	}
	if p.state.PerCPUIPIs == nil {
		p.state.PerCPUIPIs = make(map[string]float64)
	}
	if p.state.PerCPUTimers == nil {
		p.state.PerCPUTimers = make(map[string]float64)
	}
	if p.state.CoalitionCPUHistory == nil {
		p.state.CoalitionCPUHistory = make(map[int][]float64)
	}
	if p.state.CoalitionMemHistory == nil {
		p.state.CoalitionMemHistory = make(map[int][]float64)
	}
	if p.state.CoalitionNames == nil {
		p.state.CoalitionNames = make(map[int]string)
	}
	if p.state.ProcessCPUHistory == nil {
		p.state.ProcessCPUHistory = make(map[int][]float64)
	}
	if p.state.ProcessMemHistory == nil {
		p.state.ProcessMemHistory = make(map[int][]float64)
	}
	if p.state.ECoreFreqHistory == nil {
		p.state.ECoreFreqHistory = make(map[int][]float64)
	}
	if p.state.PCoreFreqHistory == nil {
		p.state.PCoreFreqHistory = make(map[int][]float64)
	}
	if p.state.AllCpuFreq == nil {
		p.state.AllCpuFreq = make(map[int]int)
	}
	if p.state.PerCPUInterruptHistory == nil {
		p.state.PerCPUInterruptHistory = make(map[string][]float64)
	}

	// Process each line through the persistent state machine
	lines := strings.Split(output, "\n")
	for _, rawLine := range lines {
		if err := p.stateMachine.ProcessLine(rawLine); err != nil {
			// Log error to debug file, not console
			if debugFile, err2 := os.OpenFile("/tmp/powermetrics-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err2 == nil {
				debugFile.WriteString(fmt.Sprintf("[%s] Parser error: %v\n",
					time.Now().Format("15:04:05"), err))
				debugFile.Close()
			}
		}
	}

	// Get the accumulated values from the state machine context BEFORE finalizing
	// (FinalizeCurrentState transitions to WaitingForSample which resets these)
	ctx := p.stateMachine.GetContext()

	// Update interrupt totals
	if ctx.IPITotal > 0 {
		p.state.IPICount = int(ctx.IPITotal)
	}
	if ctx.TimerTotal > 0 {
		p.state.TimerCount = int(ctx.TimerTotal)
	}
	if ctx.InterruptsTotal > 0 {
		p.state.TotalInterrupts = int(ctx.InterruptsTotal)
	}

	// Force final processing if we're in RunningTasks state
	// This ensures the Exit method is called to commit the data
	p.stateMachine.FinalizeCurrentState()

	// Note: Process and coalition tracking is now handled in the TasksHandler's Exit method

	// Organize CPU frequencies based on what we detected
	organizeCPUFrequencies(p.state)

	// Update per-CPU interrupt history for all known CPUs
	// Since we reset all CPUs to 0 at the start, this will include zeros for missing CPUs
	for cpu := range p.state.AllSeenCPUs {
		total := p.state.PerCPUInterrupts[cpu] // Will be 0 if CPU wasn't in this sample
		if p.state.PerCPUInterruptHistory[cpu] == nil {
			p.state.PerCPUInterruptHistory[cpu] = make([]float64, 0, 30)
		}
		p.state.PerCPUInterruptHistory[cpu] = models.AddToHistory(p.state.PerCPUInterruptHistory[cpu], total, 30)
	}

	// Update history
	p.state.History.IPIHistory = models.AddToIntHistory(p.state.History.IPIHistory, p.state.IPICount, p.state.History.MaxHistory)
	p.state.History.TimerHistory = models.AddToIntHistory(p.state.History.TimerHistory, p.state.TimerCount, p.state.History.MaxHistory)
	p.state.History.TotalHistory = models.AddToIntHistory(p.state.History.TotalHistory, p.state.TotalInterrupts, p.state.History.MaxHistory)
	p.state.History.CPUPowerHistory = models.AddToHistory(p.state.History.CPUPowerHistory, p.state.CPUPower, p.state.History.MaxHistory)
	p.state.History.GPUPowerHistory = models.AddToHistory(p.state.History.GPUPowerHistory, p.state.GPUPower, p.state.History.MaxHistory)
	p.state.History.SystemHistory = models.AddToHistory(p.state.History.SystemHistory, p.state.SystemPower, p.state.History.MaxHistory)
	p.state.History.NetworkInHistory = models.AddToHistory(p.state.History.NetworkInHistory, p.state.NetworkIn, p.state.History.MaxHistory)
	p.state.History.NetworkOutHistory = models.AddToHistory(p.state.History.NetworkOutHistory, p.state.NetworkOut, p.state.History.MaxHistory)
	p.state.History.DiskReadHistory = models.AddToHistory(p.state.History.DiskReadHistory, p.state.DiskRead, p.state.History.MaxHistory)
	p.state.History.DiskWriteHistory = models.AddToHistory(p.state.History.DiskWriteHistory, p.state.DiskWrite, p.state.History.MaxHistory)
	p.state.History.BatteryHistory = models.AddToHistory(p.state.History.BatteryHistory, p.state.BatteryCharge, p.state.History.MaxHistory)
	p.state.History.MemoryHistory = models.AddToHistory(p.state.History.MemoryHistory, p.state.MemoryUsed, p.state.History.MaxHistory)

	// Update GPU frequency history
	p.state.GPUFreqHistory = models.AddToHistory(p.state.GPUFreqHistory, float64(p.state.GPUFreq), 30)

	// Update average temperature
	if len(p.state.Temperature) > 0 {
		var avgTemp float64
		for _, temp := range p.state.Temperature {
			avgTemp += temp
		}
		avgTemp /= float64(len(p.state.Temperature))
		p.state.History.TempHistory = models.AddToHistory(p.state.History.TempHistory, avgTemp, p.state.History.MaxHistory)
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

// ParsePowerMetricsOutput is the legacy API that creates a new parser each time
// DEPRECATED: Use NewParser().ParseOutput() for better performance and to save bad samples
func ParsePowerMetricsOutput(output string, state *models.MetricsState) {
	parser := NewParser(state)
	parser.ParseOutput(output)
}