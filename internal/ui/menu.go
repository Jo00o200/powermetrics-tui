package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
)

// ViewInfo contains information about each view
type ViewInfo struct {
	Name     string
	Shortcut string
	Icon     string
}

// GetViewInfo returns information about all views
func GetViewInfo() []ViewInfo {
	return []ViewInfo{
		{Name: "Interrupts", Shortcut: "1", Icon: "⚡"},
		{Name: "Power", Shortcut: "2", Icon: "🔋"},
		{Name: "Frequency", Shortcut: "3", Icon: "📊"},
		{Name: "Processes", Shortcut: "4", Icon: "📱"},
		{Name: "Network", Shortcut: "5", Icon: "🌐"},
		{Name: "Disk", Shortcut: "6", Icon: "💾"},
		{Name: "Thermal", Shortcut: "7", Icon: "🌡️"},
		{Name: "Battery", Shortcut: "8", Icon: "🔌"},
		{Name: "System", Shortcut: "9", Icon: "💻"},
		{Name: "Combined", Shortcut: "0", Icon: "📈"},
	}
}

// DrawMenuBar draws the menu bar at the top of the screen
func DrawMenuBar(screen tcell.Screen, width int, currentView ViewType) {
	views := GetViewInfo()

	// Draw background bar
	for x := 0; x < width; x++ {
		screen.SetContent(x, 0, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDarkBlue))
	}

	// Calculate spacing
	x := 2
	for i, view := range views {
		// Determine if this is the current view
		isCurrent := ViewType(i) == currentView

		// Create the menu item text
		menuItem := fmt.Sprintf(" %s %s ", view.Shortcut, view.Name)

		// Set style based on whether it's selected
		style := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
		if isCurrent {
			style = tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack).Bold(true)
		}

		// Draw the menu item
		for _, ch := range menuItem {
			screen.SetContent(x, 0, ch, nil, style)
			x++
		}

		// Add separator if not last item
		if i < len(views)-1 {
			screen.SetContent(x, 0, '│', nil, tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorGray))
			x++
		}
	}
}

// DrawCompactMenuBar draws a more compact menu bar
func DrawCompactMenuBar(screen tcell.Screen, width int, currentView ViewType) int {
	views := GetViewInfo()

	y := 0
	// Draw title bar
	title := "PowerMetrics TUI"
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)

	// Fill the entire line with background
	for x := 0; x < width; x++ {
		screen.SetContent(x, y, ' ', nil, titleStyle)
	}

	// Draw title
	x := 2
	for _, ch := range title {
		screen.SetContent(x, y, ch, nil, titleStyle)
		x++
	}

	y++

	// Draw view tabs on the second line
	x = 2
	for i, view := range views {
		isCurrent := ViewType(i) == currentView

		// Use brackets for current view
		var menuItem string
		if isCurrent {
			menuItem = fmt.Sprintf("[%s]", view.Name)
		} else {
			menuItem = fmt.Sprintf(" %s ", view.Name)
		}

		// Set style
		style := tcell.StyleDefault.Foreground(tcell.ColorGray)
		if isCurrent {
			style = tcell.StyleDefault.Foreground(tcell.ColorTeal).Bold(true)
		}

		// Draw the menu item
		for _, ch := range menuItem {
			screen.SetContent(x, y, ch, nil, style)
			x++
		}

		// Add space between items
		if i < len(views)-1 {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
			x++
		}
	}

	return y + 1 // Return the next available y position
}