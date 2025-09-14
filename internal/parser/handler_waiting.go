package parser

import (
	"powermetrics-tui/internal/models"
)

// WaitingForSampleHandler handles the initial state waiting for a new sample
type WaitingForSampleHandler struct{}

func (h *WaitingForSampleHandler) Name() string {
	return "WaitingForSample"
}

func (h *WaitingForSampleHandler) Enter(ctx *ParserContext) {
	// Reset accumulators for new sample
	ctx.IPITotal = 0
	ctx.TimerTotal = 0
	ctx.InterruptsTotal = 0
	ctx.CurrentCPU = ""
	ctx.CurrentCluster = ""
}

func (h *WaitingForSampleHandler) ProcessLine(ctx *ParserContext, line string) ParserState {
	if IsNewSample(line) {
		ctx.SampleCount++
		return StateInSample
	}
	return StateWaitingForSample
}

func (h *WaitingForSampleHandler) Exit(ctx *ParserContext) {
	// Prepare for new sample parsing
	ctx.NewProcesses = make([]models.ProcessInfo, 0)
	ctx.NewCoalitions = make([]models.ProcessCoalition, 0)
	ctx.OrphanedSubprocesses = make([]models.ProcessInfo, 0)
	ctx.CurrentCoalition = nil

	// Reset per-CPU data for this sample (but keep seen CPUs)
	if ctx.MetricsState.AllSeenCPUs != nil {
		for cpu := range ctx.MetricsState.AllSeenCPUs {
			ctx.MetricsState.PerCPUInterrupts[cpu] = 0
			ctx.MetricsState.PerCPUIPIs[cpu] = 0
			ctx.MetricsState.PerCPUTimers[cpu] = 0
		}
	}
}