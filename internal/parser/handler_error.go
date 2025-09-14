package parser

// ErrorHandler handles error recovery state
type ErrorHandler struct{}

func (h *ErrorHandler) Name() string {
	return "Error"
}

func (h *ErrorHandler) Enter(ctx *ParserContext) {
	// Log that we entered error state if debug is enabled
	if ctx.DebugEnabled {
		// Error handling initialization
	}
}

func (h *ErrorHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Try to recover by looking for a new sample
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	// Stay in error state until we find a new sample boundary
	return StateError
}

func (h *ErrorHandler) Exit(ctx *ParserContext) {
	// Reset any error conditions
	ctx.CurrentCPU = ""
	ctx.CurrentCluster = ""
	ctx.CurrentCoalition = nil
}