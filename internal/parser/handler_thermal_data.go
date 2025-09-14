package parser

import (
	"strings"
)

// ThermalDataHandler handles thermal and temperature data parsing
type ThermalDataHandler struct{}

func (h *ThermalDataHandler) Name() string {
	return "ThermalData"
}

func (h *ThermalDataHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *ThermalDataHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions

	if IsSection(line) {
		return StateInSample
	}

	// Parse thermal pressure
	if matches := thermalRegex.FindStringSubmatch(line); matches != nil {
		ctx.MetricsState.ThermalPressure = matches[1]
	}

	// Parse temperature data
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

func (h *ThermalDataHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}