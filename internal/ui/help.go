package ui

import (
	"github.com/gdamore/tcell/v2"
)

// MetricDescriptions provides user-friendly explanations for technical terms
var MetricDescriptions = map[string]string{
	// Interrupt metrics
	"IPI": "Inter-Processor Interrupts - Communication between CPU cores for coordination",
	"Timer": "Timer Interrupts - Regular system timekeeping and task scheduling events",
	"Total IRQ": "Total Interrupts - All hardware/software events requiring CPU attention",

	// Power metrics
	"CPU Power": "Processor power consumption - Higher values mean more battery drain",
	"GPU Power": "Graphics processor power - Increases during video/gaming/rendering",
	"ANE Power": "Apple Neural Engine - AI/ML acceleration chip power usage",
	"DRAM Power": "Memory power consumption - RAM energy usage",
	"System Power": "Total system power draw - Overall energy consumption",

	// Frequency metrics
	"E-Cores": "Efficiency Cores - Low-power cores for background tasks (Apple Silicon)",
	"P-Cores": "Performance Cores - High-power cores for demanding tasks (Apple Silicon)",
	"CPU Frequency": "Clock speed in MHz - Higher means faster but more power",
	"GPU Frequency": "Graphics clock speed - Adjusts based on graphics workload",

	// Thermal metrics
	"Thermal Pressure": "System thermal state - 'Nominal' is normal, 'Heavy' means throttling",
	"Temperature": "Component temperatures in Celsius - Higher temps may reduce performance",

	// Battery metrics
	"Battery Charge": "Current battery level percentage",
	"Battery State": "Charging, discharging, or AC powered status",

	// Memory metrics
	"Memory Used": "RAM currently in use by applications",
	"Memory Available": "Free RAM available for new applications",
	"Swap Used": "Disk space used as virtual memory when RAM is full",

	// Network/Disk metrics
	"Network In/Out": "Data received/sent over network connections",
	"Disk Read/Write": "Data read from/written to storage drives",

	// Process metrics
	"CPU%": "Percentage of CPU time used by process",
	"Memory MB": "RAM used by process in megabytes",
}

// GetDescription returns a user-friendly description for a metric
func GetDescription(metric string) string {
	if desc, ok := MetricDescriptions[metric]; ok {
		return desc
	}
	return ""
}

// DrawHelpFooter draws contextual help at the bottom of the screen
func DrawHelpFooter(screen tcell.Screen, width, height int, context string) {
	if desc := GetDescription(context); desc != "" {
		// Clear the help line
		ClearLine(screen, height-3, width)

		// Draw help text in gray
		helpText := "ℹ " + desc
		if len(helpText) > width-4 {
			helpText = helpText[:width-7] + "..."
		}
		DrawText(screen, 2, height-3, helpText, tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
	}
}