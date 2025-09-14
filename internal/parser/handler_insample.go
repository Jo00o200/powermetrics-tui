package parser

import (
	"strings"
)

// InSampleHandler handles the general in-sample state, routing to specific section handlers
type InSampleHandler struct{}

func (h *InSampleHandler) Name() string {
	return "InSample"
}

func (h *InSampleHandler) Enter(ctx *ParserContext) {
	// Nothing special needed when entering general sample state
}

func (h *InSampleHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	_ = strings.TrimSpace(line) // trimmed variable for future use

	// Check for new sample (transition back to waiting)

	// Check for specific sections
	if IsRunningTasks(line) {
		return StateRunningTasks
	}

	if IsSection(line) {
		// For other sections, stay in general sample state
		// Individual parsers will be triggered by regex matches
		return StateInSample
	}

	// Try to match specific patterns and route to appropriate states
	if matches := cpuInterruptRegex.FindStringSubmatch(line); matches != nil {
		ctx.CurrentCPU = "CPU" + matches[1]
		// Mark this CPU as seen
		if ctx.MetricsState.AllSeenCPUs == nil {
			ctx.MetricsState.AllSeenCPUs = make(map[string]bool)
		}
		ctx.MetricsState.AllSeenCPUs[ctx.CurrentCPU] = true
		return StateCPUInterrupts
	}

	// Check for power metrics
	if cpuPowerRegex.MatchString(line) || gpuPowerRegex.MatchString(line) ||
	   anePowerRegex.MatchString(line) || dramPowerRegex.MatchString(line) ||
	   systemPowerRegex.MatchString(line) {
		// Parse power metrics inline
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
		return StatePowerMetrics
	}

	// Check for frequency data
	if ecoreFreqRegex.MatchString(line) || pcoreFreqRegex.MatchString(line) ||
	   gpuFreqRegex.MatchString(line) || cpuFreqRegex.MatchString(line) {
		return StateFrequencies
	}

	// Check for network I/O
	if networkInRegex.MatchString(line) || networkOutRegex.MatchString(line) {
		return StateNetworkIO
	}

	// Check for disk I/O
	if diskReadRegex.MatchString(line) || diskWriteRegex.MatchString(line) {
		return StateDiskIO
	}

	// Check for memory stats
	if memoryUsedRegex.MatchString(line) || memoryAvailRegex.MatchString(line) || swapUsedRegex.MatchString(line) {
		return StateMemoryStats
	}

	// Check for thermal data
	if thermalRegex.MatchString(line) || tempRegex.MatchString(line) {
		// Parse thermal inline
		if matches := thermalRegex.FindStringSubmatch(line); matches != nil {
			ctx.MetricsState.ThermalPressure = matches[1]
		}
		if matches := tempRegex.FindStringSubmatch(line); matches != nil {
			sensorName := strings.TrimSpace(matches[1])
			if temp, err := ParseFloat(matches[2]); err == nil {
				if ctx.MetricsState.Temperature == nil {
					ctx.MetricsState.Temperature = make(map[string]float64)
				}
				ctx.MetricsState.Temperature[sensorName] = temp
			}
		}
		return StateThermalData
	}

	// Check for battery data
	if batteryRegex.MatchString(line) || batteryStateRegex.MatchString(line) {
		// Parse battery inline since it's simple
		if matches := batteryRegex.FindStringSubmatch(line); matches != nil {
			if charge, err := ParseFloat(matches[1]); err == nil {
				ctx.MetricsState.BatteryCharge = charge
			}
		}
		if matches := batteryStateRegex.FindStringSubmatch(line); matches != nil {
			ctx.MetricsState.BatteryState = matches[1]
		}
	}

	// Track cluster headers for frequency parsing context
	// Use the new utility functions instead of hardcoded checks
	if IsECluster(line) {
		ctx.CurrentCluster = "E"
	} else if IsPCluster(line) {
		ctx.CurrentCluster = "P"
	}

	return StateInSample
}

func (h *InSampleHandler) Exit(ctx *ParserContext) {
	// Nothing special needed when leaving general sample state
}