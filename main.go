package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"powermetrics-tui/internal/models"
	"powermetrics-tui/internal/parser"
	"powermetrics-tui/internal/ui"
)

var (
	samplers     = flag.String("samplers", "default", "Comma-separated list of samplers (interrupts,cpu_power,gpu_power,thermal,battery,tasks,all,default)")
	interval     = flag.Int("interval", 1000, "Sampling interval in milliseconds")
	combined     = flag.Bool("combined", false, "Show all metrics in combined view")
	debug        = flag.Bool("debug", false, "Enable debug output")
	currentView  ui.ViewType
	metricsState *models.MetricsState
	showHelp     bool = true // Show descriptions by default for casual users
)

func main() {
	flag.Parse()

	// Initialize state
	metricsState = models.NewMetricsState()

	// Initialize tcell screen
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("Error creating screen: %v", err)
	}
	if err := screen.Init(); err != nil {
		log.Fatalf("Error initializing screen: %v", err)
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.Clear()

	// Determine which samplers to use
	samplerList := determineSamplers()

	// Start powermetrics monitoring
	go runPowerMetrics(samplerList)

	// Main event loop
	eventChan := make(chan tcell.Event)
	go func() {
		for {
			eventChan <- screen.PollEvent()
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-eventChan:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyEscape || ev.Rune() == 'q' || ev.Rune() == 'Q' {
					return
				}
				if ev.Key() == tcell.KeyTab {
					currentView = (currentView + 1) % ui.ViewCount
				}
				if ev.Key() == tcell.KeyCtrlC {
					return
				}
				if ev.Rune() == 'h' || ev.Rune() == 'H' || ev.Rune() == '?' {
					showHelp = !showHelp // Toggle help descriptions
				}
				// Number key shortcuts for quick view switching
				if ev.Rune() >= '1' && ev.Rune() <= '9' {
					currentView = ui.ViewType(ev.Rune() - '1')
				}
				if ev.Rune() == '0' {
					currentView = ui.ViewCombined
				}
			case *tcell.EventResize:
				screen.Clear()
			}

		case <-ticker.C:
			drawUI(screen)
		}
	}
}

func determineSamplers() string {
	if *combined {
		return "all"
	}

	// Map view names to powermetrics samplers
	samplerMap := map[string]string{
		"interrupts": "interrupts",
		"cpu_power":  "cpu_power",
		"gpu_power":  "gpu_power",
		"thermal":    "thermal,smc",
		"battery":    "battery",
		"tasks":      "tasks",
		"processes":  "tasks",
		"all":        "all",
		"default":    "default",
	}

	samplerParts := strings.Split(*samplers, ",")
	var result []string

	for _, s := range samplerParts {
		s = strings.TrimSpace(s)
		if mapped, ok := samplerMap[s]; ok {
			if mapped != "" {
				result = append(result, mapped)
			}
		} else {
			result = append(result, s)
		}
	}

	if len(result) == 0 {
		return "default"
	}

	// Determine initial view based on samplers
	if result[0] == "all" || result[0] == "default" || *combined {
		currentView = ui.ViewInterrupts  // Start with interrupts view when all samplers are enabled
	} else if strings.Contains(result[0], "interrupts") {
		currentView = ui.ViewInterrupts
	} else if strings.Contains(result[0], "cpu_power") || strings.Contains(result[0], "gpu_power") {
		currentView = ui.ViewPower
	} else if strings.Contains(result[0], "thermal") || strings.Contains(result[0], "smc") {
		currentView = ui.ViewThermal
	} else if strings.Contains(result[0], "battery") {
		currentView = ui.ViewBattery
	}

	return strings.Join(result, ",")
}

func runPowerMetrics(samplerList string) {
	for {
		args := []string{
			"powermetrics",
			"--samplers", samplerList,
			"-i", fmt.Sprintf("%d", *interval),
			"-n", "1",
		}

		cmd := exec.Command("sudo", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			metricsState.Mu.Lock()
			metricsState.UpdateErrors++
			// Store error message for debugging
			if *debug && len(output) > 0 {
				fmt.Fprintf(os.Stderr, "powermetrics error: %s\n", string(output))
			}
			metricsState.Mu.Unlock()
			time.Sleep(time.Duration(*interval) * time.Millisecond)
			continue
		}

		if *debug {
			fmt.Fprintf(os.Stderr, "powermetrics output (%d bytes)\n", len(output))
			// Save to file for inspection
			os.WriteFile("/tmp/powermetrics_debug.txt", output, 0644)
		}

		parser.ParsePowerMetricsOutput(string(output), metricsState)
		metricsState.Mu.Lock()
		metricsState.LastUpdate = time.Now()
		metricsState.Mu.Unlock()

		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
}

func drawUI(screen tcell.Screen) {
	screen.Clear()
	width, height := screen.Size()

	// Draw the menu bar at the top and get the next Y position
	startY := ui.DrawCompactMenuBar(screen, width, currentView)

	// Draw view based on current selection, starting from the correct Y position
	switch currentView {
	case ui.ViewInterrupts:
		ui.DrawInterruptsViewWithHelp(screen, metricsState, width, height, showHelp, startY)
	case ui.ViewPower:
		ui.DrawPowerViewWithHelp(screen, metricsState, width, height, showHelp, startY)
	case ui.ViewFrequency:
		ui.DrawFrequencyViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewProcesses:
		ui.DrawProcessesViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewNetwork:
		ui.DrawNetworkViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewDisk:
		ui.DrawDiskViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewThermal:
		ui.DrawThermalViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewBattery:
		ui.DrawBatteryViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewSystem:
		ui.DrawSystemViewWithStartY(screen, metricsState, width, height, startY)
	case ui.ViewCombined:
		ui.DrawCombinedViewWithStartY(screen, metricsState, width, height, startY)
	}

	// Draw footer
	drawFooter(screen, width, height)

	screen.Show()
}

func drawFooter(screen tcell.Screen, width, height int) {
	footer := " 1-9,0: Jump to View | Tab: Next | H: Help | Q: Quit "

	// Show current view name
	viewNames := []string{
		"Interrupts", "Power", "Frequency", "Processes", "Network",
		"Disk", "Thermal", "Battery", "System", "Combined",
	}

	if int(currentView) < len(viewNames) {
		status := fmt.Sprintf("[%s]", viewNames[currentView])
		ui.DrawText(screen, 2, height-1, status, tcell.StyleDefault.Foreground(tcell.ColorTeal))

		// Show help status
		if showHelp {
			ui.DrawText(screen, 2+len(status)+1, height-1, "📖", tcell.StyleDefault)
		}
	}

	// Draw controls on the right
	ui.DrawText(screen, width-len(footer)-2, height-1, footer,
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	// Show error indicator if there are update errors
	metricsState.Mu.RLock()
	errors := metricsState.UpdateErrors
	metricsState.Mu.RUnlock()

	if errors > 0 {
		errMsg := fmt.Sprintf(" Errors: %d ", errors)
		ui.DrawText(screen, width/2-len(errMsg)/2, height-1, errMsg,
			tcell.StyleDefault.Foreground(tcell.ColorRed))
	}
}

func init() {
	// Check if we have sudo access
	cmd := exec.Command("sudo", "-n", "true")
	if err := cmd.Run(); err != nil {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("PowerMetrics TUI requires sudo access.")
		fmt.Println("Please run: sudo -v")
		fmt.Println("Then restart this application.")
		fmt.Print("Press Enter to exit...")
		scanner.Scan()
		os.Exit(1)
	}
}