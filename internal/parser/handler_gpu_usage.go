package parser

// GPUUsageHandler handles the GPU usage section
type GPUUsageHandler struct{}

func (h *GPUUsageHandler) Name() string {
	return "GPUUsage"
}

func (h *GPUUsageHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *GPUUsageHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for section transitions
	if IsSection(line) {
		return StateInSample
	}

	// Parse GPU usage statistics
	// GPU Active residency: 2.61% (130.67 ms)
	if matches := gpuActiveRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.GPUActive = val
		}
	}

	// GPU frequency
	if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
		if freq, err := ParseInt(matches[1]); err == nil {
			ctx.MetricsState.GPUFreq = freq
		}
	}

	// GPU Power
	if matches := gpuPowerRegex.FindStringSubmatch(line); matches != nil {
		if val, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.GPUPower = val
		}
	}

	return StateGPUUsage
}

func (h *GPUUsageHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}