package parser

// SFIHandler handles the Selective Forced Idle section
type SFIHandler struct{}

func (h *SFIHandler) Name() string {
	return "SFI"
}

func (h *SFIHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *SFIHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for section transitions
	if IsSection(line) {
		return StateInSample
	}

	// Parse SFI statistics
	// For now, we don't parse SFI data but the handler is here for completeness
	// and to properly handle state transitions

	return StateSFI
}

func (h *SFIHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}