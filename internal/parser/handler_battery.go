package parser

// BatteryHandler handles the Battery and backlight usage section
type BatteryHandler struct{}

func (h *BatteryHandler) Name() string {
	return "Battery"
}

func (h *BatteryHandler) Enter(ctx *ParserContext) {
	// Nothing special needed
}

func (h *BatteryHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	// Check for section transitions
	if IsSection(line) {
		return StateInSample
	}

	// Parse battery statistics
	if matches := batteryRegex.FindStringSubmatch(line); matches != nil {
		if charge, err := ParseFloat(matches[1]); err == nil {
			ctx.MetricsState.BatteryCharge = charge
		}
	}

	if matches := batteryStateRegex.FindStringSubmatch(line); matches != nil {
		ctx.MetricsState.BatteryState = matches[1]
	}

	// Parse backlight level
	if matches := backlightRegex.FindStringSubmatch(line); matches != nil {
		if level, err := ParseInt(matches[1]); err == nil {
			ctx.MetricsState.BacklightLevel = level
		}
	}

	return StateBattery
}

func (h *BatteryHandler) Exit(ctx *ParserContext) {
	// Nothing special needed
}