package parser

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"powermetrics-tui/internal/models"
)

var (
	// New format: interrupts/sec
	ipiRateRegex    = regexp.MustCompile(`\|-> IPI:\s+([0-9.]+)\s+interrupts/sec`)
	timerRateRegex  = regexp.MustCompile(`\|-> TIMER:\s+([0-9.]+)\s+interrupts/sec`)
	totalRateRegex  = regexp.MustCompile(`Total IRQ:\s+([0-9.]+)\s+interrupts/sec`)

	// Per-CPU interrupt patterns
	cpuInterruptRegex = regexp.MustCompile(`^CPU (\d+):$`)

	// Old format: absolute counts (keeping for compatibility)
	ipiRegex         = regexp.MustCompile(`CPU (\d+) IPI:\s+(\d+)`)
	timerRegex       = regexp.MustCompile(`CPU (\d+) Timer:\s+(\d+)`)
	totalRegex       = regexp.MustCompile(`CPU (\d+) Total:\s+(\d+)`)
	cpuPowerRegex    = regexp.MustCompile(`(?:CPU Power|CPU Energy|Combined Power \(CPU\)):\s+([0-9.]+)\s*mW`)
	gpuPowerRegex    = regexp.MustCompile(`(?:GPU Power|GPU Energy|Combined Power \(GPU\)):\s+([0-9.]+)\s*mW`)
	anePowerRegex    = regexp.MustCompile(`(?:ANE Power|ANE Energy|Combined Power \(ANE\)):\s+([0-9.]+)\s*mW`)
	dramPowerRegex   = regexp.MustCompile(`(?:DRAM Power|DRAM Energy|Combined Power \(DRAM\)):\s+([0-9.]+)\s*mW`)
	systemPowerRegex = regexp.MustCompile(`(?:Combined Power|System Power|System Average).*?:\s+([0-9.]+)\s*(?:mW|Watts)`)
	// Thermal pattern - Updated for actual format: "Current pressure level: Nominal"
	thermalRegex     = regexp.MustCompile(`Current pressure level:\s+(\w+)`)
	tempRegex        = regexp.MustCompile(`([^:]+):\s+([0-9.]+)\s*(?:C|°C)`)
	batteryRegex     = regexp.MustCompile(`(?:Battery charge|State of Charge|percent_charge):\s+([0-9.]+)(?:%)?`)
	batteryStateRegex = regexp.MustCompile(`Battery state:\s+(\w+)`)

	// CPU frequency patterns (various formats)
	ecoreFreqRegex = regexp.MustCompile(`E-Cluster HW active frequency:\s+([0-9]+)\s*MHz`)
	pcoreFreqRegex = regexp.MustCompile(`P\d*-Cluster HW active frequency:\s+([0-9]+)\s*MHz`)  // Matches P0-Cluster, P1-Cluster, P-Cluster
	gpuFreqRegex   = regexp.MustCompile(`(?:GPU active frequency|GPU frequency):\s+([0-9]+)\s*MHz`)

	// Per-CPU frequency
	cpuFreqRegex   = regexp.MustCompile(`CPU (\d+) frequency:\s+([0-9]+)\s*MHz`)

	// Network patterns
	// Network patterns - Updated for actual format: "in: 70.77 packets/s, 69338.38 bytes/s"
	networkInRegex  = regexp.MustCompile(`in:\s+[0-9.]+\s+packets/s,\s+([0-9.]+)\s+bytes/s`)
	networkOutRegex = regexp.MustCompile(`out:\s+[0-9.]+\s+packets/s,\s+([0-9.]+)\s+bytes/s`)

	// Disk I/O patterns - Updated for actual format: "read: 0.00 ops/s 0.00 KBytes/s"
	diskReadRegex  = regexp.MustCompile(`read:\s+[0-9.]+\s+ops/s\s+([0-9.]+)\s+KBytes/s`)
	diskWriteRegex = regexp.MustCompile(`write:\s+[0-9.]+\s+ops/s\s+([0-9.]+)\s+KBytes/s`)

	// Memory patterns
	memoryUsedRegex  = regexp.MustCompile(`(?:Memory Used|Physical Memory Used):\s+([0-9.]+)\s*(?:MB|GB)`)
	memoryAvailRegex = regexp.MustCompile(`(?:Memory Available|Physical Memory Available):\s+([0-9.]+)\s*(?:MB|GB)`)
	swapUsedRegex    = regexp.MustCompile(`(?:Swap Used|VM Swap Used):\s+([0-9.]+)\s*(?:MB|GB)`)

	// Process patterns
	// Updated regex for "Running tasks" format
	// Format: Name (padded to ~35 chars) ID CPU_ms/s User% ...
	// Using a simpler approach: capture everything before the first number as name
	processRegex = regexp.MustCompile(`^(.+?)\s+(\d+)\s+([0-9.]+)\s+([0-9.]+)`)
)

// ParsePowerMetricsOutput parses the output from powermetrics command
func ParsePowerMetricsOutput(output string, state *models.MetricsState) {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	lines := strings.Split(output, "\n")

	// Reset accumulators
	ipiTotal := 0.0
	timerTotal := 0.0
	interrupts := 0.0
	currentCluster := "" // Track which cluster we're currently parsing

	// Don't reset frequency arrays here - we'll rebuild them from individual CPU data

	// Create new process and coalition lists, but preserve history
	newProcesses := []models.ProcessInfo{}
	newCoalitions := []models.ProcessCoalition{}
	orphanedSubprocesses := []models.ProcessInfo{} // Collect orphaned subprocesses for later assignment

	inProcessSection := false
	currentCoalition := (*models.ProcessCoalition)(nil) // Track current coalition being parsed
	currentCPU := ""  // Track which CPU we're parsing interrupts for

	// Initialize maps if needed
	if state.AllSeenCPUs == nil {
		state.AllSeenCPUs = make(map[string]bool)
	}
	if state.PerCPUInterrupts == nil {
		state.PerCPUInterrupts = make(map[string]float64)
	}
	if state.PerCPUIPIs == nil {
		state.PerCPUIPIs = make(map[string]float64)
	}
	if state.PerCPUTimers == nil {
		state.PerCPUTimers = make(map[string]float64)
	}

	// Reset all known CPUs to 0 (don't remove them)
	for cpu := range state.AllSeenCPUs {
		state.PerCPUInterrupts[cpu] = 0
		state.PerCPUIPIs[cpu] = 0
		state.PerCPUTimers[cpu] = 0
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// Check for section boundaries (lines starting with ***)
		if strings.HasPrefix(line, "***") {
			// If we were in the process section, finish it up
			if inProcessSection {
				// Save the current coalition if any
				if currentCoalition != nil {
					newCoalitions = append(newCoalitions, *currentCoalition)
					currentCoalition = nil
				}
				inProcessSection = false
			}

			// Check what new section we're entering
			if strings.Contains(line, "*** Running tasks ***") {
				// Only process the first tasks section to avoid duplicates
				if len(newProcesses) == 0 && len(newCoalitions) == 0 {
					inProcessSection = true
				}
			}
			// Could add other section checks here if needed
			continue
		}

		// Skip the header line in process section
		if inProcessSection && strings.Contains(line, "Name") && strings.Contains(line, "ID") {
			continue
		}

		// Check for CPU interrupt header (e.g., "CPU 0:")
		if matches := cpuInterruptRegex.FindStringSubmatch(line); matches != nil {
			currentCPU = "CPU" + matches[1]
			// Mark this CPU as seen
			state.AllSeenCPUs[currentCPU] = true
			continue
		}

		// Parse interrupts - new format (interrupts/sec)
		if matches := ipiRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				ipiTotal += val
				// If we have a current CPU, track per-CPU data
				if currentCPU != "" {
					state.PerCPUIPIs[currentCPU] = val
					state.AllSeenCPUs[currentCPU] = true
				}
			}
		}

		if matches := timerRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				timerTotal += val
				// If we have a current CPU, track per-CPU data
				if currentCPU != "" {
					state.PerCPUTimers[currentCPU] = val
					state.AllSeenCPUs[currentCPU] = true
				}
			}
		}

		if matches := totalRateRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				interrupts += val
				// If we have a current CPU, track per-CPU data
				if currentCPU != "" {
					state.PerCPUInterrupts[currentCPU] = val
					state.AllSeenCPUs[currentCPU] = true
				}
			}
		}

		// Parse interrupts - old format (absolute counts)
		if matches := ipiRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				ipiTotal += float64(val)
			}
		}

		if matches := timerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				timerTotal += float64(val)
			}
		}

		if matches := totalRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[2]); err == nil {
				interrupts += float64(val)
			}
		}

		// Parse power metrics
		if matches := cpuPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.CPUPower = val
			}
		}

		if matches := gpuPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.GPUPower = val
			}
		}

		if matches := anePowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.ANEPower = val
			}
		}

		if matches := dramPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DRAMPower = val
			}
		}

		if matches := systemPowerRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// Convert watts to milliwatts if needed
				if strings.Contains(line, "Watts") {
					val *= 1000
				}
				state.SystemPower = val
			}
		}

		// Track cluster headers to determine CPU membership
		if strings.Contains(line, "E-Cluster Online:") {
			currentCluster = "E"
		} else if strings.Contains(line, "P0-Cluster Online:") || strings.Contains(line, "P1-Cluster Online:") {
			currentCluster = "P"
		} else if strings.Contains(line, "P-Cluster Online:") { // Some systems might just have "P-Cluster"
			currentCluster = "P"
		}

		// Parse individual CPU frequencies and store with cluster context
		if matches := cpuFreqRegex.FindStringSubmatch(line); matches != nil {
			if cpuNum, err := strconv.Atoi(matches[1]); err == nil {
				if freq, err := strconv.Atoi(matches[2]); err == nil {
					// Store individual CPU frequency in temporary map
					if state.AllCpuFreq == nil {
						state.AllCpuFreq = make(map[int]int)
					}
					state.AllCpuFreq[cpuNum] = freq

					// Store cluster membership info in a more structured way
					// We'll use this in organizeCPUFrequencies
					if currentCluster != "" {
						// Mark which cluster this CPU belongs to
						if state.AllSeenCPUs == nil {
							state.AllSeenCPUs = make(map[string]bool)
						}
						// Store cluster membership
						clusterKey := fmt.Sprintf("%s-CPU%d", currentCluster, cpuNum)
						state.AllSeenCPUs[clusterKey] = true
					}
				}
			}
		}

		if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.Atoi(matches[1]); err == nil {
				state.GPUFreq = val
			}
		}

		// Parse network (bytes/s to KB/s)
		if matches := networkInRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.NetworkIn = val / 1024.0 // Convert bytes/s to KB/s
			}
		}

		if matches := networkOutRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.NetworkOut = val / 1024.0 // Convert bytes/s to KB/s
			}
		}

		// Parse disk I/O (KB/s to MB/s)
		if matches := diskReadRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DiskRead = val / 1024.0 // Convert KB/s to MB/s
			}
		}

		if matches := diskWriteRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.DiskWrite = val / 1024.0 // Convert KB/s to MB/s
			}
		}

		// Parse battery
		if matches := batteryRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.BatteryCharge = val
			}
		}

		if matches := batteryStateRegex.FindStringSubmatch(line); matches != nil {
			state.BatteryState = matches[1]
		}

		// Parse thermal
		if matches := thermalRegex.FindStringSubmatch(line); matches != nil {
			state.ThermalPressure = matches[1]
		}

		// Parse temperatures
		if matches := tempRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[2], 64); err == nil {
				state.Temperature[matches[1]] = val
			}
		}

		// Parse memory
		if matches := memoryUsedRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.MemoryUsed = convertToMB(val, line)
			}
		}

		if matches := memoryAvailRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.MemoryAvailable = convertToMB(val, line)
			}
		}

		if matches := swapUsedRegex.FindStringSubmatch(line); matches != nil {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				state.SwapUsed = convertToMB(val, line)
			}
		}

		// Parse processes from "Running tasks" section with coalition/subprocess hierarchy
		if inProcessSection {
			// Skip empty lines
			if line == "" {
				continue
			}

			// Check for end of tasks section
			if strings.HasPrefix(line, "ALL_TASKS") {
				// Save the current coalition before ending
				if currentCoalition != nil {
					newCoalitions = append(newCoalitions, *currentCoalition)
					currentCoalition = nil
				}
				inProcessSection = false
				continue
			}

			// Skip DEAD_TASKS entries - they have special format and aren't real processes
			if strings.Contains(rawLine, "DEAD_TASKS") {
				continue
			}

			if matches := processRegex.FindStringSubmatch(rawLine); matches != nil {
				name := strings.TrimSpace(matches[1])
				id, _ := strconv.Atoi(matches[2])
				cpuMs, _ := strconv.ParseFloat(matches[3], 64)
				userPercent, _ := strconv.ParseFloat(matches[4], 64)


				// Convert CPU ms/s to percentage (approximate)
				cpuPercent := cpuMs / 10.0

				// Check indentation using rawLine to determine hierarchy
				// Subprocess: starts with whitespace (spaces or tabs)
				// Coalition: no leading whitespace
				isSubprocess := strings.HasPrefix(rawLine, " ") || strings.HasPrefix(rawLine, "\t")
				if isSubprocess {
					// This is a subprocess (indented with spaces or tabs)
					if currentCoalition == nil {
						// Collect orphaned subprocess for later assignment
						orphanedProcess := models.ProcessInfo{
							PID:           id,
							Name:          name,
							CoalitionName: "", // Will be filled later
							CPUPercent:    cpuPercent,
							MemoryMB:      userPercent,
							DiskMB:        0,
							NetworkMB:     0,
							CPUHistory:    make([]float64, 0),
							MemoryHistory: make([]float64, 0),
						}
						orphanedSubprocesses = append(orphanedSubprocesses, orphanedProcess)
						continue
					}

					// Update process history
					if state.ProcessCPUHistory[id] == nil {
						state.ProcessCPUHistory[id] = make([]float64, 0, 10)
					}
					state.ProcessCPUHistory[id] = models.AddToHistory(state.ProcessCPUHistory[id], cpuPercent, 10)

					if state.ProcessMemHistory[id] == nil {
						state.ProcessMemHistory[id] = make([]float64, 0, 10)
					}
					state.ProcessMemHistory[id] = models.AddToHistory(state.ProcessMemHistory[id], userPercent, 10)

					// Create subprocess and add to current coalition
					subprocess := models.ProcessInfo{
						PID:           id,
						Name:          name,
						CoalitionName: currentCoalition.Name,
						CPUPercent:    cpuPercent,
						MemoryMB:      userPercent,
						DiskMB:        0,
						NetworkMB:     0,
						CPUHistory:    state.ProcessCPUHistory[id],
						MemoryHistory: state.ProcessMemHistory[id],
					}
					currentCoalition.Subprocesses = append(currentCoalition.Subprocesses, subprocess)
					newProcesses = append(newProcesses, subprocess)

				} else {
					// This is a coalition (no leading whitespace)
					// Save previous coalition if it exists
					if currentCoalition != nil {
						newCoalitions = append(newCoalitions, *currentCoalition)
					}

					// Update coalition history
					if state.CoalitionCPUHistory[id] == nil {
						state.CoalitionCPUHistory[id] = make([]float64, 0, 10)
					}
					state.CoalitionCPUHistory[id] = models.AddToHistory(state.CoalitionCPUHistory[id], cpuPercent, 10)

					if state.CoalitionMemHistory[id] == nil {
						state.CoalitionMemHistory[id] = make([]float64, 0, 10)
					}
					state.CoalitionMemHistory[id] = models.AddToHistory(state.CoalitionMemHistory[id], userPercent, 10)

					// Create new coalition
					currentCoalition = &models.ProcessCoalition{
						CoalitionID:   id,
						Name:          name,
						CPUPercent:    cpuPercent,
						MemoryMB:      userPercent,
						DiskMB:        0,
						NetworkMB:     0,
						Subprocesses:  make([]models.ProcessInfo, 0),
						CPUHistory:    state.CoalitionCPUHistory[id],
						MemoryHistory: state.CoalitionMemHistory[id],
					}
				}
			}
		}
	}

	// Save final coalition if it exists
	if currentCoalition != nil {
		newCoalitions = append(newCoalitions, *currentCoalition)
	}

	// Try to assign orphaned subprocesses to coalitions
	// Create a map of coalition names to indices for faster lookup
	coalitionMap := make(map[string]*models.ProcessCoalition)
	for i := range newCoalitions {
		coalitionMap[newCoalitions[i].Name] = &newCoalitions[i]
	}

	// Assign orphaned subprocesses to coalitions or treat them as standalone
	for _, orphanedProc := range orphanedSubprocesses {
		// Update process history for orphaned process
		if state.ProcessCPUHistory[orphanedProc.PID] == nil {
			state.ProcessCPUHistory[orphanedProc.PID] = make([]float64, 0, 10)
		}
		state.ProcessCPUHistory[orphanedProc.PID] = models.AddToHistory(state.ProcessCPUHistory[orphanedProc.PID], orphanedProc.CPUPercent, 10)

		if state.ProcessMemHistory[orphanedProc.PID] == nil {
			state.ProcessMemHistory[orphanedProc.PID] = make([]float64, 0, 10)
		}
		state.ProcessMemHistory[orphanedProc.PID] = models.AddToHistory(state.ProcessMemHistory[orphanedProc.PID], orphanedProc.MemoryMB, 10)

		orphanedProc.CPUHistory = state.ProcessCPUHistory[orphanedProc.PID]
		orphanedProc.MemoryHistory = state.ProcessMemHistory[orphanedProc.PID]

		// Even if orphaned, add it to the process list
		// Set coalition name to indicate it's orphaned
		orphanedProc.CoalitionName = "<orphaned>"
		newProcesses = append(newProcesses, orphanedProc)
	}

	// Track recently exited processes - only track actual subprocesses, not coalitions
	currentTime := time.Now()
	currentPIDs := make(map[int]bool)
	currentCoalitionIDs := make(map[int]bool)

	// Collect all current PIDs (only subprocesses) and coalition IDs
	for _, proc := range newProcesses {
		currentPIDs[proc.PID] = true
	}
	for _, coalition := range newCoalitions {
		currentCoalitionIDs[coalition.CoalitionID] = true
	}


	// Check for processes that are no longer present
	// IMPORTANT: Only track SUBPROCESS PIDs as exited, not coalition IDs
	for pid := range state.LastSeenPIDs {
		if !currentPIDs[pid] {
			// Double-check this isn't actually a coalition ID that moved
			if currentCoalitionIDs[pid] {
				// This PID is now a coalition ID, clean up old tracking
				delete(state.LastSeenPIDs, pid)
				continue
			}

			processName := state.ProcessNames[pid]
			if processName == "" {
				// This PID has no name - it was never actually parsed!
				// This shouldn't happen, but if it does, clean it up
				// Check if this is actually a coalition ID first
				if _, isCoalition := state.CoalitionNames[pid]; isCoalition {
					// This is a coalition ID that was mistakenly tracked as a PID
					delete(state.LastSeenPIDs, pid)
					continue
				}

				// This is a ghost PID - it's in LastSeenPIDs but was never parsed
				// Log it for debugging then clean it up
				if debugFile, err := os.OpenFile("/tmp/powermetrics-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					debugFile.WriteString(fmt.Sprintf("[%s] GHOST PID %d in LastSeenPIDs but never parsed (no name)\n",
						time.Now().Format("15:04:05"), pid))
					debugFile.Close()
				}

				// Clean up this ghost PID
				delete(state.LastSeenPIDs, pid)
				continue // Skip tracking it as exited since it never existed
			}

			// IMPORTANT: Chrome processes often disappear intermittently from powermetrics
			// but are still alive. We should verify if a process is truly dead before
			// marking it as exited. For now, we'll still track it but this is a known issue.

			// Check if process is actually dead using ps (expensive but accurate)
			psCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "pid=")
			if err := psCmd.Run(); err == nil {
				// Process is still alive - this is a false positive!
				// Skip marking it as exited
				continue
			}

			// Process is truly dead, track it as exited
			found := false
			for i := range state.RecentlyExited {
				if state.RecentlyExited[i].Name == processName {
					// Don't add duplicate PIDs
					pidExists := false
					for _, existingPID := range state.RecentlyExited[i].PIDs {
						if existingPID == pid {
							pidExists = true
							break
						}
					}
					if !pidExists {
						state.RecentlyExited[i].PIDs = append(state.RecentlyExited[i].PIDs, pid)
						// Only increment occurrences once per unique PID exit, not every sample
						// This was the bug - we were counting every sample as a new occurrence!
						state.RecentlyExited[i].Occurrences = len(state.RecentlyExited[i].PIDs)
					}
					state.RecentlyExited[i].LastExitTime = currentTime
					found = true
					break
				}
			}

			if !found {
				if lastSeen, exists := state.LastSeenPIDs[pid]; exists {
					exitedProc := models.ExitedProcessInfo{
						Name:          processName,
						PIDs:          []int{pid},
						Occurrences:   1, // One unique PID has exited
						LastExitTime:  currentTime,
						FirstSeenTime: lastSeen,
					}
					state.RecentlyExited = append(state.RecentlyExited, exitedProc)
				}
			}

			// Clean up tracking maps
			delete(state.LastSeenPIDs, pid)
			delete(state.ProcessCPUHistory, pid)
			delete(state.ProcessMemHistory, pid)
			// DO NOT delete from ProcessNames - we need to keep the name history
			// for debugging and proper exit tracking
		}
	}

	// Clean up old exited processes (older than 5 minutes)
	var cleanedExited []models.ExitedProcessInfo
	for _, proc := range state.RecentlyExited {
		if currentTime.Sub(proc.LastExitTime) < 5*time.Minute {
			cleanedExited = append(cleanedExited, proc)
		}
	}
	state.RecentlyExited = cleanedExited

	// Update tracking maps with current processes and coalitions (after exit detection)
	for _, proc := range newProcesses {
		// Only track subprocess PIDs, not coalition IDs
		state.LastSeenPIDs[proc.PID] = currentTime
		// Only set the name if we haven't seen this PID before
		// PIDs are unique and their names don't change during their lifetime
		if existingName, exists := state.ProcessNames[proc.PID]; !exists {
			state.ProcessNames[proc.PID] = proc.Name
		} else if existingName != proc.Name {
			// This should never happen - PIDs don't change names
			// Log it for debugging if it does
			if debugFile, err := os.OpenFile("/tmp/powermetrics-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				debugFile.WriteString(fmt.Sprintf("[%s] WARNING: PID %d name changed from '%s' to '%s'\n",
					time.Now().Format("15:04:05"), proc.PID, existingName, proc.Name))
				debugFile.Close()
			}
		}

		// IMPORTANT: If this process was marked as exited but has reappeared,
		// remove it from the RecentlyExited list (it was a false positive)
		for i := 0; i < len(state.RecentlyExited); i++ {
			exitInfo := &state.RecentlyExited[i]
			// Check if this PID is in the exited list
			for j := 0; j < len(exitInfo.PIDs); j++ {
				if exitInfo.PIDs[j] == proc.PID {
					// Remove this PID from the list
					exitInfo.PIDs = append(exitInfo.PIDs[:j], exitInfo.PIDs[j+1:]...)
					exitInfo.Occurrences = len(exitInfo.PIDs)
					j-- // Adjust index after removal

					// If no more PIDs for this process name, remove the entire entry
					if len(exitInfo.PIDs) == 0 {
						state.RecentlyExited = append(state.RecentlyExited[:i], state.RecentlyExited[i+1:]...)
						i-- // Adjust index after removal
					}
					break
				}
			}
		}
	}

	// Update coalition names (but don't track them as processes for exit detection)
	for _, coalition := range newCoalitions {
		// Track coalition names in the SEPARATE CoalitionNames map, not ProcessNames
		// Only set the name if we haven't seen this coalition ID before
		// Coalition IDs are unique and their names don't change during their lifetime
		if existingName, exists := state.CoalitionNames[coalition.CoalitionID]; !exists {
			state.CoalitionNames[coalition.CoalitionID] = coalition.Name
		} else if existingName != coalition.Name {
			// This should never happen - coalition IDs don't change names
			// Log it for debugging if it does
			if debugFile, err := os.OpenFile("/tmp/powermetrics-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				debugFile.WriteString(fmt.Sprintf("[%s] WARNING: Coalition ID %d name changed from '%s' to '%s'\n",
					time.Now().Format("15:04:05"), coalition.CoalitionID, existingName, coalition.Name))
				debugFile.Close()
			}
		}
	}

	// Update the processes and coalitions lists with the new data
	state.Processes = newProcesses
	state.Coalitions = newCoalitions

	// Organize CPU frequencies based on what we detected
	organizeCPUFrequencies(state)

	// CPU frequency history is now updated in organizeCPUFrequencies

	// Update interrupt totals (convert float rates to int for display)
	if ipiTotal > 0 {
		state.IPICount = int(ipiTotal)
	}
	if timerTotal > 0 {
		state.TimerCount = int(timerTotal)
	}
	if interrupts > 0 {
		state.TotalInterrupts = int(interrupts)
	}

	// Update per-CPU interrupt history for all known CPUs
	// Since we reset all CPUs to 0 at the start, this will include zeros for missing CPUs
	for cpu := range state.AllSeenCPUs {
		total := state.PerCPUInterrupts[cpu] // Will be 0 if CPU wasn't in this sample
		if state.PerCPUInterruptHistory[cpu] == nil {
			state.PerCPUInterruptHistory[cpu] = make([]float64, 0, 30)
		}
		state.PerCPUInterruptHistory[cpu] = models.AddToHistory(state.PerCPUInterruptHistory[cpu], total, 30)
	}

	// Update history
	state.History.IPIHistory = models.AddToIntHistory(state.History.IPIHistory, state.IPICount, state.History.MaxHistory)
	state.History.TimerHistory = models.AddToIntHistory(state.History.TimerHistory, state.TimerCount, state.History.MaxHistory)
	state.History.TotalHistory = models.AddToIntHistory(state.History.TotalHistory, state.TotalInterrupts, state.History.MaxHistory)
	state.History.CPUPowerHistory = models.AddToHistory(state.History.CPUPowerHistory, state.CPUPower, state.History.MaxHistory)
	state.History.GPUPowerHistory = models.AddToHistory(state.History.GPUPowerHistory, state.GPUPower, state.History.MaxHistory)
	state.History.SystemHistory = models.AddToHistory(state.History.SystemHistory, state.SystemPower, state.History.MaxHistory)
	state.History.NetworkInHistory = models.AddToHistory(state.History.NetworkInHistory, state.NetworkIn, state.History.MaxHistory)
	state.History.NetworkOutHistory = models.AddToHistory(state.History.NetworkOutHistory, state.NetworkOut, state.History.MaxHistory)
	state.History.DiskReadHistory = models.AddToHistory(state.History.DiskReadHistory, state.DiskRead, state.History.MaxHistory)
	state.History.DiskWriteHistory = models.AddToHistory(state.History.DiskWriteHistory, state.DiskWrite, state.History.MaxHistory)
	state.History.BatteryHistory = models.AddToHistory(state.History.BatteryHistory, state.BatteryCharge, state.History.MaxHistory)
	state.History.MemoryHistory = models.AddToHistory(state.History.MemoryHistory, state.MemoryUsed, state.History.MaxHistory)

	// Update average temperature
	if len(state.Temperature) > 0 {
		var avgTemp float64
		for _, temp := range state.Temperature {
			avgTemp += temp
		}
		avgTemp /= float64(len(state.Temperature))
		state.History.TempHistory = models.AddToHistory(state.History.TempHistory, avgTemp, state.History.MaxHistory)
	}
}

func convertToMB(value float64, line string) float64 {
	if strings.Contains(line, "KB") {
		return value / 1024
	} else if strings.Contains(line, "GB") {
		return value * 1024
	} else if strings.Contains(line, "bytes") {
		return value / (1024 * 1024)
	}
	return value
}

// organizeCPUFrequencies categorizes CPUs based on cluster information parsed from powermetrics
func organizeCPUFrequencies(state *models.MetricsState) {
	if state.AllCpuFreq == nil || len(state.AllCpuFreq) == 0 {
		return
	}

	// Separate CPUs based on cluster membership information
	ecoreCPUs := make([]int, 0)
	pcoreCPUs := make([]int, 0)

	// Check if we have cluster membership info
	hasClusterInfo := false
	if state.AllSeenCPUs != nil {
		for key := range state.AllSeenCPUs {
			if strings.HasPrefix(key, "E-CPU") || strings.HasPrefix(key, "P-CPU") {
				hasClusterInfo = true
				break
			}
		}
	}

	if hasClusterInfo {
		// Use cluster membership information from parsing
		for key := range state.AllSeenCPUs {
			if strings.HasPrefix(key, "E-CPU") {
				// Extract CPU number from "E-CPU0", "E-CPU1", etc.
				cpuNumStr := strings.TrimPrefix(key, "E-CPU")
				if cpuNum, err := strconv.Atoi(cpuNumStr); err == nil {
					ecoreCPUs = append(ecoreCPUs, cpuNum)
				}
			} else if strings.HasPrefix(key, "P-CPU") {
				// Extract CPU number from "P-CPU2", "P-CPU3", etc.
				cpuNumStr := strings.TrimPrefix(key, "P-CPU")
				if cpuNum, err := strconv.Atoi(cpuNumStr); err == nil {
					pcoreCPUs = append(pcoreCPUs, cpuNum)
				}
			}
		}

		// Sort CPU lists for consistent display
		for i := 0; i < len(ecoreCPUs); i++ {
			for j := i + 1; j < len(ecoreCPUs); j++ {
				if ecoreCPUs[i] > ecoreCPUs[j] {
					ecoreCPUs[i], ecoreCPUs[j] = ecoreCPUs[j], ecoreCPUs[i]
				}
			}
		}
		for i := 0; i < len(pcoreCPUs); i++ {
			for j := i + 1; j < len(pcoreCPUs); j++ {
				if pcoreCPUs[i] > pcoreCPUs[j] {
					pcoreCPUs[i], pcoreCPUs[j] = pcoreCPUs[j], pcoreCPUs[i]
				}
			}
		}
	} else {
		// Fallback: No cluster info, categorize all CPUs as P-cores (Intel Mac)
		for cpuID := range state.AllCpuFreq {
			pcoreCPUs = append(pcoreCPUs, cpuID)
		}
		// Sort for consistency
		for i := 0; i < len(pcoreCPUs); i++ {
			for j := i + 1; j < len(pcoreCPUs); j++ {
				if pcoreCPUs[i] > pcoreCPUs[j] {
					pcoreCPUs[i], pcoreCPUs[j] = pcoreCPUs[j], pcoreCPUs[i]
				}
			}
		}
	}

	// Build E-core frequencies
	newECores := make([]int, 0, len(ecoreCPUs))
	for i, cpuID := range ecoreCPUs {
		freq := 0
		if f, exists := state.AllCpuFreq[cpuID]; exists {
			freq = f
		}
		newECores = append(newECores, freq)

		// Update history
		if state.ECoreFreqHistory[i] == nil {
			state.ECoreFreqHistory[i] = make([]float64, 0, 30)
		}
		state.ECoreFreqHistory[i] = models.AddToHistory(
			state.ECoreFreqHistory[i], float64(freq), 30)
	}

	// Build P-core frequencies
	newPCores := make([]int, 0, len(pcoreCPUs))
	for i, cpuID := range pcoreCPUs {
		freq := 0
		if f, exists := state.AllCpuFreq[cpuID]; exists {
			freq = f
		}
		newPCores = append(newPCores, freq)

		// Update history
		if state.PCoreFreqHistory[i] == nil {
			state.PCoreFreqHistory[i] = make([]float64, 0, 30)
		}
		state.PCoreFreqHistory[i] = models.AddToHistory(
			state.PCoreFreqHistory[i], float64(freq), 30)
	}

	// Update state
	state.ECoreFreq = newECores
	state.PCoreFreq = newPCores

	// Update max cores seen
	if len(state.ECoreFreq) > state.MaxECores {
		state.MaxECores = len(state.ECoreFreq)
	}
	if len(state.PCoreFreq) > state.MaxPCores {
		state.MaxPCores = len(state.PCoreFreq)
	}

	// Keep the AllCpuFreq for reference/debugging
}