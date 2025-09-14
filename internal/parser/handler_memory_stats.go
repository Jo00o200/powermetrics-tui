package parser

// MemoryStatsHandler handles memory statistics parsing
type MemoryStatsHandler struct{}

func (h *MemoryStatsHandler) Name() string {
	return "MemoryStats"
}

func (h *MemoryStatsHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *MemoryStatsHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	if IsSection(line) {
		return StateInSample
	}

	// Parse memory stats
	if matches := memoryUsedRegex.FindStringSubmatch(line); matches != nil {
		if mb, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.MemoryUsed = mb
		}
	}

	if matches := memoryAvailRegex.FindStringSubmatch(line); matches != nil {
		if mb, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.MemoryAvailable = mb
		}
	}

	if matches := swapUsedRegex.FindStringSubmatch(line); matches != nil {
		if mb, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.SwapUsed = mb
		}
	}

	return StateMemoryStats
}

func (h *MemoryStatsHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}