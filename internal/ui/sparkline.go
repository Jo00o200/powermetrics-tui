package ui

import (
	"math"
	"strings"
)

// Sparkline characters for different levels (8 levels)
var sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline creates a sparkline string from float64 values
func RenderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return strings.Repeat("─", width)
	}

	// If we have more values than width, take the last 'width' values
	if len(values) > width {
		values = values[len(values)-width:]
	}

	// Build sparkline
	var result strings.Builder

	// If we have fewer values than width, start with placeholder
	if len(values) < width {
		// Add placeholder for missing data
		for i := 0; i < width-len(values); i++ {
			result.WriteRune('─')
		}
	}

	for _, v := range values {
		// Scale value to 0-7 range (8 levels) based on 0-100% CPU
		scaled := v / 100.0 * 7.0
		index := int(math.Round(scaled))
		if index < 0 {
			index = 0
		}
		if index >= len(sparklineChars) {
			index = len(sparklineChars) - 1
		}
		result.WriteRune(sparklineChars[index])
	}

	return result.String()
}