package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"powermetrics-tui/internal/models"
	"powermetrics-tui/internal/parser"
)

type ProcessDelta struct {
	Added   []ProcessInfo
	Removed []ProcessInfo
	Changed []ProcessInfo
}

type ProcessInfo struct {
	PID        int
	Name       string
	Coalition  string
	CPUPercent float64
}

func main() {
	fmt.Println("=== PowerMetrics Process Delta Analyzer ===")
	fmt.Println("This tool analyzes process parsing consistency")
	fmt.Println("It verifies if 'missing' processes are truly dead using ps")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// PIDs to specifically watch - from user's report of false positives
	watchPIDs := []int{65560, 2089, 2579}
	if len(watchPIDs) > 0 {
		fmt.Printf("Watching specific PIDs: %v\n", watchPIDs)
		fmt.Printf("These are reported as dead but are actually alive\n\n")
	}

	// Run powermetrics with same settings as main app
	args := []string{
		"powermetrics",
		"--samplers", "tasks",
		"-i", "1000", // 1 second interval
		"--show-all",
	}

	cmd := exec.Command("sudo", args...)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting powermetrics: %v\n", err)
		return
	}

	// Read output continuously
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for large outputs
	buf := make([]byte, 0, 512*1024)
	scanner.Buffer(buf, 512*1024)

	var outputBuffer strings.Builder

	// State tracking flags for proper section parsing
	type ParserState int
	const (
		StateWaitingForSample ParserState = iota
		StateInSample
		StateInTaskSection
	)

	currentState := StateWaitingForSample
	sampleCount := 0

	// Track processes across samples
	previousProcesses := make(map[int]ProcessInfo)
	processSeenCount := make(map[int]int) // Track how many times we've seen each PID

	// Create a single state object to maintain history across samples
	// This is critical for exit detection to work properly!
	persistentState := models.NewMetricsState()

	maxSamples := 5  // Only run 5 iterations
	for scanner.Scan() {
		line := scanner.Text()

		// State machine for proper section tracking
		switch {
		case strings.Contains(line, "*** Sampled system activity"):
			// We found a new sample marker
			if currentState == StateInSample && outputBuffer.Len() > 0 {
				// Parse the previous complete sample
				sampleContent := outputBuffer.String()
				sampleCount++

				// Check for multiple samples/sections in buffer
				sampledActivityCount := strings.Count(sampleContent, "*** Sampled system activity")
				runningTasksCount := strings.Count(sampleContent, "*** Running tasks ***")
				if sampledActivityCount > 0 {
					fmt.Printf("\n⚠️ Buffer: %d 'Sampled activity', ", sampledActivityCount)
				}
				if runningTasksCount != 1 {
					fmt.Printf("%d 'Running tasks' sections! ", runningTasksCount)
				}
				fmt.Printf("\nSample %d: ", sampleCount)

				// Count lines in the Running tasks section using proper state tracking
				type AnalyzerState int
				const (
					AnalyzerStateSearching AnalyzerState = iota
					AnalyzerStateInTasks
				)

				analyzerState := AnalyzerStateSearching
				taskLineCount := 0
				coalitionCount := 0
				subprocessCount := 0
				taskSectionCount := 0

				for _, line := range strings.Split(sampleContent, "\n") {
					switch analyzerState {
					case AnalyzerStateSearching:
						if strings.Contains(line, "*** Running tasks ***") {
							taskSectionCount++
							analyzerState = AnalyzerStateInTasks
						}

					case AnalyzerStateInTasks:
						// Check for section end conditions
						if strings.HasPrefix(line, "***") {
							analyzerState = AnalyzerStateSearching
							continue
						}
						if strings.HasPrefix(line, "ALL_TASKS") {
							// ALL_TASKS is the summary line at the end
							analyzerState = AnalyzerStateSearching
							continue
						}

						// Skip non-data lines
						if strings.Contains(line, "Name") && strings.Contains(line, "ID") {
							continue // Skip header
						}
						if strings.Contains(line, "----") {
							continue // Skip separator
						}
						if strings.TrimSpace(line) == "" {
							continue // Skip empty lines
						}

						// Count actual data lines
						taskLineCount++
						// Count coalitions vs subprocesses
						if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
							subprocessCount++
						} else {
							coalitionCount++
						}
					}
				}

				// Log the line counts immediately
				if taskSectionCount > 1 {
					fmt.Printf("⚠️ %d Task sections! ", taskSectionCount)
				}
				fmt.Printf("Task lines: %d (Coalitions: %d, Subprocesses: %d)",
					taskLineCount, coalitionCount, subprocessCount)

				// Count Chrome processes specifically
				chromeCount := 0
				for _, line := range strings.Split(sampleContent, "\n") {
					if strings.Contains(line, "Chrome") && !strings.Contains(line, "***") {
						chromeCount++
					}
				}
				if chromeCount > 0 {
					fmt.Printf(" [%d Chrome lines]", chromeCount)
				}

				// Parse using the persistent state (not a new one each time!)
				parser.ParsePowerMetricsOutput(sampleContent, persistentState)

				// Log what the parser found
				fmt.Printf(" → Parsed: %d processes, %d coalitions",
					len(persistentState.Processes), len(persistentState.Coalitions))

				// Check if any watched PIDs are in RecentlyExited
				if len(watchPIDs) > 0 && len(persistentState.RecentlyExited) > 0 {
					fmt.Printf(" [%d exited]", len(persistentState.RecentlyExited))
					for _, exitInfo := range persistentState.RecentlyExited {
						for _, pid := range exitInfo.PIDs {
							for _, watchPID := range watchPIDs {
								if pid == watchPID {
									fmt.Printf(" ⚠️PID %d marked as exited!", pid)
								}
							}
						}
					}
				}
				fmt.Printf("\n")

				// Build current process map
				currentProcesses := make(map[int]ProcessInfo)
				for _, proc := range persistentState.Processes {
					currentProcesses[proc.PID] = ProcessInfo{
						PID:        proc.PID,
						Name:       proc.Name,
						Coalition:  proc.CoalitionName,
						CPUPercent: proc.CPUPercent,
					}
					processSeenCount[proc.PID]++
				}

				// Always do detailed analysis when we have watched PIDs
				if len(watchPIDs) > 0 {
					fmt.Printf("  Watched PIDs in this sample: ")
					foundCount := 0
					for _, pid := range watchPIDs {
						if _, exists := currentProcesses[pid]; exists {
							fmt.Printf("%d✓ ", pid)
							foundCount++
						} else {
							fmt.Printf("%d✗ ", pid)
						}
					}
					fmt.Printf("(%d/%d found)\n", foundCount, len(watchPIDs))
				}


				// Skip all the detailed analysis - just update previous
				// Analyze delta but don't print unless it's for watched PIDs
				delta := analyzeDelta(previousProcesses, currentProcesses)

				// Check if any watched PIDs disappeared
				for _, p := range delta.Removed {
					for _, watchPID := range watchPIDs {
						if p.PID == watchPID {
							// Check if really dead
							psCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", p.PID), "-o", "pid=,comm=")
							psOutput, psErr := psCmd.Output()
							isAlive := psErr == nil && len(psOutput) > 0

							if isAlive {
								fmt.Printf("  ⚠️ PID %d (%s) disappeared but is still alive!\n", p.PID, p.Name)
							}
						}
					}
				}

				// Update previous for next iteration
				previousProcesses = currentProcesses

				// Exit after maxSamples
				if sampleCount >= maxSamples {
					fmt.Printf("\n\nCompleted %d samples. Exiting.\n", maxSamples)
					return
				}
			}
			// Start new sample
			outputBuffer.Reset()
			currentState = StateInSample
			// Don't include the header line itself in the buffer
			continue

		case strings.Contains(line, "*** Running tasks ***"):
			// Mark that we're in the task section
			if currentState == StateInSample {
				currentState = StateInTaskSection
			}

		case strings.HasPrefix(line, "***"):
			// Any other section marker - we're leaving the current section
			if currentState == StateInTaskSection {
				currentState = StateInSample
			}

		case strings.HasPrefix(line, "ALL_TASKS"):
			// End of task section
			if currentState == StateInTaskSection {
				currentState = StateInSample
			}
		}

		// Only buffer lines when we're actively in a sample
		if currentState != StateWaitingForSample {
			outputBuffer.WriteString(line)
			outputBuffer.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading powermetrics output: %v\n", err)
	}
}

func analyzeDelta(previous, current map[int]ProcessInfo) ProcessDelta {
	delta := ProcessDelta{
		Added:   []ProcessInfo{},
		Removed: []ProcessInfo{},
		Changed: []ProcessInfo{},
	}

	// Find added processes
	for pid, proc := range current {
		if _, exists := previous[pid]; !exists {
			delta.Added = append(delta.Added, proc)
		}
	}

	// Find removed processes
	for pid, proc := range previous {
		if _, exists := current[pid]; !exists {
			delta.Removed = append(delta.Removed, proc)
		}
	}

	// Sort for consistent output
	sort.Slice(delta.Added, func(i, j int) bool {
		return delta.Added[i].PID < delta.Added[j].PID
	})
	sort.Slice(delta.Removed, func(i, j int) bool {
		return delta.Removed[i].PID < delta.Removed[j].PID
	})

	return delta
}