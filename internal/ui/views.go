package ui

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"powermetrics-tui/internal/models"
)

// ViewType represents the current view
type ViewType int

const (
	ViewInterrupts ViewType = iota
	ViewPower
	ViewFrequency
	ViewProcesses
	ViewNetwork
	ViewDisk
	ViewThermal
	ViewBattery
	ViewSystem
	ViewCombined
	ViewCount
)

// DrawInterruptsViewWithHelp draws the CPU interrupts view with optional help and custom start Y
func DrawInterruptsViewWithHelp(screen tcell.Screen, state *models.MetricsState, width, height int, showHelp bool, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "CPU INTERRUPTS", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal))
	if showHelp {
		DrawText(screen, 20, y, "(System events requiring CPU attention)", tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
	y += 2

	// IPI Interrupts with description
	DrawText(screen, 2, y, fmt.Sprintf("IPI:   %8d/s", state.IPICount), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, float64(state.IPICount), 10000, tcell.ColorBlue)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Inter-Processor Interrupts - CPU core communication", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}

	// Sparkline for IPI history
	if len(state.History.IPIHistory) > 0 {
		ipiFloat := make([]float64, len(state.History.IPIHistory))
		for i, v := range state.History.IPIHistory {
			ipiFloat[i] = float64(v)
		}
		DrawSparkline(screen, 25, y, width-30, ipiFloat, tcell.ColorBlue)
		y += 2
	}

	// Timer Interrupts with description
	DrawText(screen, 2, y, fmt.Sprintf("Timer: %8d/s", state.TimerCount), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, float64(state.TimerCount), 10000, tcell.ColorGreen)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Timer events for task scheduling", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}

	// Sparkline for Timer history
	if len(state.History.TimerHistory) > 0 {
		timerFloat := make([]float64, len(state.History.TimerHistory))
		for i, v := range state.History.TimerHistory {
			timerFloat[i] = float64(v)
		}
		DrawSparkline(screen, 25, y, width-30, timerFloat, tcell.ColorGreen)
		y += 2
	}

	// Per-CPU breakdown if available
	if len(state.AllSeenCPUs) > 0 {
		DrawText(screen, 2, y, "Per-CPU Breakdown:", tcell.StyleDefault.Bold(true))
		y++

		// Use all CPUs ever seen for consistent display
		var cpuKeys []string
		for cpu := range state.AllSeenCPUs {
			cpuKeys = append(cpuKeys, cpu)
		}
		sort.Strings(cpuKeys)

		// Display up to 10 CPUs
		for i, cpu := range cpuKeys {
			if i >= 10 || y >= height-2 {
				break
			}

			// Get values, defaulting to 0 if not present in current sample
			ipi := state.PerCPUIPIs[cpu]
			timer := state.PerCPUTimers[cpu]
			total := state.PerCPUInterrupts[cpu]

			// Format: CPU0: Total: 2057/s (IPI: 1197/s, Timer: 713/s)
			text := fmt.Sprintf("%-5s Total:%5.0f/s  IPI:%5.0f/s  Timer:%5.0f/s",
				cpu+":", total, ipi, timer)

			// Color based on activity level
			color := tcell.ColorWhite
			sparkColor := tcell.ColorDarkGray
			if total > 2000 {
				color = tcell.ColorRed
				sparkColor = tcell.ColorRed
			} else if total > 1000 {
				color = tcell.ColorYellow
				sparkColor = tcell.ColorYellow
			} else if total > 500 {
				color = tcell.ColorGreen
				sparkColor = tcell.ColorGreen
			}

			DrawText(screen, 4, y, text, tcell.StyleDefault.Foreground(color))

			// Draw sparkline for this CPU's interrupt history
			if history, ok := state.PerCPUInterruptHistory[cpu]; ok && len(history) > 0 {
				// Draw a compact sparkline next to the text
				DrawSparkline(screen, 58, y, 20, history, sparkColor)
			}

			y++
		}
	} else {
		// Fallback to total if no per-CPU data
		DrawText(screen, 2, y, fmt.Sprintf("Total: %8d/s", state.TotalInterrupts), tcell.StyleDefault)
		DrawBar(screen, 25, y, width-30, float64(state.TotalInterrupts), 20000, tcell.ColorYellow)
		y++
		if showHelp {
			DrawText(screen, 4, y, "All interrupt events combined", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
			y++
		}

		// Sparkline for Total history
		if len(state.History.TotalHistory) > 0 {
			totalFloat := make([]float64, len(state.History.TotalHistory))
			for i, v := range state.History.TotalHistory {
				totalFloat[i] = float64(v)
			}
			DrawSparkline(screen, 25, y, width-30, totalFloat, tcell.ColorYellow)
		}
	}
}


// DrawPowerViewWithHelp draws the power consumption view with optional help and custom start Y
func DrawPowerViewWithHelp(screen tcell.Screen, state *models.MetricsState, width, height int, showHelp bool, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "POWER CONSUMPTION", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow))
	if showHelp {
		DrawText(screen, 20, y, "(Energy usage - affects battery life)", tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
	y += 2

	// CPU Power with description
	DrawText(screen, 2, y, fmt.Sprintf("CPU:    %7.1f mW", state.CPUPower), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.CPUPower, 10000, tcell.ColorRed)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Processor power consumption", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}

	if len(state.History.CPUPowerHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.CPUPowerHistory, tcell.ColorRed)
		y += 2
	}

	// GPU Power
	DrawText(screen, 2, y, fmt.Sprintf("GPU:    %7.1f mW", state.GPUPower), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.GPUPower, 10000, tcell.ColorGreen)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Graphics processor power", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}

	if len(state.History.GPUPowerHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.GPUPowerHistory, tcell.ColorGreen)
		y += 2
	}

	// ANE Power
	DrawText(screen, 2, y, fmt.Sprintf("ANE:    %7.1f mW", state.ANEPower), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.ANEPower, 5000, tcell.ColorBlue)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Apple Neural Engine - AI/ML accelerator", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}
	y++

	// DRAM Power
	DrawText(screen, 2, y, fmt.Sprintf("DRAM:   %7.1f mW", state.DRAMPower), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.DRAMPower, 5000, tcell.ColorPurple)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Memory (RAM) power consumption", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}
	y++

	// System Power
	DrawText(screen, 2, y, fmt.Sprintf("System: %7.1f mW", state.SystemPower), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.SystemPower, 30000, tcell.ColorYellow)
	y++
	if showHelp {
		DrawText(screen, 4, y, "Total system power draw", tcell.StyleDefault.Foreground(tcell.ColorGray).Italic(true))
		y++
	}

	if len(state.History.SystemHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.SystemHistory, tcell.ColorYellow)
	}
}


// DrawFrequencyViewWithStartY draws the CPU/GPU frequency view with custom start Y
func DrawFrequencyViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "FREQUENCY MONITORING", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorPurple))
	y += 2

	// E-Core frequencies (Apple Silicon)
	if len(state.ECoreFreq) > 0 {
		// Find max E-core frequency from history (or use sensible default)
		maxECoreFreq := 2748.0 // M3 max, but we'll adjust based on observed values
		for _, history := range state.ECoreFreqHistory {
			for _, freq := range history {
				if freq > maxECoreFreq {
					maxECoreFreq = freq * 1.1 // Add 10% headroom
				}
			}
		}

		DrawText(screen, 2, y, "E-Cores (Efficiency):", tcell.StyleDefault.Bold(true))
		y++
		for i, freq := range state.ECoreFreq {
			// Show individual CPU cores (E-cores typically start from CPU 0)
			label := fmt.Sprintf("E-Core %d: %4d MHz", i, freq)

			// Color based on activity
			textColor := tcell.StyleDefault
			barColor := tcell.ColorBlue
			if freq == 0 {
				textColor = tcell.StyleDefault.Foreground(tcell.ColorGray)
				barColor = tcell.ColorDarkGray
			}

			DrawText(screen, 4, y, label, textColor)
			DrawBar(screen, 25, y, width-30, float64(freq), maxECoreFreq, barColor)
			y++

			// Draw sparkline for this core's frequency history with lighter color
			if history, ok := state.ECoreFreqHistory[i]; ok && len(history) > 0 {
				sparkColor := tcell.ColorDarkCyan
				if freq == 0 {
					sparkColor = tcell.ColorDarkGray
				}
				DrawSparkline(screen, 25, y, width-30, history, sparkColor)
				y++
			}
		}
		y++
	}

	// P-Core frequencies (Apple Silicon) or all cores (Intel)
	if len(state.PCoreFreq) > 0 {
		// Find max P-core frequency from history (or use sensible default)
		maxPCoreFreq := 4056.0 // M3 Pro/Max can reach this
		isIntel := len(state.ECoreFreq) == 0
		if isIntel {
			maxPCoreFreq = 5000.0 // Intel can go higher
		}

		// Adjust based on observed values
		for _, history := range state.PCoreFreqHistory {
			for _, freq := range history {
				if freq > maxPCoreFreq {
					maxPCoreFreq = freq * 1.1 // Add 10% headroom
				}
			}
		}

		label := "P-Cores (Performance):"
		if isIntel {
			// No E-cores detected, this is likely an Intel Mac
			label = "CPU Cores:"
		}
		DrawText(screen, 2, y, label, tcell.StyleDefault.Bold(true))
		y++
		for i, freq := range state.PCoreFreq {
			// Show individual CPU cores
			coreLabel := fmt.Sprintf("P-Core %d: %4d MHz", i, freq)
			if isIntel {
				coreLabel = fmt.Sprintf("Core %d: %4d MHz", i, freq)
			}

			// Color based on activity
			textColor := tcell.StyleDefault
			barColor := tcell.ColorRed
			if freq == 0 {
				textColor = tcell.StyleDefault.Foreground(tcell.ColorGray)
				barColor = tcell.ColorDarkGray
			}

			DrawText(screen, 4, y, coreLabel, textColor)
			DrawBar(screen, 25, y, width-30, float64(freq), maxPCoreFreq, barColor)
			y++

			// Draw sparkline for this core's frequency history with lighter color
			if history, ok := state.PCoreFreqHistory[i]; ok && len(history) > 0 {
				// Use a lighter shade for sparklines
				sparkColor := tcell.ColorIndianRed
				if isIntel {
					sparkColor = tcell.ColorOrangeRed
				}
				if freq == 0 {
					sparkColor = tcell.ColorDarkGray
				}
				DrawSparkline(screen, 25, y, width-30, history, sparkColor)
				y++
			}
		}
		y++
	}

	// GPU frequency
	if state.GPUFreq > 0 {
		DrawText(screen, 2, y, fmt.Sprintf("GPU:    %4d MHz", state.GPUFreq), tcell.StyleDefault)
		DrawBar(screen, 25, y, width-30, float64(state.GPUFreq), 2000, tcell.ColorGreen)
	}

	// Show a note if no frequency data is available
	if len(state.ECoreFreq) == 0 && len(state.PCoreFreq) == 0 && state.GPUFreq == 0 {
		DrawText(screen, 2, y, "No frequency data available.", tcell.StyleDefault.Foreground(tcell.ColorGray))
		y += 2
		DrawText(screen, 2, y, "Try running with --samplers cpu_power", tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DrawProcessesViewWithStartY draws the top processes view with custom start Y
func DrawProcessesViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int, showOnlyCoalitions bool) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	// Show process counts in the title
	var title string
	if showOnlyCoalitions {
		coalitionCount := 0
		for _, proc := range state.Processes {
			if proc.Name == proc.CoalitionName {
				coalitionCount++
			}
		}
		title = fmt.Sprintf("TOP COALITIONS (%d parents, %d total)", coalitionCount, len(state.Processes))
	} else {
		title = fmt.Sprintf("TOP PROCESSES (%d active, %d exited)", len(state.Processes), len(state.RecentlyExited))
	}
	DrawText(screen, 2, y, title, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal))
	y += 2

	// Header - properly aligned with exact spacing, with sparkline columns
	header := fmt.Sprintf("%-8s %-28s %7s %12s %12s %12s  %-10s %-10s",
		"PID", "Process", "CPU%", "Memory", "Disk", "Network", "CPU Hist", "Mem Hist")
	DrawText(screen, 2, y, header, tcell.StyleDefault.Bold(true))
	y++

	// Filter and sort processes by CPU usage
	var processes []models.ProcessInfo
	if showOnlyCoalitions {
		// Show only coalition parent processes - convert from ProcessCoalition to ProcessInfo format
		for _, coalition := range state.Coalitions {
			// Convert ProcessCoalition to ProcessInfo for consistent display
			coalitionAsProcess := models.ProcessInfo{
				PID:           coalition.CoalitionID,
				Name:          coalition.Name,
				CoalitionName: coalition.Name, // Coalition is its own parent
				CPUPercent:    coalition.CPUPercent,
				MemoryMB:      coalition.MemoryMB,
				DiskMB:        coalition.DiskMB,
				NetworkMB:     coalition.NetworkMB,
				CPUHistory:    coalition.CPUHistory,
				MemoryHistory: coalition.MemoryHistory,
			}
			processes = append(processes, coalitionAsProcess)
		}
	} else {
		// Show all processes
		processes = make([]models.ProcessInfo, len(state.Processes))
		copy(processes, state.Processes)
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	// Display as many processes as can fit on screen
	maxProcesses := height - y - 2 // Leave 2 lines for bottom border
	if maxProcesses > 30 {
		maxProcesses = 30 // Cap at 30 to keep it readable
	}

	for i, proc := range processes {
		if i >= maxProcesses || y >= height-2 {
			break
		}

		// Check if this is a coalition parent process (process name matches coalition name)
		isCoalition := proc.Name == proc.CoalitionName

		// Truncate long process names
		processName := proc.Name
		if len(processName) > 28 {
			processName = processName[:25] + "..."
		}

		// Format the line with proper column alignment
		line := fmt.Sprintf("%-8d %-28s %6.1f%% %10.1fMB %10.1fMB %10.1fMB",
			proc.PID, processName, proc.CPUPercent, proc.MemoryMB, proc.DiskMB, proc.NetworkMB)

		// Color based on process type and CPU usage
		var color tcell.Color
		var sparkColor tcell.Color

		if isCoalition {
			// Coalition parent processes - use teal/cyan colors
			if proc.CPUPercent > 50 {
				color = tcell.ColorDarkRed
				sparkColor = tcell.ColorDarkRed
			} else if proc.CPUPercent > 25 {
				color = tcell.ColorOrange
				sparkColor = tcell.ColorOrange
			} else {
				color = tcell.ColorTeal
				sparkColor = tcell.ColorTeal
			}
		} else {
			// Subprocess - use standard colors
			if proc.CPUPercent > 50 {
				color = tcell.ColorRed
				sparkColor = tcell.ColorRed
			} else if proc.CPUPercent > 25 {
				color = tcell.ColorYellow
				sparkColor = tcell.ColorYellow
			} else {
				color = tcell.ColorWhite
				sparkColor = tcell.ColorTeal
			}
		}

		DrawText(screen, 2, y, line, tcell.StyleDefault.Foreground(color))

		// Draw sparkline for CPU history (starts at column 88)
		if len(proc.CPUHistory) > 0 {
			DrawCPUSparkline(screen, 88, y, 10, proc.CPUHistory, sparkColor)
		}

		// Draw sparkline for Memory history (starts at column 99)
		if len(proc.MemoryHistory) > 0 {
			// Memory sparkline in MB, scale appropriately
			memColor := tcell.ColorBlue
			if proc.MemoryMB > 500 {
				memColor = tcell.ColorRed
			} else if proc.MemoryMB > 200 {
				memColor = tcell.ColorYellow
			}
			DrawSparkline(screen, 99, y, 10, proc.MemoryHistory, memColor)
		}

		y++
	}

	// Display recently exited processes if there's room
	if len(state.RecentlyExited) > 0 && y < height-3 {
		y += 2
		exitedTitle := fmt.Sprintf("RECENTLY EXITED PROCESSES (showing %d of %d)",
			min(10, len(state.RecentlyExited)), len(state.RecentlyExited))
		DrawText(screen, 2, y, exitedTitle, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGray))
		y++

		// Header for exited processes with verification column
		exitHeader := fmt.Sprintf("%-30s %12s %15s %-20s %8s",
			"Process", "Occurrences", "Last Seen", "PIDs", "Verified")
		DrawText(screen, 2, y, exitHeader, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGray))
		y++

		// Display up to 10 most recent exited processes
		displayCount := len(state.RecentlyExited)
		if displayCount > 10 {
			displayCount = 10
		}

		// Sort by occurrences (highest first), then by last exit time
		sortedExited := make([]models.ExitedProcessInfo, len(state.RecentlyExited))
		copy(sortedExited, state.RecentlyExited)
		sort.Slice(sortedExited, func(i, j int) bool {
			// Primary sort: by occurrences (descending)
			if sortedExited[i].Occurrences != sortedExited[j].Occurrences {
				return sortedExited[i].Occurrences > sortedExited[j].Occurrences
			}
			// Secondary sort: by last exit time (most recent first)
			return sortedExited[i].LastExitTime.After(sortedExited[j].LastExitTime)
		})

		for i := 0; i < displayCount && y < height-1; i++ {
			proc := sortedExited[i]

			// Truncate long process names
			processName := proc.Name
			if len(processName) > 29 {
				processName = processName[:26] + "..."
			}

			// Format PIDs list (show up to 5 PIDs, then ...)
			pidsStr := ""
			if len(proc.PIDs) > 0 {
				if len(proc.PIDs) <= 5 {
					// Show all PIDs
					pidStrings := make([]string, len(proc.PIDs))
					for i, pid := range proc.PIDs {
						pidStrings[i] = fmt.Sprintf("%d", pid)
					}
					pidsStr = strings.Join(pidStrings, ",")
				} else {
					// Show first 4 PIDs and count
					pidStrings := make([]string, 4)
					for i := 0; i < 4; i++ {
						pidStrings[i] = fmt.Sprintf("%d", proc.PIDs[i])
					}
					pidsStr = strings.Join(pidStrings, ",") + fmt.Sprintf(",+%d", len(proc.PIDs)-4)
				}
			}

			// Format last seen time
			duration := time.Since(proc.LastExitTime)
			lastSeenStr := ""
			if duration.Seconds() < 60 {
				lastSeenStr = fmt.Sprintf("%ds ago", int(duration.Seconds()))
			} else if duration.Minutes() < 60 {
				lastSeenStr = fmt.Sprintf("%dm ago", int(duration.Minutes()))
			} else {
				lastSeenStr = fmt.Sprintf("%dh ago", int(duration.Hours()))
			}

			// Format occurrences with 'x' suffix
			occurrencesStr := fmt.Sprintf("%dx", proc.Occurrences)
			if proc.Occurrences == 1 {
				occurrencesStr = "1x"
			}

			// Check if all PIDs are actually dead
			verificationStatus := "✓" // checkmark for all dead
			verificationColor := tcell.ColorGreen

			for _, pid := range proc.PIDs {
				psCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "pid=")
				if err := psCmd.Run(); err == nil {
					// Process is still alive!
					verificationStatus = "✗" // X for some alive
					verificationColor = tcell.ColorRed
					break
				}
			}

			line := fmt.Sprintf("%-30s %12s %15s %-20s",
				processName, occurrencesStr, lastSeenStr, pidsStr)

			// Use different shades of gray based on how recent
			color := tcell.ColorGray
			if duration.Minutes() < 1 {
				color = tcell.ColorSilver
			}

			DrawText(screen, 2, y, line, tcell.StyleDefault.Foreground(color))
			// Draw verification status with appropriate color
			DrawText(screen, 2+len(line)+1, y, verificationStatus, tcell.StyleDefault.Foreground(verificationColor))
			y++
		}
	}
}

// DrawNetworkViewWithStartY draws the network I/O view with custom start Y
func DrawNetworkViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "NETWORK I/O", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGreen))
	y += 2

	// Network In
	DrawText(screen, 2, y, fmt.Sprintf("In:  %8.2f MB/s", state.NetworkIn), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.NetworkIn, 100, tcell.ColorGreen)
	y++

	if len(state.History.NetworkInHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.NetworkInHistory, tcell.ColorGreen)
		y += 2
	}

	// Network Out
	DrawText(screen, 2, y, fmt.Sprintf("Out: %8.2f MB/s", state.NetworkOut), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.NetworkOut, 100, tcell.ColorBlue)
	y++

	if len(state.History.NetworkOutHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.NetworkOutHistory, tcell.ColorBlue)
		y += 2
	}

	// Total throughput
	total := state.NetworkIn + state.NetworkOut
	DrawText(screen, 2, y, fmt.Sprintf("Total: %7.2f MB/s", total), tcell.StyleDefault.Bold(true))
}

// DrawDiskViewWithStartY draws the disk I/O view with custom start Y
func DrawDiskViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "DISK I/O", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorPurple))
	y += 2

	// Disk Read
	DrawText(screen, 2, y, fmt.Sprintf("Read:  %8.2f MB/s", state.DiskRead), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.DiskRead, 500, tcell.ColorGreen)
	y++

	if len(state.History.DiskReadHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.DiskReadHistory, tcell.ColorGreen)
		y += 2
	}

	// Disk Write
	DrawText(screen, 2, y, fmt.Sprintf("Write: %8.2f MB/s", state.DiskWrite), tcell.StyleDefault)
	DrawBar(screen, 25, y, width-30, state.DiskWrite, 500, tcell.ColorRed)
	y++

	if len(state.History.DiskWriteHistory) > 0 {
		DrawSparkline(screen, 25, y, width-30, state.History.DiskWriteHistory, tcell.ColorRed)
		y += 2
	}

	// Total throughput
	total := state.DiskRead + state.DiskWrite
	DrawText(screen, 2, y, fmt.Sprintf("Total: %7.2f MB/s", total), tcell.StyleDefault.Bold(true))
}

// DrawThermalViewWithStartY draws the thermal monitoring view with custom start Y
func DrawThermalViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "THERMAL STATUS", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorRed))
	y += 2

	// Thermal pressure
	pressureColor := tcell.ColorGreen
	if state.ThermalPressure == "Heavy" || state.ThermalPressure == "Critical" {
		pressureColor = tcell.ColorRed
	} else if state.ThermalPressure == "Moderate" {
		pressureColor = tcell.ColorYellow
	}

	DrawText(screen, 2, y, fmt.Sprintf("Thermal Pressure: %s", state.ThermalPressure),
		tcell.StyleDefault.Foreground(pressureColor))
	y += 2

	// Temperatures
	if len(state.Temperature) > 0 {
		DrawText(screen, 2, y, "Temperature Sensors:", tcell.StyleDefault.Bold(true))
		y++

		// Sort temperature sensors by name for consistent display
		var sensors []string
		for sensor := range state.Temperature {
			sensors = append(sensors, sensor)
		}
		sort.Strings(sensors)

		for _, sensor := range sensors {
			temp := state.Temperature[sensor]
			tempColor := GetColorForValue(temp, 50, 80)

			line := fmt.Sprintf("%-30s: %5.1f°C", sensor, temp)
			DrawText(screen, 4, y, line, tcell.StyleDefault.Foreground(tempColor))

			// Draw temperature bar
			DrawBar(screen, 45, y, width-50, temp, 100, tempColor)
			y++

			if y >= height-2 {
				break
			}
		}
	}

	// Average temperature history
	if len(state.History.TempHistory) > 0 && y < height-3 {
		y++
		DrawText(screen, 2, y, "Average Temperature History:", tcell.StyleDefault.Bold(true))
		y++
		DrawSparkline(screen, 4, y, width-10, state.History.TempHistory, tcell.ColorYellow)
	}
}

// DrawBatteryViewWithStartY draws the battery status view with custom start Y
func DrawBatteryViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "BATTERY STATUS", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGreen))
	y += 2

	// Battery charge
	chargeColor := tcell.ColorGreen
	if state.BatteryCharge < 20 {
		chargeColor = tcell.ColorRed
	} else if state.BatteryCharge < 50 {
		chargeColor = tcell.ColorYellow
	}

	DrawText(screen, 2, y, fmt.Sprintf("Charge: %.1f%%", state.BatteryCharge),
		tcell.StyleDefault.Foreground(chargeColor))

	// Draw battery bar
	DrawBar(screen, 25, y, width-30, state.BatteryCharge, 100, chargeColor)
	y += 2

	// Battery state
	if state.BatteryState != "" {
		stateColor := tcell.ColorWhite
		if state.BatteryState == "charging" {
			stateColor = tcell.ColorGreen
		} else if state.BatteryState == "discharging" {
			stateColor = tcell.ColorYellow
		}

		DrawText(screen, 2, y, fmt.Sprintf("State: %s", state.BatteryState),
			tcell.StyleDefault.Foreground(stateColor))
		y += 2
	}

	// Battery history
	if len(state.History.BatteryHistory) > 0 {
		DrawText(screen, 2, y, "Charge History:", tcell.StyleDefault.Bold(true))
		y++
		DrawSparkline(screen, 4, y, width-10, state.History.BatteryHistory, chargeColor)
		y += 2
	}

	// System power
	DrawText(screen, 2, y, fmt.Sprintf("System Power: %.1f W", state.SystemPower/1000),
		tcell.StyleDefault)
	y++

	if len(state.History.SystemHistory) > 0 {
		DrawSparkline(screen, 4, y, width-10, state.History.SystemHistory, tcell.ColorYellow)
	}
}

// DrawSystemViewWithStartY draws the system overview with custom start Y
func DrawSystemViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "SYSTEM OVERVIEW", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal))
	y += 2

	// Memory usage
	if state.MemoryUsed > 0 || state.MemoryAvailable > 0 {
		total := state.MemoryUsed + state.MemoryAvailable
		usagePercent := (state.MemoryUsed / total) * 100

		DrawText(screen, 2, y, fmt.Sprintf("Memory: %.1f GB / %.1f GB (%.1f%%)",
			state.MemoryUsed/1024, total/1024, usagePercent), tcell.StyleDefault)

		memColor := GetColorForValue(usagePercent, 60, 80)
		DrawBar(screen, 4, y+1, width-10, usagePercent, 100, memColor)
		y += 3

		if len(state.History.MemoryHistory) > 0 {
			DrawSparkline(screen, 4, y, width-10, state.History.MemoryHistory, memColor)
			y += 2
		}
	}

	// Swap usage
	if state.SwapUsed > 0 {
		DrawText(screen, 2, y, fmt.Sprintf("Swap: %.1f GB", state.SwapUsed/1024), tcell.StyleDefault)
		y += 2
	}

	// Quick stats
	DrawText(screen, 2, y, "Quick Stats:", tcell.StyleDefault.Bold(true))
	y++

	stats := []string{
		fmt.Sprintf("CPU Power:  %.1f W", state.CPUPower/1000),
		fmt.Sprintf("GPU Power:  %.1f W", state.GPUPower/1000),
		fmt.Sprintf("Network:    ↓%.1f ↑%.1f MB/s", state.NetworkIn, state.NetworkOut),
		fmt.Sprintf("Disk:       ↓%.1f ↑%.1f MB/s", state.DiskRead, state.DiskWrite),
		fmt.Sprintf("Battery:    %.0f%%", state.BatteryCharge),
		fmt.Sprintf("Thermal:    %s", state.ThermalPressure),
	}

	for _, stat := range stats {
		if y >= height-2 {
			break
		}
		DrawText(screen, 4, y, stat, tcell.StyleDefault)
		y++
	}

	// Last update time
	if !state.LastUpdate.IsZero() {
		updateText := fmt.Sprintf("Last update: %s", state.LastUpdate.Format(time.Kitchen))
		DrawText(screen, width-len(updateText)-2, height-2, updateText,
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// DrawCombinedView draws a combined view of all metrics
func DrawCombinedViewWithStartY(screen tcell.Screen, state *models.MetricsState, width, height int, startY int) {
	state.Mu.RLock()
	defer state.Mu.RUnlock()

	y := startY
	DrawText(screen, 2, y, "SYSTEM METRICS", tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal))
	y += 2

	// Compact display of all metrics
	sections := []struct {
		title string
		lines []string
	}{
		{
			"CPU",
			[]string{
				fmt.Sprintf("IPI: %d  Timer: %d  Total: %d", state.IPICount, state.TimerCount, state.TotalInterrupts),
				fmt.Sprintf("Power: %.1fW", state.CPUPower/1000),
			},
		},
		{
			"GPU",
			[]string{
				fmt.Sprintf("Power: %.1fW  Freq: %dMHz", state.GPUPower/1000, state.GPUFreq),
			},
		},
		{
			"Memory",
			[]string{
				fmt.Sprintf("Used: %.1fGB  Swap: %.1fGB", state.MemoryUsed/1024, state.SwapUsed/1024),
			},
		},
		{
			"Network",
			[]string{
				fmt.Sprintf("In: %.1fMB/s  Out: %.1fMB/s", state.NetworkIn, state.NetworkOut),
			},
		},
		{
			"Disk",
			[]string{
				fmt.Sprintf("Read: %.1fMB/s  Write: %.1fMB/s", state.DiskRead, state.DiskWrite),
			},
		},
		{
			"System",
			[]string{
				fmt.Sprintf("Power: %.1fW  Battery: %.0f%%  Thermal: %s",
					state.SystemPower/1000, state.BatteryCharge, state.ThermalPressure),
			},
		},
	}

	for _, section := range sections {
		if y >= height-2 {
			break
		}

		DrawText(screen, 2, y, section.title+":", tcell.StyleDefault.Bold(true))
		y++

		for _, line := range section.lines {
			if y >= height-2 {
				break
			}
			DrawText(screen, 4, y, line, tcell.StyleDefault)
			y++
		}
		y++
	}
}