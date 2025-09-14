package parser

import (
	"strings"
)

// ProcessorUsageHandler handles the Processor usage section
type ProcessorUsageHandler struct{}

func (h *ProcessorUsageHandler) Name() string {
	return "ProcessorUsage"
}

func (h *ProcessorUsageHandler) Enter(ctx *ParserContext) {
	// Initialize frequency map if needed
	if ctx.MetricsState.AllCpuFreq == nil {
		ctx.MetricsState.AllCpuFreq = make(map[int]int)
	}
}

func (h *ProcessorUsageHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for section transitions
	if IsSection(line) {
		return StateInSample
	}

	// Track that we're processing lines in this handler
	// (for debugging - will remove later)

	// Parse CPU frequency data
	if matches := cpuFreqRegex.FindStringSubmatch(line); matches != nil {
		cpuID, _ := ParseInt(matches[1])
		freq, _ := ParseInt(matches[2])
		if ctx.MetricsState.AllCpuFreq == nil {
			ctx.MetricsState.AllCpuFreq = make(map[int]int)
		}
		ctx.MetricsState.AllCpuFreq[cpuID] = freq
		// Track that we parsed at least one CPU frequency
		ctx.MetricsState.AllSeenCPUs["found-cpu-freq"] = true
	}

	// Parse cluster frequencies (stored but not used directly)
	if ecoreFreqRegex.MatchString(line) {
		ctx.CurrentCluster = "E"
	} else if pcoreFreqRegex.MatchString(line) {
		ctx.CurrentCluster = "P"
	}

	// Parse GPU frequency
	if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
		freq, _ := ParseInt(matches[1])
		ctx.MetricsState.GPUFreq = freq
	}

	// Parse power metrics (these appear at the end of Processor usage section)
	if matches := cpuPowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.CPUPower = val
		}
	}

	if matches := gpuPowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.GPUPower = val
		}
	}

	if matches := anePowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.ANEPower = val
		}
	}

	if matches := dramPowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.DRAMPower = val
		}
	}

	if matches := systemPowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			// Convert watts to milliwatts if needed
			if strings.Contains(line, "Watts") {
				val *= 1000
			}
			ctx.MetricsState.SystemPower = val
		}
	}

	return StateProcessorUsage
}

func (h *ProcessorUsageHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}
