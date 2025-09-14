package parser

// CPUInterruptsHandler handles CPU interrupt parsing
type CPUInterruptsHandler struct{}

func (h *CPUInterruptsHandler) Name() string {
	return "CPUInterrupts"
}

func (h *CPUInterruptsHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *CPUInterruptsHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions out of this state
	if IsSection(line) {
		return StateInSample
	}

	// Check for new CPU header
	if matches := cpuInterruptRegex.FindStringSubmatch(line); matches != nil {
		ctx.CurrentCPU = "CPU" + matches[1]
		if ctx.MetricsState.AllSeenCPUs == nil {
			ctx.MetricsState.AllSeenCPUs = make(map[string]bool)
		}
		ctx.MetricsState.AllSeenCPUs[ctx.CurrentCPU] = true
		return StateCPUInterrupts
	}

	// Parse interrupt data
	if matches := ipiRateRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.IPITotal += val
			if ctx.CurrentCPU != "" {
				ctx.MetricsState.PerCPUIPIs[ctx.CurrentCPU] = val
				ctx.MetricsState.AllSeenCPUs[ctx.CurrentCPU] = true
			}
		}
	}

	if matches := timerRateRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.TimerTotal += val
			if ctx.CurrentCPU != "" {
				ctx.MetricsState.PerCPUTimers[ctx.CurrentCPU] = val
				ctx.MetricsState.AllSeenCPUs[ctx.CurrentCPU] = true
			}
		}
	}

	if matches := totalRateRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.InterruptsTotal += val
			if ctx.CurrentCPU != "" {
				ctx.MetricsState.PerCPUInterrupts[ctx.CurrentCPU] = val
				ctx.MetricsState.AllSeenCPUs[ctx.CurrentCPU] = true
			}
		}
	}

	// Parse old format interrupts (absolute counts) for compatibility
	if matches := ipiRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseInt(matches[2]); err == nil {
			ctx.IPITotal += float64(val)
		}
	}

	if matches := timerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseInt(matches[2]); err == nil {
			ctx.TimerTotal += float64(val)
		}
	}

	if matches := totalRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseInt(matches[2]); err == nil {
			ctx.InterruptsTotal += float64(val)
		}
	}

	return StateCPUInterrupts
}

func (h *CPUInterruptsHandler) Exit(ctx *ParserContext) {
	// Clear current CPU context when leaving
	ctx.CurrentCPU = ""
}