package parser

// NetworkIOHandler handles network I/O statistics parsing
type NetworkIOHandler struct{}

func (h *NetworkIOHandler) Name() string {
	return "NetworkIO"
}

func (h *NetworkIOHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *NetworkIOHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	if IsSection(line) {
		return StateInSample
	}

	// Parse network I/O
	if matches := networkInRegex.FindStringSubmatch(line); matches != nil {
		if bytes, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.NetworkIn = ConvertToMB(bytes, line)
		}
	}

	if matches := networkOutRegex.FindStringSubmatch(line); matches != nil {
		if bytes, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.NetworkOut = ConvertToMB(bytes, line)
		}
	}

	return StateNetworkIO
}

func (h *NetworkIOHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}