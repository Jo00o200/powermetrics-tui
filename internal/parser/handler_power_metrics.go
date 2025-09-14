package parser

import (
	"strings"
)

// PowerMetricsHandler handles power consumption data parsing
type PowerMetricsHandler struct{}

func (h *PowerMetricsHandler) Name() string {
	return "PowerMetrics"
}

func (h *PowerMetricsHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *PowerMetricsHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	if IsSection(line) {
		return StateInSample
	}

	// Parse power metrics
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

func (h *PowerMetricsHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}