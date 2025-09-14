package parser

// DiskIOHandler handles disk I/O statistics parsing
type DiskIOHandler struct{}

func (h *DiskIOHandler) Name() string {
	return "DiskIO"
}

func (h *DiskIOHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *DiskIOHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for transitions
	if IsNewSample(line) {
		return StateWaitingForSample
	}

	if IsSection(line) {
		return StateInSample
	}

	// Parse disk I/O
	if matches := diskReadRegex.FindStringSubmatch(line); matches != nil {
		if bytes, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.DiskRead = ConvertToMB(bytes, line)
		}
	}

	if matches := diskWriteRegex.FindStringSubmatch(line); matches != nil {
		if bytes, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.DiskWrite = ConvertToMB(bytes, line)
		}
	}

	return StateDiskIO
}

func (h *DiskIOHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}