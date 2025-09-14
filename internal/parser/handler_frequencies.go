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
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	if IsSection(line) {
		return StateInSample
	}

	// Parse E-core frequencies
	if matches := ecoreFreqRegex.FindStringSubmatch(line); matches != nil {
		if freq, err := ParseInt(matches[1]); err == nil {
			if ctx.MetricsState.ECoreFreq == nil {
				ctx.MetricsState.ECoreFreq = make([]int, 0)
			}
			ctx.MetricsState.ECoreFreq = append(ctx.MetricsState.ECoreFreq, freq)
		}
	}

	// Parse P-core frequencies
	if matches := pcoreFreqRegex.FindStringSubmatch(line); matches != nil {
		if freq, err := ParseInt(matches[1]); err == nil {
			if ctx.MetricsState.PCoreFreq == nil {
				ctx.MetricsState.PCoreFreq = make([]int, 0)
			}
			ctx.MetricsState.PCoreFreq = append(ctx.MetricsState.PCoreFreq, freq)
		}
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