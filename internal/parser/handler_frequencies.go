package parser

// FrequenciesHandler handles CPU and GPU frequency parsing
type FrequenciesHandler struct{}

func (h *FrequenciesHandler) Name() string {
	return "Frequencies"
}

func (h *FrequenciesHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *FrequenciesHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions

	if IsSection(line) {
		return StateInSample
	}

	// Parse E-core frequencies
	// For now, we track these for debugging but don't store in state
	if ecoreFreqRegex.MatchString(line) {
		ctx.CurrentCluster = "E"
	}

	// Parse P-core frequencies
	// For now, we track these for debugging but don't store in state
	if pcoreFreqRegex.MatchString(line) {
		ctx.CurrentCluster = "P"
	}

	// Parse GPU frequency
	if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
		if freq, err := ParseInt(matches[1]); err == nil {
			ctx.MetricsState.GPUFreq = freq
		}
	}

	// Parse individual CPU frequencies
	if matches := cpuFreqRegex.FindStringSubmatch(line); matches != nil {
		if cpuNum, err := ParseInt(matches[1]); err == nil {
			if freq, err := ParseInt(matches[2]); err == nil {
				if ctx.MetricsState.AllCpuFreq == nil {
					ctx.MetricsState.AllCpuFreq = make(map[int]int)
				}
				ctx.MetricsState.AllCpuFreq[cpuNum] = freq
			}
		}
	}

	return StateFrequencies
}

func (h *FrequenciesHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}