package ui

import (
	"math"
	"strings"
)

// Sparkline characters for different levels (8 levels)
var sparklineChars = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline creates a sparkline string from float64 values
func RenderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return strings.Repeat(" ", width)
	}

	// If we have more values than width, take the last 'width' values
	if len(values) > width {
		values = values[len(values)-width:]
	}

	// Find min and max for scaling
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// If all values are the same, show middle bar
	if max == min {
		if max == 0 {
			return strings.Repeat(string(sparklineChars[0]), width)
		}
		return strings.Repeat(string(sparklineChars[4]), width)
	}

	// Build sparkline
	var result strings.Builder
	for _, v := range values {
		// Scale value to 0-7 range (8 levels)
		scaled := (v - min) / (max - min) * 7
		index := int(math.Round(scaled))
		if index < 0 {
			index = 0
		}
		if index >= len(sparklineChars)-1 {
			index = len(sparklineChars) - 1
		}
		result.WriteRune(sparklineChars[index])
	}

	// Pad with spaces if needed
	for result.Len() < width {
		result.WriteRune(' ')
	}

	return result.String()
}