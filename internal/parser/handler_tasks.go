package parser

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"powermetrics-tui/internal/models"
)

// RunningTasksHandler handles the running tasks section parsing
type RunningTasksHandler struct{}

func (h *RunningTasksHandler) Name() string {
	return "RunningTasks"
}

func (h *RunningTasksHandler) Enter(ctx *ParserContext) {
	// Finalize any previous coalition
	if ctx.CurrentCoalition != nil {
		ctx.NewCoalitions = append(ctx.NewCoalitions, *ctx.CurrentCoalition)
		ctx.CurrentCoalition = nil
	}

	// Reset process collections for this tasks section
	ctx.NewProcesses = make([]models.ProcessInfo, 0)
	ctx.NewCoalitions = make([]models.ProcessCoalition, 0)
	ctx.OrphanedSubprocesses = make([]models.ProcessInfo, 0)
}

func (h *RunningTasksHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	trimmed := strings.TrimSpace(line)

	// Check for transitions out of tasks section

	if IsEndOfTasks(line) {
		return StateInSample
	}

	if IsSection(line) && !IsRunningTasks(line) {
		return StateInSample
	}

	// Skip header lines and empty lines
	if IsTasksHeader(line) || strings.Contains(line, "----") || trimmed == "" {
		return StateRunningTasks
	}

	// Skip DEAD_TASKS entries
	if strings.Contains(line, "DEAD_TASKS") {
		return StateRunningTasks
	}

	// Parse process/coalition data
	if matches := processRegex.FindStringSubmatch(line); matches != nil {
		name := strings.TrimSpace(matches[1])
		id, _ := ParseInt(matches[2])
		cpuMs, _ := ParseFloat(matches[3])
		userPercent, _ := ParseFloat(matches[4])

		// Handle empty names - these are likely processes that died during sampling
		// Treat them as DEAD_TASKS
		if name == "" {
			// Give it a name indicating it's a dead process
			name = fmt.Sprintf("<dead-process-%d>", id)
			// Mark this as a dead/exited process immediately
		}

		// Convert CPU ms/s to percentage (approximate)
		cpuPercent := cpuMs / 10.0

		// Check if this is a subprocess (indented) or coalition (not indented)
		isSubprocess := IsIndented(line)

		if isSubprocess {
			return h.handleSubprocess(ctx, name, id, cpuPercent, userPercent)
		} else {
			return h.handleCoalition(ctx, name, id, cpuPercent, userPercent)
		}
	}

	return StateRunningTasks
}

func (h *RunningTasksHandler) handleSubprocess(ctx *ParserContext, name string, id int, cpuPercent, userPercent float64) ParserState {
	// Check if this is a dead process (marked with <dead-process- prefix)
	isDead := strings.HasPrefix(name, "<dead-process-")

	if isDead {
		// Track this as an exited process immediately
		processName := fmt.Sprintf("Unknown Process (PID %d)", id)
		h.trackExitedProcess(ctx, id, processName, time.Now())

		// Don't add to active processes list
		return StateRunningTasks
	}

	// Update process history
	if ctx.MetricsState.ProcessCPUHistory[id] == nil {
		ctx.MetricsState.ProcessCPUHistory[id] = make([]float64, 0, 10)
	}
	ctx.MetricsState.ProcessCPUHistory[id] = models.AddToHistory(ctx.MetricsState.ProcessCPUHistory[id], cpuPercent, 10)

	if ctx.MetricsState.ProcessMemHistory[id] == nil {
		ctx.MetricsState.ProcessMemHistory[id] = make([]float64, 0, 10)
	}
	ctx.MetricsState.ProcessMemHistory[id] = models.AddToHistory(ctx.MetricsState.ProcessMemHistory[id], userPercent, 10)

	// Create subprocess
	subprocess := models.ProcessInfo{
		PID:           id,
		Name:          name,
		CPUPercent:    cpuPercent,
		MemoryMB:      userPercent,
		DiskMB:        0,
		NetworkMB:     0,
		CPUHistory:    ctx.MetricsState.ProcessCPUHistory[id],
		MemoryHistory: ctx.MetricsState.ProcessMemHistory[id],
	}

	if ctx.CurrentCoalition != nil {
		// Add to current coalition
		subprocess.CoalitionName = ctx.CurrentCoalition.Name
		ctx.CurrentCoalition.Subprocesses = append(ctx.CurrentCoalition.Subprocesses, subprocess)
	} else {
		// Orphaned subprocess - collect for later assignment
		subprocess.CoalitionName = ""
		ctx.OrphanedSubprocesses = append(ctx.OrphanedSubprocesses, subprocess)
	}

	ctx.NewProcesses = append(ctx.NewProcesses, subprocess)
	return StateRunningTasks
}

func (h *RunningTasksHandler) handleCoalition(ctx *ParserContext, name string, id int, cpuPercent, userPercent float64) ParserState {
	// Save previous coalition if it exists
	if ctx.CurrentCoalition != nil {
		ctx.NewCoalitions = append(ctx.NewCoalitions, *ctx.CurrentCoalition)
	}

	// Check if this is a dead process (marked with <dead-process- prefix)
	if strings.HasPrefix(name, "<dead-process-") {
		// Track this as an exited process immediately
		processName := fmt.Sprintf("Unknown Process (PID %d)", id)
		h.trackExitedProcess(ctx, id, processName, time.Now())

		// Don't create a coalition for it
		ctx.CurrentCoalition = nil
		return StateRunningTasks
	}

	// Update coalition history
	if ctx.MetricsState.CoalitionCPUHistory[id] == nil {
		ctx.MetricsState.CoalitionCPUHistory[id] = make([]float64, 0, 10)
	}
	ctx.MetricsState.CoalitionCPUHistory[id] = models.AddToHistory(ctx.MetricsState.CoalitionCPUHistory[id], cpuPercent, 10)

	if ctx.MetricsState.CoalitionMemHistory[id] == nil {
		ctx.MetricsState.CoalitionMemHistory[id] = make([]float64, 0, 10)
	}
	ctx.MetricsState.CoalitionMemHistory[id] = models.AddToHistory(ctx.MetricsState.CoalitionMemHistory[id], userPercent, 10)

	// Track coalition name
	if ctx.MetricsState.CoalitionNames == nil {
		ctx.MetricsState.CoalitionNames = make(map[int]string)
	}
	ctx.MetricsState.CoalitionNames[id] = name

	// Create new coalition
	ctx.CurrentCoalition = &models.ProcessCoalition{
		CoalitionID:   id,
		Name:          name,
		CPUPercent:    cpuPercent,
		MemoryMB:      userPercent,
		DiskMB:        0,
		NetworkMB:     0,
		Subprocesses:  make([]models.ProcessInfo, 0),
		CPUHistory:    ctx.MetricsState.CoalitionCPUHistory[id],
		MemoryHistory: ctx.MetricsState.CoalitionMemHistory[id],
	}

	return StateRunningTasks
}

func (h *RunningTasksHandler) Exit(ctx *ParserContext) {
	// Finalize current coalition
	if ctx.CurrentCoalition != nil {
		ctx.NewCoalitions = append(ctx.NewCoalitions, *ctx.CurrentCoalition)
		ctx.CurrentCoalition = nil
	}

	// Handle orphaned subprocesses
	h.handleOrphanedSubprocesses(ctx)

	// Update process tracking
	h.updateProcessTracking(ctx)

	// Update the state with new data
	ctx.MetricsState.Processes = ctx.NewProcesses
	ctx.MetricsState.Coalitions = ctx.NewCoalitions

	// Update coalition names tracking
	for _, coalition := range ctx.NewCoalitions {
		if _, exists := ctx.MetricsState.CoalitionNames[coalition.CoalitionID]; !exists {
			ctx.MetricsState.CoalitionNames[coalition.CoalitionID] = coalition.Name
		}
	}
}

func (h *RunningTasksHandler) handleOrphanedSubprocesses(ctx *ParserContext) {
	// Create a map of coalition names for faster lookup
	coalitionMap := make(map[string]*models.ProcessCoalition)
	for i := range ctx.NewCoalitions {
		coalitionMap[ctx.NewCoalitions[i].Name] = &ctx.NewCoalitions[i]
	}

	// Process orphaned subprocesses
	for _, orphanedProc := range ctx.OrphanedSubprocesses {
		// Update process history for orphaned process
		if ctx.MetricsState.ProcessCPUHistory[orphanedProc.PID] == nil {
			ctx.MetricsState.ProcessCPUHistory[orphanedProc.PID] = make([]float64, 0, 10)
		}
		ctx.MetricsState.ProcessCPUHistory[orphanedProc.PID] = models.AddToHistory(
			ctx.MetricsState.ProcessCPUHistory[orphanedProc.PID], orphanedProc.CPUPercent, 10)

		if ctx.MetricsState.ProcessMemHistory[orphanedProc.PID] == nil {
			ctx.MetricsState.ProcessMemHistory[orphanedProc.PID] = make([]float64, 0, 10)
		}
		ctx.MetricsState.ProcessMemHistory[orphanedProc.PID] = models.AddToHistory(
			ctx.MetricsState.ProcessMemHistory[orphanedProc.PID], orphanedProc.MemoryMB, 10)

		orphanedProc.CPUHistory = ctx.MetricsState.ProcessCPUHistory[orphanedProc.PID]
		orphanedProc.MemoryHistory = ctx.MetricsState.ProcessMemHistory[orphanedProc.PID]

		// Mark as orphaned and add to process list
		orphanedProc.CoalitionName = "<orphaned>"
		ctx.NewProcesses = append(ctx.NewProcesses, orphanedProc)
	}
}

func (h *RunningTasksHandler) updateProcessTracking(ctx *ParserContext) {
	currentTime := time.Now()
	currentPIDs := make(map[int]bool)
	currentCoalitionIDs := make(map[int]bool)

	// Collect current PIDs and coalition IDs
	for _, proc := range ctx.NewProcesses {
		currentPIDs[proc.PID] = true
	}
	for _, coalition := range ctx.NewCoalitions {
		currentCoalitionIDs[coalition.CoalitionID] = true
	}

	// Initialize maps if needed
	if ctx.MetricsState.LastSeenPIDs == nil {
		ctx.MetricsState.LastSeenPIDs = make(map[int]time.Time)
	}
	if ctx.MetricsState.ProcessNames == nil {
		ctx.MetricsState.ProcessNames = make(map[int]string)
	}
	if ctx.MetricsState.CoalitionNames == nil {
		ctx.MetricsState.CoalitionNames = make(map[int]string)
	}

	// Check for processes that are no longer present
	for pid := range ctx.MetricsState.LastSeenPIDs {
		if !currentPIDs[pid] {
			processName := ctx.MetricsState.ProcessNames[pid]
			if processName == "" {
				// This is a ghost PID - a PID in LastSeenPIDs without a name
				// This can happen if the app was restarted with stale state
				delete(ctx.MetricsState.LastSeenPIDs, pid)
				delete(ctx.MetricsState.ProcessCPUHistory, pid)
				delete(ctx.MetricsState.ProcessMemHistory, pid)
				delete(ctx.MetricsState.ProcessNames, pid)  // Also clean this up just in case

				// Don't log this anymore since we know it happens on startup
				continue
			}

			// Verify if process is actually dead using ps
			psCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "pid=")
			if err := psCmd.Run(); err == nil {
				// Process is still alive - skip marking as exited
				continue
			}

			// Process is truly dead, track it as exited
			h.trackExitedProcess(ctx, pid, processName, currentTime)

			// Clean up tracking maps
			delete(ctx.MetricsState.LastSeenPIDs, pid)
			delete(ctx.MetricsState.ProcessCPUHistory, pid)
			delete(ctx.MetricsState.ProcessMemHistory, pid)
		}
	}

	// Clean up old exited processes (older than 5 minutes)
	var cleanedExited []models.ExitedProcessInfo
	for _, proc := range ctx.MetricsState.RecentlyExited {
		if currentTime.Sub(proc.LastExitTime) < 5*time.Minute {
			cleanedExited = append(cleanedExited, proc)
		}
	}
	ctx.MetricsState.RecentlyExited = cleanedExited

	// Update tracking maps with current processes
	for _, proc := range ctx.NewProcesses {
		ctx.MetricsState.LastSeenPIDs[proc.PID] = currentTime

		// ALWAYS set process name when we see a process to prevent ghost PIDs
		ctx.MetricsState.ProcessNames[proc.PID] = proc.Name

		// Remove from recently exited if it reappeared (false positive)
		h.removeFromRecentlyExited(ctx, proc.PID)
	}
}

func (h *RunningTasksHandler) trackExitedProcess(ctx *ParserContext, pid int, processName string, currentTime time.Time) {
	// Find existing exit info for this process name
	found := false
	for i := range ctx.MetricsState.RecentlyExited {
		if ctx.MetricsState.RecentlyExited[i].Name == processName {
			// Check if PID already exists
			pidExists := false
			for _, existingPID := range ctx.MetricsState.RecentlyExited[i].PIDs {
				if existingPID == pid {
					pidExists = true
					break
				}
			}
			if !pidExists {
				ctx.MetricsState.RecentlyExited[i].PIDs = append(ctx.MetricsState.RecentlyExited[i].PIDs, pid)
				ctx.MetricsState.RecentlyExited[i].Occurrences = len(ctx.MetricsState.RecentlyExited[i].PIDs)
			}
			ctx.MetricsState.RecentlyExited[i].LastExitTime = currentTime
			found = true
			break
		}
	}

	if !found {
		// For dead processes, we might not have seen them before
		// (they died during sampling), so LastSeenPIDs might not have them
		lastSeen, exists := ctx.MetricsState.LastSeenPIDs[pid]
		if !exists {
			// For processes we've never seen, use current time as first seen
			lastSeen = currentTime
		}

		exitedProc := models.ExitedProcessInfo{
			Name:          processName,
			PIDs:          []int{pid},
			Occurrences:   1,
			LastExitTime:  currentTime,
			FirstSeenTime: lastSeen,
		}
		ctx.MetricsState.RecentlyExited = append(ctx.MetricsState.RecentlyExited, exitedProc)
	}
}

func (h *RunningTasksHandler) removeFromRecentlyExited(ctx *ParserContext, pid int) {
	for i := 0; i < len(ctx.MetricsState.RecentlyExited); i++ {
		exitInfo := &ctx.MetricsState.RecentlyExited[i]
		for j := 0; j < len(exitInfo.PIDs); j++ {
			if exitInfo.PIDs[j] == pid {
				// Remove this PID from the list
				exitInfo.PIDs = append(exitInfo.PIDs[:j], exitInfo.PIDs[j+1:]...)
				exitInfo.Occurrences = len(exitInfo.PIDs)
				j-- // Adjust index after removal

				// If no more PIDs for this process name, remove the entire entry
				if len(exitInfo.PIDs) == 0 {
					ctx.MetricsState.RecentlyExited = append(ctx.MetricsState.RecentlyExited[:i], ctx.MetricsState.RecentlyExited[i+1:]...)
					i-- // Adjust index after removal
				}
				break
			}
		}
	}
}