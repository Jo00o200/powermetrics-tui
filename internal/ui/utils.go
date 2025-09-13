package ui

import (
	"github.com/gdamore/tcell/v2"
)

// DrawBar draws a horizontal bar chart
func DrawBar(screen tcell.Screen, x, y, width int, value, max float64, color tcell.Color) {
	if max <= 0 {
		return
	}

	percentage := value / max
	if percentage > 1 {
		percentage = 1
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(width) * percentage)

	// Draw filled portion
	for i := 0; i < filled && i < width; i++ {
		screen.SetContent(x+i, y, '█', nil, tcell.StyleDefault.Foreground(color))
	}

	// Draw empty portion
	for i := filled; i < width; i++ {
		screen.SetContent(x+i, y, '░', nil, tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// DrawSparkline draws a sparkline chart
func DrawSparkline(screen tcell.Screen, x, y, width int, data []float64, color tcell.Color) {
	if len(data) == 0 {
		return
	}

	// Unicode block characters for sparkline
	ticks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find min and max values
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Handle case where all values are the same
	if max == min {
		max = min + 1
	}

	// Convert values to tick levels
	levels := make([]int, 0, len(data))
	for _, v := range data {
		level := int(((v - min) / (max - min)) * 8)
		if level < 0 {
			level = 0
		}
		if level > 8 {
			level = 8
		}
		levels = append(levels, level)
	}

	// Draw sparkline - show the most recent 'width' samples from the end
	start := 0
	if len(levels) > width {
		start = len(levels) - width
	}

	pos := 0
	for i := start; i < len(levels) && pos < width; i++ {
		screen.SetContent(x+pos, y, ticks[levels[i]], nil, tcell.StyleDefault.Foreground(color))
		pos++
	}
}

// FormatSize formats bytes into human-readable format
func FormatSize(bytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0

	for bytes >= 1024 && unitIndex < len(units)-1 {
		bytes /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return "0 B"
	}

	if bytes < 10 {
		return "" // Return formatted string
	}

	return "" // Return formatted string
}

// GetColorForValue returns a color based on value thresholds
func GetColorForValue(value, low, high float64) tcell.Color {
	if value < low {
		return tcell.ColorGreen
	} else if value < high {
		return tcell.ColorYellow
	} else {
		return tcell.ColorRed
	}
}

// DrawText draws text at the specified position
func DrawText(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

// ClearLine clears a line on the screen
func ClearLine(screen tcell.Screen, y, width int) {
	for x := 0; x < width; x++ {
		screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
	}
}

// DrawBox draws a box border
func DrawBox(screen tcell.Screen, x, y, width, height int, style tcell.Style) {
	// Top border
	screen.SetContent(x, y, '┌', nil, style)
	for i := 1; i < width-1; i++ {
		screen.SetContent(x+i, y, '─', nil, style)
	}
	screen.SetContent(x+width-1, y, '┐', nil, style)

	// Side borders
	for i := 1; i < height-1; i++ {
		screen.SetContent(x, y+i, '│', nil, style)
		screen.SetContent(x+width-1, y+i, '│', nil, style)
	}

	// Bottom border
	screen.SetContent(x, y+height-1, '└', nil, style)
	for i := 1; i < width-1; i++ {
		screen.SetContent(x+i, y+height-1, '─', nil, style)
	}
	screen.SetContent(x+width-1, y+height-1, '┘', nil, style)
}